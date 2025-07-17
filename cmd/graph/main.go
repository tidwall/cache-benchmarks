package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tidwall/gjson"
)

var json string
var clients int
var coperations int
var caches []string   // caches
var threadz []int     // threads
var versions []string // versions for caches
var dir string = "results"
var kind string = "median"
var which string = "get"
var pipeline int = 1
var ppp string = "99"
var bench string = "throughput"
var force bool = false
var scale string = "logarithmic"
var scase string = ""

const fontfamily string = "Futura"

var colors = []string{
	"#ff7f0e", "#d62728", "#1f77b4", "#8c564b", "#2ca02c", "#e64098",
}

func main() {

	// fmt.Printf("%s\n", os.Args)
	flag.IntVar(&pipeline, "pipeline", pipeline, "1,10,25,50")
	flag.StringVar(&ppp, "percentile", ppp, "99th: avg,min,max,50,90,99,999,9999")
	flag.StringVar(&which, "which", which, "set,get")
	flag.StringVar(&bench, "bench", bench, "throughput,latency,cpucycles")
	flag.StringVar(&kind, "kind", kind, "median,average,best,worst")
	flag.StringVar(&dir, "dir", dir, "Results directory")
	flag.BoolVar(&force, "force", force, "Force write (overwrite)")
	flag.StringVar(&scale, "scale", scale, "logarithmic,linear")
	flag.StringVar(&scase, "scase", scase, "special case: 1=remove garnet (thread 1)")
	flag.Parse()

	data, err := os.ReadFile(dir + "/output.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	json = string(data)

	switch bench {
	case "throughput", "cpucycles", "latency":
	default:
		fmt.Printf("invalid flag --bench='%s'\n", bench)
		os.Exit(1)
	}
	switch scale {
	case "logarithmic", "linear":
	default:
		fmt.Printf("invalid flag --scale='%s'\n", scale)
		os.Exit(1)
	}

	// Get the name of all cache programs and the versions
	cm := map[string]bool{}
	gjson.Get(json, "#.data.info.cache").ForEach(
		func(_, name gjson.Result) bool {
			if !cm[name.String()] {
				caches = append(caches, name.String())
				cm[name.String()] = true
				versions = append(versions, gjson.Get(json,
					"#(data.info.cache="+name.String()+
						").data.info.version").String())
			}
			return true
		},
	)

	tm := map[int]bool{}
	gjson.Get(json, "#.data.info.threads").ForEach(
		func(_, name gjson.Result) bool {
			if !tm[int(name.Int())] {
				threadz = append(threadz, int(name.Int()))
				tm[int(name.Int())] = true
			}
			return true
		},
	)
	sort.Ints(threadz)

	clients = int(gjson.Get(json, "0.data.info.connections").Int())
	coperations = int(gjson.Get(json, "0.data.info.operations").Int())

	switch bench {
	case "throughput":
		graphThroughput()
	case "latency":
		graphLatency()
	case "cpucycles":
		graphCPUCycles()
	}
}

func graphCPUCycles() {
	filename := "graph_cpucycles-pipeline_" + fmt.Sprint(pipeline) +
		"-kind_" + kind + "-scale_" + scale
	if scase != "" {
		filename += "-case_" + scase
	}
	filename += ".png"
	filename = filepath.Join(dir, "graphs", filename)
	if !force {
		_, err := os.Stat(filename)
		if err == nil {
			// fmt.Printf("File exists\n")
			os.Exit(0)
		}
	}

	title := fmt.Sprintf("GET+SET - %d Clients - %d Ops - Pipeline %d",
		clients, coperations*2, pipeline)

	ytitle := "CPU Cycles (cycles/op)"

	res := gjson.Get(json, ""+
		"#(data.perf.cycles!='')#|"+
		"#(data.info.kind="+fmt.Sprint(kind)+")#|"+
		"#(data.info.pipeline="+fmt.Sprint(pipeline)+")#")

	// Colors
	var outColors string
	for i, cache := range caches {
		outColors += fmt.Sprintf("    \"%s\": \"%s\",\n", cache, colors[i])
	}

	// Threads
	var outXSeries string
	for i, threads := range threadz {
		if i > 0 {
			outXSeries += ", "
		}
		outXSeries += fmt.Sprintf("%d", threads)
	}

	// Benchmarks
	var outData string
	for _, cache := range caches {
		outData += fmt.Sprintf("    \"%s\": [", cache)
		res1 := res.Get("#(data.info.cache=" + cache + ")#")
		for i, threads := range threadz {
			if i > 0 {
				outData += ", "
			}
			res2 := res1.Get("#(data.info.threads=" + fmt.Sprint(threads) + ")")
			point := fmt.Sprintf("%.0f",
				res2.Get("data.perf.cycles").Float()/float64(coperations*2))
			if scase == "1" && cache == "garnet" && threads == 1 {
				point = "0"
			}
			outData += point
		}
		outData += "],\n"
	}
	drawGraph(title, ytitle, filename, outXSeries, outData, outColors, scale)
}

func graphLatency() {
	label := ""
	switch which {
	case "get":
		which = "gets"
		label = "GET"
	case "set":
		which = "sets"
		label = "SET"
	default:
		fmt.Printf("invalid flag --which='%s'\n", which)
		os.Exit(1)
	}
	pwhich := ""
	plabel := ""
	switch ppp {
	case "min":
		pwhich = "min"
		plabel = "MIN"
	case "max":
		pwhich = "max"
		plabel = "MAX"
	case "avg":
		pwhich = "avg"
		plabel = "AVG"
	case "50":
		pwhich = "p50_00"
		plabel = "P50"
	case "90":
		pwhich = "p90_00"
		plabel = "P90"
	case "99":
		pwhich = "p99_00"
		plabel = "P99"
	case "999":
		pwhich = "p99_90"
		plabel = "P999"
	case "9999":
		pwhich = "p99_99"
		plabel = "P9999"
	default:
		fmt.Printf("invalid flag --percentile='%s'\n", ppp)
		os.Exit(1)
	}

	filename := "graph_latency_" + pwhich + "-which_" + which +
		"-pipeline_" + fmt.Sprint(pipeline) + "-kind_" + kind +
		"-scale_" + scale
	if scase != "" {
		filename += "-case_" + scase
	}
	filename += ".png"
	filename = filepath.Join(dir, "graphs", filename)
	if !force {
		_, err := os.Stat(filename)
		if err == nil {
			// fmt.Printf("File exists\n")
			os.Exit(0)
		}
	}

	title := fmt.Sprintf("%s - %d Clients - %d Ops - Pipeline %d",
		label, clients, coperations, pipeline)

	ytitle := fmt.Sprintf("%s Latency (microseconds)", plabel)

	res := gjson.Get(json, ""+
		"#(data.perf.cycles!=~*)#|"+
		"#(data.info.kind="+fmt.Sprint(kind)+")#|"+
		"#(data.info.pipeline="+fmt.Sprint(pipeline)+")#")

	// Colors
	var outColors string
	for i, cache := range caches {
		outColors += fmt.Sprintf("    \"%s\": \"%s\",\n", cache, colors[i])
	}

	// Threads
	var outXSeries string
	for i, threads := range threadz {
		if i > 0 {
			outXSeries += ", "
		}
		outXSeries += fmt.Sprintf("%d", threads)
	}

	// Benchmarks
	var outData string
	for _, cache := range caches {
		outData += fmt.Sprintf("    \"%s\": [", cache)
		res1 := res.Get("#(data.info.cache=" + cache + ")#")
		for i, threads := range threadz {
			if i > 0 {
				outData += ", "
			}
			res2 := res1.Get("#(data.info.threads=" + fmt.Sprint(threads) + ")")
			point := fmt.Sprintf("%.0f", res2.Get("data."+which+".latency."+
				pwhich).Float()*1000)
			if scase == "1" && cache == "garnet" && threads == 1 {
				point = "0"
			}
			outData += point
		}
		outData += "],\n"
	}
	drawGraph(title, ytitle, filename, outXSeries, outData, outColors, scale)
}

func graphThroughput() {
	label := ""
	switch which {
	case "get":
		which = "gets"
		label = "GET"
	case "set":
		which = "sets"
		label = "SET"
	default:
		fmt.Printf("invalid flag --which='%s'\n", which)
		os.Exit(1)
	}

	filename := "graph_opsec-which_" + which +
		"-pipeline_" + fmt.Sprint(pipeline) + "-kind_" + kind +
		"-scale_" + scale
	if scase != "" {
		filename += "-case_" + scase
	}
	filename += ".png"
	filename = filepath.Join(dir, "graphs", filename)
	if !force {
		_, err := os.Stat(filename)
		if err == nil {
			// fmt.Printf("File exists\n")
			os.Exit(0)
		}
	}

	title := fmt.Sprintf("%s - %d Clients - %d Ops - Pipeline %d",
		label, clients, coperations, pipeline)

	ytitle := "Throughput (Kops/sec)"

	res := gjson.Get(json, ""+
		"#(data.perf.cycles!=~*)#|"+
		"#(data.info.kind="+fmt.Sprint(kind)+")#|"+
		"#(data.info.pipeline="+fmt.Sprint(pipeline)+")#")
	// Colors
	var outColors string
	for i, cache := range caches {
		outColors += fmt.Sprintf("    \"%s\": \"%s\",\n", cache, colors[i])
	}

	// Threads
	var outXSeries string
	for i, threads := range threadz {
		if i > 0 {
			outXSeries += ", "
		}
		outXSeries += fmt.Sprintf("%d", threads)
	}

	// Benchmarks
	var outData string
	for _, cache := range caches {
		outData += fmt.Sprintf("    \"%s\": [", cache)
		res1 := res.Get("#(data.info.cache=" + cache + ")#")
		for i, threads := range threadz {
			if i > 0 {
				outData += ", "
			}
			res2 := res1.Get("#(data.info.threads=" + fmt.Sprint(threads) + ")")
			point := fmt.Sprintf("%d", res2.Get("data."+which+".opsec").Int()/1000)
			if scase == "1" && cache == "garnet" && threads == 1 {
				point = "0"
			}
			outData += point
		}
		outData += "],\n"
	}
	drawGraph(title, ytitle, filename, outXSeries, outData, outColors, scale)
}

func drawGraph(title, ytitle, filename, outXSeries, outData, outColors, scale string) {
	xtitle := "Threads"
	var script string
	if scale == "linear" {
		script = BarScriptLinear
	} else {
		script = BarScriptLogarithmic
	}
	script = strings.Replace(script, "{{.TITLE}}", title, -1)
	script = strings.Replace(script, "{{.XTITLE}}", xtitle, -1)
	script = strings.Replace(script, "{{.YTITLE}}", ytitle, -1)
	script = strings.Replace(script, "{{.FONTFAMILY}}", fontfamily, -1)
	script = strings.Replace(script, "{{.FILENAME}}", filename, -1)
	script = strings.Replace(script, "{{.XSERIES}}", outXSeries, -1)
	script = strings.Replace(script, "{{.DATA}}", outData, -1)
	script = strings.Replace(script, "{{.COLORS}}", outColors, -1)
	err := os.WriteFile("graph.py", []byte(script), 0666)
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("python3", "graph.py")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
	os.RemoveAll("graph.py")
}

const BarScriptLogarithmic = `
###############################################################
xseries = [{{.XSERIES}}]
data = {
    {{.DATA}}
}
colors = {
    {{.COLORS}}
}
title = "{{.TITLE}}"
ytitle = "{{.YTITLE}}"
xtitle = "{{.XTITLE}}"
filename = "{{.FILENAME}}"
fontfamily = "{{.FONTFAMILY}}"
###############################################################

import matplotlib.pyplot as plt
import matplotlib.colors as mcolors
from PIL import Image, ImageOps
import numpy as np
import math

plt.rcParams['font.family'] = fontfamily

# Create darker edge colors
edgecolors = {
    label: tuple(max(0, min(1, c * 0.4)) for c in mcolors.to_rgb(colors[label]))
    for label in colors
}

# Axes setup
x = np.arange(len(xseries))
width = 0.12
max_val = max(max(vals) for vals in data.values())

min_val = min(min(vals) for vals in data.values())
top_tick = 10 ** math.ceil(math.log(max_val, 10))
bottom_tick = 10 ** math.floor(math.log10(min_val))

# Y-ticks and quarter-decade lines
yticks = [10 ** i for i in range(int(math.log10(bottom_tick)), int(math.log10(top_tick)) + 1)]
quarter_decade_lines = []
exp_range = np.arange(math.log10(bottom_tick), math.log10(top_tick), 0.25/2)
for exp in exp_range:
    val = 10 ** exp
    if val not in yticks:
        quarter_decade_lines.append(val)

# Plot
plt.figure(figsize=(12, 7))
bars = []
for i, (label, values) in enumerate(data.items()):
    bar = plt.bar(
        x + i * width,
        values,
        width=width,
        label=label,
        color=colors[label],
        edgecolor=edgecolors[label],
        linewidth=1.5,
        zorder=3
    )
    bars.append(bar[0])

plt.yscale("log")
plt.yticks(yticks, [f"{y:,}" for y in yticks], fontsize=13)
plt.ylim(bottom_tick, top_tick)

# Quarter-decade lines and labels
for y in quarter_decade_lines:
    plt.axhline(y, color='gray', linestyle='-', linewidth=0.4, alpha=0.3, zorder=0)
    plt.text(
        -0.78, y, f"{int(round(y)):,}",
        fontsize=8,
		fontfamily='Verdana',
        color='gray',
        verticalalignment='center',
        horizontalalignment='right',
    )

# Labels and title
plt.ylabel(ytitle, fontsize=18, fontweight='bold', labelpad=20)
plt.xlabel(xtitle, fontsize=18, fontweight='bold', labelpad=20)
plt.title(title, fontsize=20, fontweight='bold', pad=30)

# X-ticks
plt.xticks(x + width * 2.5, xseries, fontsize=12)
for label in plt.gca().get_xticklabels():
    label.set_y(-0.02)
for label in plt.gca().get_yticklabels():
    label.set_horizontalalignment('right')
    label.set_x(-0.01)

# Grid
plt.grid(axis="y", which="major", linestyle='-', linewidth=0.7, alpha=0.7, zorder=0)
plt.grid(axis="x", linestyle='', zorder=0)

# Clean axis lines and ticks
ax = plt.gca()
for spine in ['left', 'bottom', 'top', 'right']:
    ax.spines[spine].set_visible(False)
ax.tick_params(axis='y', which='both', length=0)
ax.tick_params(axis='x', which='both', length=0)

# Legend
plt.legend(
    handles=bars,
    labels=data.keys(),
    fontsize=12,
    loc='center left',
    bbox_to_anchor=(1.01, 0.5),
    borderaxespad=0.,
    frameon=False,
    labelspacing=1.2,
    handlelength=1.0,
    handleheight=1.0,
    handletextpad=0.6
)

# Layout and margin
plt.tight_layout(rect=[0, 0, 0.92, 0.93])
plt.savefig(filename, dpi=150, bbox_inches='tight')
ImageOps.expand(Image.open(filename), border=40, fill='white').save(filename)
`

//
//
//
//
//
//
//

const BarScriptLinear = `
###############################################################
xseries = [{{.XSERIES}}]
data = {
    {{.DATA}}
}
colors = {
    {{.COLORS}}
}
title = "{{.TITLE}}"
ytitle = "{{.YTITLE}}"
xtitle = "{{.XTITLE}}"
filename = "{{.FILENAME}}"
fontfamily = "{{.FONTFAMILY}}"
###############################################################

import matplotlib.pyplot as plt
import matplotlib.colors as mcolors
from PIL import Image, ImageOps
import numpy as np
import math

plt.rcParams['font.family'] = fontfamily

# Create darker edge colors
edgecolors = {
    label: tuple(max(0, min(1, c * 0.4)) for c in mcolors.to_rgb(colors[label]))
    for label in colors
}

# Axes setup
x = np.arange(len(xseries))
width = 0.12
max_val = max(max(vals) for vals in data.values())


# Main linear ticks
base_ticks = np.linspace(0, max_val * 1.1, num=20)

# Minor ticks (quarter divisions)
minor_ticks = []
for i in range(len(base_ticks) - 1):
    start = base_ticks[i]
    end = base_ticks[i + 1]
    step = (end - start) / 4
    for j in range(1, 4):
        minor_ticks.append(start + j * step)

# Create figure
plt.figure(figsize=(12, 7))
bars = []
for i, (label, values) in enumerate(data.items()):
    bar = plt.bar(
        x + i * width,
        values,
        width=width,
        label=label,
        color=colors[label],
        edgecolor=edgecolors[label],
        linewidth=1.5,
        zorder=3
    )
    bars.append(bar[0])

plt.yscale("linear")
plt.yticks(base_ticks, [f"{int(y):,}" for y in base_ticks], fontsize=12)
plt.ylim(0, max_val * 1.1)

# Minor grid lines
for y in minor_ticks:
    plt.axhline(y, color='gray', linestyle='-', linewidth=0.4, alpha=0.2, zorder=0)


# Labels and title
plt.ylabel(ytitle, fontsize=18, fontweight='bold', labelpad=20)
plt.xlabel(xtitle, fontsize=18, fontweight='bold', labelpad=20)
plt.title(title, fontsize=20, fontweight='bold', pad=30)

# X-ticks and formatting
plt.xticks(x + width * 2.5, xseries, fontsize=12)
for label in plt.gca().get_xticklabels():
    label.set_y(-0.02)
for label in plt.gca().get_yticklabels():
    label.set_horizontalalignment('right')
    label.set_x(-0.01)

# Grid
plt.grid(axis="y", which="major", linestyle='-', linewidth=0.7, alpha=0.7, zorder=0)
plt.grid(axis="x", linestyle='', zorder=0)

# Clean axis lines and ticks
ax = plt.gca()
for spine in ['left', 'bottom', 'top', 'right']:
    ax.spines[spine].set_visible(False)
ax.tick_params(axis='y', which='both', length=0)
ax.tick_params(axis='x', which='both', length=0)

# Legend
plt.legend(
    handles=bars,
    labels=data.keys(),
    fontsize=12,
    loc='center left',
    bbox_to_anchor=(1.01, 0.5),
    borderaxespad=0.,
    frameon=False,
    labelspacing=1.2,
    handlelength=1.0,
    handleheight=1.0,
    handletextpad=0.6
)

# Layout and margin
plt.tight_layout(rect=[0, 0, 0.92, 0.93])
plt.savefig(filename, dpi=150, bbox_inches='tight')
ImageOps.expand(Image.open(filename), border=40, fill='white').save(filename)
`
