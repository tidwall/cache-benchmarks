package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var path string
var prog string
var threads int
var pipeline int
var runs int
var perf string

func main() {
	flag.StringVar(&path, "path", path, "path")
	flag.StringVar(&prog, "prog", prog, "prog")
	flag.IntVar(&threads, "threads", threads, "threads")
	flag.IntVar(&pipeline, "pipeline", pipeline, "pipeline")
	flag.StringVar(&perf, "perf", perf, "perf")
	flag.IntVar(&runs, "runs", runs, "runs")
	flag.Parse()

	path += "/runs"

	// println(prog, threads, pipeline, perf, runs)

	choose("median")
	choose("best")
	choose("worst")
	choose("average")
}

func runfile(run int) string {
	return fmt.Sprintf("%s/bench_%s-threads_%d-pipeline_%d-perf_%s-run_%d.json",
		path, prog, threads, pipeline, perf, run+1)
}

func resultfile(choose string) string {
	return fmt.Sprintf("%s/bench_%s-threads_%d-pipeline_%d-perf_%s-run_%s.json",
		path, prog, threads, pipeline, perf, choose)
}

func runjson(run int) string {
	data, err := os.ReadFile(runfile(run))
	if err != nil {
		panic(err)
	}
	return string(data)
}

func cleanperf(json gjson.Result) gjson.Result {
	if !json.Get("cycles").Exists() {
		return json
	}
	raw := json.Raw
	raw, _ = sjson.SetRaw(raw, "cpu_utilized", fmt.Sprintf("%.3f", json.Get("cpu_utilized").Float()))
	raw, _ = sjson.SetRaw(raw, "cycles", fmt.Sprintf("%.0f", json.Get("cycles").Float()))
	raw, _ = sjson.SetRaw(raw, "instructions", fmt.Sprintf("%.0f", json.Get("instructions").Float()))
	raw, _ = sjson.SetRaw(raw, "branches", fmt.Sprintf("%.0f", json.Get("branches").Float()))
	raw, _ = sjson.SetRaw(raw, "branch_misses", fmt.Sprintf("%.0f", json.Get("branch_misses").Float()))
	raw, _ = sjson.SetRaw(raw, "page_faults", fmt.Sprintf("%.0f", json.Get("page_faults").Float()))
	return gjson.Parse(raw)
}

func choose(kind string) {
	var tinfo gjson.Result
	var gets []gjson.Result
	var sets []gjson.Result
	var perf []gjson.Result
	for run := 0; run < runs; run++ {
		json := runjson(run)
		gets = append(gets, gjson.Get(json, "gets"))
		sets = append(sets, gjson.Get(json, "sets"))
		perf = append(perf, gjson.Get(json, "perf"))
		tinfo = gjson.Get(json, "info")
	}
	var tgets gjson.Result
	var tsets gjson.Result
	var tperf gjson.Result
	sort.Slice(gets, func(i, j int) bool {
		return gets[i].Get("opsec").Float() < gets[j].Get("opsec").Float()
	})
	sort.Slice(sets, func(i, j int) bool {
		return sets[i].Get("opsec").Float() < sets[j].Get("opsec").Float()
	})
	sort.Slice(sets, func(i, j int) bool {
		return perf[i].Get("cycles").Int() > perf[j].Get("cycles").Int()
	})
	if runs > 10 {
		// remove outliers
		nouts := runs / 10
		gets = gets[nouts : runs-nouts]
		sets = sets[nouts : runs-nouts]
		perf = perf[nouts : runs-nouts]
		runs -= nouts * 2
	}
	if kind == "average" {
		tgets, tsets, tperf = calcAverage(gets, sets, perf)
	} else {
		m := 0
		switch kind {
		case "best":
			m = runs - 1
		case "worst":
			m = 0
		case "median":
			m = runs/2 + 1
		default:
			panic("invalid kind: " + kind)
		}
		tgets = gets[m]
		tsets = sets[m]
		tperf = perf[m]
	}
	raw, _ := sjson.Set(tinfo.Raw, "kind", kind)
	tinfo = gjson.Parse(raw)
	out := "" +
		"{\n" +
		"  \"info\": " + tinfo.Get("@ugly").Raw + ",\n" +
		"  \"sets\": " + tsets.Get("@ugly").Raw + ",\n" +
		"  \"gets\": " + tgets.Get("@ugly").Raw + ",\n" +
		"  \"perf\": " + cleanperf(tperf).Get("@ugly").Raw + "\n" +
		"}\n"
	err := os.WriteFile(resultfile(kind), []byte(out), 0666)
	if err != nil {
		panic(err)
	}
}

