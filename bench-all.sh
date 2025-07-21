#!/usr/bin/env bash

set -e
cd $(dirname "${BASH_SOURCE[0]}")

# Cache programs to benchmark
progs="memcache dragonfly valkey redis garnet pogocache"

# Cache threading to benchmark
threadz="1 2 3 4 5 6 7 8 10 12 14 16"

# Number of operations per SET and GET, per benchmark connection threads.
# Thus if you set this 1000 and running on machine with 32 threads then there
# will be 32 conncurrent benchmark connections, each executing 1000 SETSs
# followed by 1000 GETs.
nops=100000

# Number of benchmark connection threads. Setting to zero will cause the 
# benchmark tool to auto detect the total number of system threads (vCPUs) and
# use that value.
bthreads=16

# Number of connections per benchmark thread. For example if this is set to 10, 
# with 32 benchmark threads, and 100,000 ops per connection, then there will be
# 320 connection concurrently executing 100,000 SETs followed by 100,000 GETs,
# for a total of 64,000,000 operations, per run.
conns=16

# Value size range, randomly selected.
sizerange=1-1024

# Number of runs per benchmark.
runs=31

# Cache pipelining to benchmark
pipelines="1 10 25 50" 

# Performance stats. Having both no and yes will do runs with and without 
# 'perf stat'.
perfs="no yes"

# Pin caches process to CPUs. This runs 'taskset -c <ctaskset>`
ctaskset="0-15"

# Pin benchmark process to CPUs. This runs 'taskset -c <btaskset>`
btaskset="16-31"

# Final directory for storing all results
resultsdir="results"

# Bench graphs
benches="throughput latency cpucycles" 

# Latency percentiles
percentiles="50 90 99 999 9999 min max avg"

# Operations 
ops="get set"

# Graph scales
scales="logarithmic linear"

################################################################################
## FUNCTIONS
################################################################################

runfile() {
    prog=$1; threads=$2; pipeline=$3; perf=$4; run=$5
    echo "results/runs/bench_$prog-threads_$threads-pipeline_$pipeline-perf_$perf-run_$run.json"
}

# Run a benchmark
# usage: bench <prog> <threads> <pipeline> <perf> <run_number>
bench() {
    prog=$1; threads=$2; pipeline=$3; perf=$4; run=$5
    echo "=== BENCH PROG($prog) THREADS($threads) PIPELINE($pipeline) PERF($perf) RUN($run) ===" 
    json="$(runfile $prog $threads $pipeline $perf $run)"
    if [[ ! -f "$json" ]]; then
        ./bench $prog --threads=$threads --pipeline=$pipeline --perf=$perf \
            --ops=$nops --bthreads="$bthreads" --taskset="$ctaskset" \
            --btaskset="$btaskset" --sizerange="$sizerange" --conns="$conns"
        chmod 666 bench.json
        mv bench.json $json
    fi
}

# choose the best, worst, and average results
choose() {
    prog=$1; threads=$2; pipeline=$3; perf=$4
    ./choose --prog=$prog --threads=$threads --pipeline=$pipeline \
        --perf=$perf --runs=$runs --path="$resultsdir"
}

graph() {
    bench=$1; pipeline=$2; percentile=$3; op=$4; scale=$5; scase=$6; 
    echo "=== GRAPH BENCH($bench) PIPELINE($pipeline) PERCENTILE($percentile) PERF($op) SCALE($scale) ==="
    ./graph --bench=$bench --pipeline=$pipeline --percentile=$percentile \
        --which=$op --scale=$scale --dir=results --scase=$scase
}

################################################################################
## PROGRAM
################################################################################

make
mkdir -p results/runs results/graphs
chmod 777 results
chmod 777 results/runs
chmod 777 results/graphs

if [[ ! -f "$resultsdir/output.json" ]]; then
# Perform benchmark runs, this will take a while (~8 hours)
for prog in ${progs}; do
    for threads in ${threadz}; do
        for pipeline in ${pipelines}; do
            for perf in ${perfs}; do
                for run in $(seq 1 $runs); do
                    bench $prog $threads $pipeline $perf $run
                done
                choose $prog $threads $pipeline $perf
            done
        done
    done
done
./combine --path="$resultsdir"
fi
echo === SAVED OUTPUT ===
for scale in ${scales}; do
    for bench in ${benches}; do
        for pipeline in ${pipelines}; do
            if [[ "$bench" == "latency" ]]; then
                for percentile in ${percentiles}; do
                    for op in ${ops}; do
                        graph $bench $pipeline $percentile $op $scale
                    done
                done
            elif [[ "$bench" == "throughput" ]]; then
                for op in ${ops}; do
                    graph $bench $pipeline "" $op $scale
                done
            else
                graph $bench $pipeline "" "" $scale
            fi
        done
    done
done
echo "=== SPECIAL CASE(remove garnet for latency 1 thread) ==="
graph latency 1 99 set linear 1
graph latency 1 99 get linear 1
