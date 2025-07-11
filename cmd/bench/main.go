package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/jsonc"
	"github.com/tidwall/sjson"
)

var (
	config    string = "config.jsonc"   //
	threads   int    = runtime.NumCPU() // number of cache threads
	taskset   string = ""               // taskset for cache
	pipeline  int    = 1                // bench: pipeline
	btaskset  string = ""               // bench: taskset
	bthreads  int    = runtime.NumCPU() // bench: number of threads
	conns     int    = 1                // bench: number of connections per thread
	ops       int    = 100000           // bench: number of operations per connection
	sizerange string = "1-1024"         // bench: data size
	proto     string                    // bench: protocol
	tcp       bool

	perf   string = "no"              // yes or no
	isroot bool   = os.Geteuid() == 0 //

	nowarmup bool
)

const tcpport string = "19283"
const unixsocket string = "/tmp/cachebench.sock"

var success bool
var cache string
var arch string // for dragonfly binary
var vers string

func killprocs() {
	// Kill processes
	exec.Command("pkill", "valkey").Run()
	exec.Command("pkill", "redis").Run()
	exec.Command("pkill", "memcache").Run()
	exec.Command("pkill", "dragonfly").Run()
	exec.Command("pkill", "memtier").Run()
	exec.Command("pkill", "dotnet").Run()
	exec.Command("pkill", "garnet").Run()
	exec.Command("pkill", "GarnetServer").Run()
	exec.Command("pkill", "-9", "memcache").Run()
	exec.Command("pkill", "-9", "valkey").Run()
	exec.Command("pkill", "-9", "redis").Run()
	exec.Command("pkill", "-9", "dragonfly").Run()
	exec.Command("pkill", "-9", "memtier").Run()
	exec.Command("pkill", "-9", "dotnet").Run()
	exec.Command("pkill", "-9", "garnet").Run()
	exec.Command("pkill", "-9", "GarnetServer").Run()
}

func delfiles() {
	os.RemoveAll("/tmp/cachebench.sock")
	os.RemoveAll("bench-set.json")
	os.RemoveAll("bench-get.json")
	os.RemoveAll("perf.out")
}

func cleanup() {
	killprocs()
	delfiles()
}

func must[T any](v T, err error) T {
	if err != nil {
		cleanup()
		panic(err)
	}
	return v
}

func writefile(path string, data []byte) {
	must(0, os.RemoveAll(path))
	f := must(os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666))
	must(f.Write(data))
	must(0, f.Close())
}

func left(haystack, needle string) string {
	i := strings.Index(haystack, needle)
	if i == -1 {
		return ""
	}
	return haystack[:i]
}
func right(haystack, needle string) string {
	i := strings.LastIndex(haystack, needle)
	if i == -1 {
		return ""
	}
	return haystack[i+len(needle):]
}

func getpath(name string) string {
	path := gjson.Get(config, "paths."+name).String()
	if path == "" {
		panic("missing path for " + name)
	}
	path = strings.Replace(path, "${arch}", arch, -1)
	return path
}

func stripcolor(s string) string {
	parts := strings.Split(s, "\u001b[")
	for i := 1; i < len(parts); i++ {
		j := strings.IndexByte(parts[i], 'm')
		if j != -1 {
			parts[i] = parts[i][j+1:]
		}
	}
	return strings.Join(parts, "")
}

func getvers(cache string) string {
	return stripcolor(strings.TrimSpace(strings.Split(string(
		must(exec.Command(getpath(cache), "--version").CombinedOutput()),
	), "\n")[0]))
}