func sumops(json gjson.Result, add gjson.Result) gjson.Result {
	raw := json.Raw
	raw, _ = sjson.Set(raw, "opsec", json.Get("opsec").Float()+add.Get("opsec").Float())
	raw, _ = sjson.Set(raw, "mbsec", json.Get("mbsec").Float()+add.Get("mbsec").Float())
	raw, _ = sjson.Set(raw, "latency.min", json.Get("latency.min").Float()+add.Get("latency.min").Float())
	raw, _ = sjson.Set(raw, "latency.max", json.Get("latency.max").Float()+add.Get("latency.max").Float())
	raw, _ = sjson.Set(raw, "latency.avg", json.Get("latency.avg").Float()+add.Get("latency.avg").Float())
	raw, _ = sjson.Set(raw, "latency.p50_00", json.Get("latency.p50_00").Float()+add.Get("latency.p50_00").Float())
	raw, _ = sjson.Set(raw, "latency.p90_00", json.Get("latency.p90_00").Float()+add.Get("latency.p90_00").Float())
	raw, _ = sjson.Set(raw, "latency.p99_00", json.Get("latency.p99_00").Float()+add.Get("latency.p99_00").Float())
	raw, _ = sjson.Set(raw, "latency.p99_90", json.Get("latency.p99_90").Float()+add.Get("latency.p99_90").Float())
	raw, _ = sjson.Set(raw, "latency.p99_90", json.Get("latency.p99_90").Float()+add.Get("latency.p99_90").Float())
	raw, _ = sjson.Set(raw, "latency.p99_99", json.Get("latency.p99_99").Float()+add.Get("latency.p99_99").Float())
	return gjson.Parse(raw)
}

func sumperf(json gjson.Result, add gjson.Result) gjson.Result {
	if !json.Get("cycles").Exists() {
		return json
	}
	raw := json.Raw
	raw, _ = sjson.Set(raw, "cpu_utilized", json.Get("cpu_utilized").Float()+add.Get("cpu_utilized").Float())
	raw, _ = sjson.Set(raw, "cycles", json.Get("cycles").Float()+add.Get("cycles").Float())
	raw, _ = sjson.Set(raw, "instructions", json.Get("instructions").Float()+add.Get("instructions").Float())
	raw, _ = sjson.Set(raw, "branches", json.Get("branches").Float()+add.Get("branches").Float())
	raw, _ = sjson.Set(raw, "branch_misses", json.Get("branch_misses").Float()+add.Get("branch_misses").Float())
	raw, _ = sjson.Set(raw, "page_faults", json.Get("page_faults").Float()+add.Get("page_faults").Float())
	return gjson.Parse(raw)
}

func avgops(json gjson.Result) gjson.Result {
	raw := json.Raw
	raw, _ = sjson.SetRaw(raw, "opsec", fmt.Sprintf("%.3f", json.Get("opsec").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "mbsec", fmt.Sprintf("%.3f", json.Get("mbsec").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.min", fmt.Sprintf("%.3f", json.Get("latency.min").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.max", fmt.Sprintf("%.3f", json.Get("latency.max").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.avg", fmt.Sprintf("%.3f", json.Get("latency.avg").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.p50_00", fmt.Sprintf("%.3f", json.Get("latency.p50_00").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.p90_00", fmt.Sprintf("%.3f", json.Get("latency.p90_00").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.p99_00", fmt.Sprintf("%.3f", json.Get("latency.p99_00").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.p99_90", fmt.Sprintf("%.3f", json.Get("latency.p99_90").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.p99_90", fmt.Sprintf("%.3f", json.Get("latency.p99_90").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "latency.p99_99", fmt.Sprintf("%.3f", json.Get("latency.p99_99").Float()/float64(runs)))
	return gjson.Parse(raw)

}

func avgperf(json gjson.Result) gjson.Result {
	if !json.Get("cycles").Exists() {
		return json
	}
	raw := json.Raw
	raw, _ = sjson.SetRaw(raw, "cpu_utilized", fmt.Sprintf("%.3f", json.Get("cpu_utilized").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "cycles", fmt.Sprintf("%.0f", json.Get("cycles").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "instructions", fmt.Sprintf("%.0f", json.Get("instructions").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "branches", fmt.Sprintf("%.0f", json.Get("branches").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "branch_misses", fmt.Sprintf("%.0f", json.Get("branch_misses").Float()/float64(runs)))
	raw, _ = sjson.SetRaw(raw, "page_faults", fmt.Sprintf("%.0f", json.Get("page_faults").Float()/float64(runs)))
	return gjson.Parse(raw)
}

func calcAverage(agets []gjson.Result, asets []gjson.Result,
	aperf []gjson.Result,
) (tgets gjson.Result, tsets gjson.Result, tperf gjson.Result) {
	runs := len(agets)
	for run := 0; run < runs; run++ {
		gets := agets[run]
		sets := asets[run]
		perf := aperf[run]
		if run == 0 {
			tgets = gets
		} else {
			tgets = sumops(tgets, gets)
		}
		if run == 0 {
			tsets = sets
		} else {
			tsets = sumops(tsets, sets)
		}
		if run == 0 {
			tperf = perf
		} else {
			tperf = sumperf(tperf, perf)
		}
	}
	tgets = avgops(tgets)
	tsets = avgops(tsets)
	tperf = avgperf(tperf)
	return tgets, tsets, tperf
}