func writestats() {
	if !success {
		println("=== ENDED EARLY ===")
		return
	}
	println("=== WRITE FINAL OUTPUT ===")
	parsebench := func(bjson, op string) string {
		opsSec := gjson.Get(bjson, `ALL STATS.`+op+`.Ops/sec`).Float()
		avgLatency := gjson.Get(bjson, `ALL STATS.`+op+`.Average Latency`).Float()
		minLatency := gjson.Get(bjson, `ALL STATS.`+op+`.Min Latency`).Float()
		maxLatency := gjson.Get(bjson, `ALL STATS.`+op+`.Max Latency`).Float()
		kbSec := gjson.Get(bjson, `ALL STATS.`+op+`.KB/sec`).Float()
		p5000 := gjson.Get(bjson, `ALL STATS.`+op+`.Percentile Latencies.p50\.00`).Float()
		p9000 := gjson.Get(bjson, `ALL STATS.`+op+`.Percentile Latencies.p90\.00`).Float()
		p9900 := gjson.Get(bjson, `ALL STATS.`+op+`.Percentile Latencies.p99\.00`).Float()
		p9990 := gjson.Get(bjson, `ALL STATS.`+op+`.Percentile Latencies.p99\.90`).Float()
		p9999 := gjson.Get(bjson, `ALL STATS.`+op+`.Percentile Latencies.p99\.99`).Float()

		var json string
		json, _ = sjson.SetRaw(json, "opsec", fmt.Sprintf("%.3f", opsSec))
		json, _ = sjson.SetRaw(json, "mbsec", fmt.Sprintf("%.3f", kbSec/1024))
		json, _ = sjson.SetRaw(json, "latency.min", fmt.Sprintf("%.3f", minLatency))
		json, _ = sjson.SetRaw(json, "latency.max", fmt.Sprintf("%.3f", maxLatency))
		json, _ = sjson.SetRaw(json, "latency.avg", fmt.Sprintf("%.3f", avgLatency))
		json, _ = sjson.SetRaw(json, "latency.p50_00", fmt.Sprintf("%.3f", p5000))
		json, _ = sjson.SetRaw(json, "latency.p90_00", fmt.Sprintf("%.3f", p9000))
		json, _ = sjson.SetRaw(json, "latency.p99_00", fmt.Sprintf("%.3f", p9900))
		json, _ = sjson.SetRaw(json, "latency.p99_90", fmt.Sprintf("%.3f", p9990))
		json, _ = sjson.SetRaw(json, "latency.p99_99", fmt.Sprintf("%.3f", p9999))
		return json
	}
	setjson := parsebench(string(must(os.ReadFile("bench-set.json"))), "Sets")
	getjson := parsebench(string(must(os.ReadFile("bench-get.json"))), "Gets")
	var paramsjson string
	paramsjson, _ = sjson.Set(paramsjson, "cache", cache)
	paramsjson, _ = sjson.Set(paramsjson, "version", vers)
	paramsjson, _ = sjson.Set(paramsjson, "threads", threads)
	paramsjson, _ = sjson.Set(paramsjson, "bench_threads", bthreads)
	paramsjson, _ = sjson.Set(paramsjson, "connections", bthreads*conns)
	paramsjson, _ = sjson.Set(paramsjson, "operations", bthreads*conns*ops)
	paramsjson, _ = sjson.Set(paramsjson, "sizerange", sizerange)
	paramsjson, _ = sjson.Set(paramsjson, "pipeline", pipeline)

	findkey := func(perf, key string) string {
		return strings.TrimSpace(right(left(perf, key), "\n"))
	}
	finddesc := func(perf, key string) string {
		return strings.TrimSpace(right(left(perf, key), "#"))
	}

	var perfjson = "{}"
	b, err := os.ReadFile("perf.out")
	if err == nil {
		perf := string(b)
		utilized := finddesc(perf, "CPUs utilized")
		if utilized != "" {
			perfjson, _ = sjson.Set(perfjson, "cpu_utilized", utilized)
		}
		cycles := findkey(perf, "cycles")
		if cycles != "" {
			perfjson, _ = sjson.Set(perfjson, "cycles", cycles)
		}
		secsuser := findkey(perf, "seconds user")
		if secsuser != "" {
			perfjson, _ = sjson.Set(perfjson, "secsuser", secsuser)
		}
		secssys := findkey(perf, "seconds sys")
		if secssys != "" {
			perfjson, _ = sjson.Set(perfjson, "secssys", secssys)
		}
		instructions := findkey(perf, "instructions")
		if instructions != "" {
			perfjson, _ = sjson.Set(perfjson, "instructions", instructions)
		}
		branches := findkey(perf, "branches")
		if branches != "" {
			perfjson, _ = sjson.Set(perfjson, "branches", branches)
		}
		misses := findkey(perf, "branch-misses")
		if misses != "" {
			perfjson, _ = sjson.Set(perfjson, "branch_misses", misses)
		}
		faults := findkey(perf, "page-faults")
		if faults != "" {
			perfjson, _ = sjson.Set(perfjson, "page_faults", faults)
		}
	}

	json := `` +
		`{` + "\n" +
		`  "info": ` + gjson.Get(paramsjson, "@ugly").String() + `,` + "\n" +
		`  "sets": ` + gjson.Get(setjson, "@ugly").String() + `,` + "\n" +
		`  "gets": ` + gjson.Get(getjson, "@ugly").String() + ",\n" +
		`  "perf": ` + gjson.Get(perfjson, "@ugly").String() + "\n" +
		`}` + "\n"

	fmt.Printf("%s\n", json)

	println("Saving to bench.json\n")
	writefile("bench.json", []byte(json))

}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s "+
			"(valkey|reds|memcache|dragonfly|garnet) "+
			"[options]\n", os.Args[0])
		flag.PrintDefaults()
	}
	args := os.Args
	if len(args) < 2 {
		flag.Usage()
		os.Exit(1)
	}
	cache = os.Args[1]
	os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	var cacheArgs []string
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--" {
			cacheArgs = os.Args[i+1:]
			os.Args = os.Args[:i]
		}
	}

	var configPath string
	flag.StringVar(&configPath, "config", config, "config path")
	flag.StringVar(&taskset, "taskset", taskset, "taskset for cache")
	flag.IntVar(&threads, "threads", threads, "number of cache threads")
	flag.StringVar(&perf, "perf", perf, "run 'perf stat' on cache (yes or no)")
	flag.BoolVar(&tcp, "tcp", false, "bench over tcp instead of unix socket")

	flag.StringVar(&btaskset, "btaskset", btaskset, "taskset for benchmark")
	flag.IntVar(&bthreads, "bthreads", bthreads, "number of benchmark threads")
	flag.IntVar(&conns, "conns", conns, "number of connections per benchmark thread")
	flag.IntVar(&ops, "ops", ops, "number of operations per connection")
	flag.StringVar(&sizerange, "sizerange", sizerange, "number of bytes per operation")
	flag.IntVar(&pipeline, "pipeline", pipeline, "command pipeline")
	flag.StringVar(&proto, "proto", "", "protocol")

	flag.BoolVar(&nowarmup, "nowarmup", false, "nowarmup")
	flag.Parse()

	os.Args = args

	if bthreads == 0 {
		bthreads = runtime.NumCPU()
	}
	if threads == 0 {
		threads = runtime.NumCPU()
	}

	// get arch - mainly for dragonfly
	arch = runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	} else if arch == "arm64" {
		arch = "aarch64"
	}
	config = string(jsonc.ToJSONInPlace(must(os.ReadFile(configPath))))
	memtier := gjson.Get(config, "paths.memtier").String()
	vers = getvers(cache)
	var args1 []string
	switch cache {
	case "redis":
		args1 = []string{
			getpath("redis"),
			"--appendonly", "no",
			"--save", "",
			"--io-threads", fmt.Sprint(threads),
			"--maxmemory", "32gb",
		}
		if tcp {
			args1 = append(args1, "--port", tcpport)
		} else {
			args1 = append(args1, "--unixsocket", unixsocket)
			args1 = append(args1, "--port", "0")
		}
	case "dragonfly":
		maxmemory := threads * 256 // minimal mb
		if maxmemory < 32384 {
			maxmemory = 32384
		}
		maxmemory /= 1024 // convert to gb
		args1 = []string{
			getpath("dragonfly"),
			"--dir", "",
			"--dbfilename", "",
			"--proactor_threads", fmt.Sprint(threads),
			"--maxmemory", fmt.Sprint(maxmemory) + "gb",
		}
		if tcp {
			args1 = append(args1, "--port", tcpport)
		} else {
			args1 = append(args1, "--unixsocket", unixsocket)
			args1 = append(args1, "--port", "0")
		}
	case "valkey":
		args1 = []string{
			getpath("valkey"),
			"--appendonly", "no",
			"--save", "",
			"--io-threads", fmt.Sprint(threads),
			"--maxmemory", "32gb",
		}
		if tcp {
			args1 = append(args1, "--port", tcpport)
		} else {
			args1 = append(args1, "--unixsocket", unixsocket)
			args1 = append(args1, "--port", "0")
		}
	case "memcache":
		proto = "memcache_text"
		args1 = []string{
			getpath("memcache"),
			"-m", "32768",
			"-t", fmt.Sprint(threads),
		}
		if tcp {
			args1 = append(args1, "-p", tcpport)
		} else {
			args1 = append(args1, "-s", unixsocket)
			args1 = append(args1, "-p", "0")
		}
		if isroot {
			// memcache requires flag when running as root
			args1 = append(args1, "-u", "root")
		}
	case "garnet":
		args1 = []string{
			getpath("garnet"),
			"--no-obj",
			"--aof-null-device",
			"--readcache", "false",
			"--index", "2g",
			"--memory", "32g",
			"--miniothreads", fmt.Sprint(threads),
			"--maxiothreads", fmt.Sprint(threads),
			"--minthreads", fmt.Sprint(threads),
			"--maxthreads", fmt.Sprint(threads),
		}
		if tcp {
			args1 = append(args1, "--port", tcpport)
		} else {
			args1 = append(args1, "--unixsocket", unixsocket)
			args1 = append(args1, "--port", "0")
		}
	default:
		fmt.Fprintf(os.Stderr, "invalid cache: %s, expected 'valkey', "+
			"'redis', 'memcache', 'dragonfly', 'garnet'\n", cache)
		os.Exit(1)
	}

	args1 = append(args1, cacheArgs...)

	args2 := []string{
		memtier,
		"-c", fmt.Sprint(conns),
		"-t", fmt.Sprint(bthreads),
		"-n", fmt.Sprint(ops),
		"--distinct-client-seed",
		"--hide-histogram",
		"--key-prefix", "",
		"--ratio", "1:0",
		"--data-size-range", sizerange,
		"--pipeline", fmt.Sprint(pipeline),
		"--json-out-file", "bench-set.json",
		"--print-percentiles", "50,90,99,99.9,99.99",
		"--key-pattern=P:P",
	}
	if tcp {
		args2 = append(args2, "-p", tcpport)
	} else {
		args2 = append(args2, "-S", unixsocket)
	}
	args3 := []string{
		memtier,
		"-c", fmt.Sprint(conns),
		"-t", fmt.Sprint(bthreads),
		"-n", fmt.Sprint(ops),
		"--distinct-client-seed",
		"--hide-histogram",
		"--key-prefix", "",
		"--ratio", "0:1",
		"--data-size-range", sizerange,
		"--pipeline", fmt.Sprint(pipeline),
		"--json-out-file", "bench-get.json",
		"--print-percentiles", "50,90,99,99.9,99.99",
		"--key-pattern=P:P",
	}
	if tcp {
		args3 = append(args3, "-p", tcpport)
	} else {
		args3 = append(args3, "-S", unixsocket)
	}
	if taskset != "" {
		args1 = append([]string{"taskset", "-c", taskset}, args1...)
	}
	// if perf == "yes" {
	// 	args1 = append([]string{"perf", "stat"}, args1...)
	// }
	if btaskset != "" {
		args2 = append([]string{"taskset", "-c", btaskset}, args2...)
		args3 = append([]string{"taskset", "-c", btaskset}, args3...)
	}
	if proto != "" {
		args2 = append(args2, "--protocol", proto)
		args3 = append(args3, "--protocol", proto)
	}

	//////////////////////////////////////////////////////////////////////////
	cleanup()
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for range sigch {
			fmt.Printf("=== SIGNAL ===\n")
			cleanup()
			os.Exit(1)
		}
	}()

	println("=== START CACHE ===")
	fmt.Printf("%s\n", args1)
	cmd1 := exec.Command(args1[0], args1[1:]...)
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	must(0, cmd1.Start())
	if cmd1.Process.Pid <= 0 {
		panic("bad pid")
	}
	// wait for server to come online
	start := time.Now()
	for {
		ok := func() bool {
			var conn net.Conn
			var err error
			if tcp {
				conn, err = net.Dial("tcp", ":"+tcpport)
			} else {
				conn, err = net.Dial("unix", unixsocket)
			}
			if err != nil {
				return false
			}
			defer conn.Close()
			msg := []byte("*1\r\n$4\r\nPING\r\n")
			n, err := conn.Write(msg)
			if err != nil || n != len(msg) {
				return false
			}
			var buf [64]byte
			n, err = conn.Read(buf[:])
			if err != nil {
				return false
			}
			if string(buf[:n]) != "+PONG\r\n" &&
				// Memcache will send back three ERROR messages
				string(buf[:n]) != "ERROR\r\nERROR\r\nERROR\r\n" {
				return false
			}
			return true
		}()
		if ok {
			break
		}
		if time.Since(start) > time.Second*10 {
			fmt.Printf("=== CONNECTION TIMEOUT ===\n")
			os.Exit(1)
		}
		time.Sleep(time.Millisecond * 10)
	}
	time.Sleep(time.Millisecond * 100)

	var cmd2 *exec.Cmd
	if !nowarmup {
		// The server is up and running. Perform a warmup SET benchmark.
		// This will ensure that the hashtables are filled, giving the final
		// SET benchmark the best opportunity for lowest latency.
		println("=== START MEMTIER SET(warmup) ===")
		fmt.Printf("%s\n", args2)
		cmd2 = exec.Command(args2[0], args2[1:]...)
		cmd2.Stderr = os.Stderr
		cmd2.Stdout = os.Stdout
		must(0, cmd2.Run())
	}

	// The server is up and running and the warmup run has finished.
	// Start the performance counter and begin final benchmarking.
	var cmdP *exec.Cmd
	var perfwg sync.WaitGroup
	if perf == "yes" {
		// use 'perf stat'
		fmt.Printf("=== PERF STAT REQUIRES SUDO ===\n")
		cmdS := exec.Command("sudo", "echo", "OK")
		cmdS.Stderr = os.Stderr
		cmdS.Stdout = os.Stdout
		cmdS.Stdin = os.Stdin
		if err := cmdS.Run(); err != nil {
			panic(err)
		}
		cmdP = exec.Command("sudo", "perf", "stat", "--no-big-num",
			"--no-scale", "-p", fmt.Sprint(cmd1.Process.Pid))
		rd1 := bufio.NewReader(io.MultiReader(must(cmdP.StdoutPipe()),
			must(cmdP.StderrPipe())))
		must(0, cmdP.Start())
		perfwg.Add(1)
		go func() {
			var perfstats []byte
			var perfok bool
			for {
				line, err := rd1.ReadBytes('\n')
				if err != nil {
					break
				}
				if bytes.Contains(line, []byte("Performance counter stats")) {
					println("=== PERFORMANCE STATS ===")
					perfok = true
					perfstats = append(perfstats, string(line)+"]]]]"...)
				} else if perfok {
					perfstats = append(perfstats, line...)
				}
				os.Stderr.Write(line)
			}
			if perfok {
				writefile("perf.out", perfstats)
			}
			perfwg.Done()
		}()
	}

	println("=== START MEMTIER SET ===")
	fmt.Printf("%s\n", args2)
	cmd2 = exec.Command(args2[0], args2[1:]...)
	cmd2.Stderr = os.Stderr
	cmd2.Stdout = os.Stdout
	must(0, cmd2.Run())

	println("=== START MEMTIER GET ===")
	fmt.Printf("%s\n", args3)
	cmd2 = exec.Command(args3[0], args3[1:]...)
	cmd2.Stderr = os.Stderr
	cmd2.Stdout = os.Stdout
	must(0, cmd2.Run())

	fmt.Printf("=== BENCHMARK COMPLETE ===\n")
	if perf == "yes" {
		exec.Command("sudo", "kill", "-INT", fmt.Sprint(cmdP.Process.Pid)).Run()
		perfwg.Wait()
	}
	killprocs()
	success = true
	writestats()
	cleanup()
}
