# Cache Benchmarks

These benchmarks measure thoughput, latency, and CPU cycles for
[Memcache](https://github.com/memcached/memcached),
[Redis](https://github.com/redis/redis),
[Valkey](https://github.com/valkey-io/valkey),
[Dragonfly](https://github.com/dragonflydb/dragonfly), and
[Garnet](https://github.com/microsoft/garnet).

- Persistance is turned off for all caches, no disk operations.
- All connetions are local, UNIX named pipes.
- The hardware is an AWS c8g.8xlarge (32 core non-NUMA ARM64).
- The benchmarking tool is [memtier_benchmark](https://github.com/RedisLabs/memtier_benchmark).
- Includes pipelining for 1, 10, 25, and 50.
- Each benchmark has 31 runs. About 15K total runs.
- The median of the 31 is used for graphing.
- Latency is measured in 50th, 90th, 99th, 99.9th, 99.99th percentiles.
- Latency also includes MAX, the absolute slowest single request.
- CPU cycles are measured using the `perf` Linux utility.
- All graphs are a [logarithmic scale](https://en.wikipedia.org/wiki/Logarithmic_scale).

The "Threads" at the bottom of each graph represents the number of I/O Threads
that the caching server software is configured to use for that specific benchmark.
All caching sofware benchmarked has some type of multithreaded networking support through
the following startup flags.

- Memcache: `-t`
- Redis: `--io-threads`
- Valkey: `--io-threads`
- Dragonfly: `--proactor_threads`
- Garnet: `--miniothreads/maxiothreads --minthreads/maxthreads`


For each benchmark, a fresh instance of the cache server software is started,
which is dedicated to 16 cores using `taskset -c 0-15`.
The memtier_benchmark tool uses the other 16 cores `taskset -c 16-31`.
Of those 16 cores, there are 256 clients spread evenly between 16 threads.
Those clients perform 100K SET and 100K GET operations, each.

There's a warmup stage that occurs at the start of each run, just after the
cache software is started. It performs a dry run of all SET operations.
This warmup is not a part of the measurements.

The `./bench-all.sh` scripts starts running the benchmarks and produces results
that are be placed in the [results](results) directory. 
Expect it to take about two weeks from start to finish to complete all runs.

| CACHE | VERSION |
| ----- | ------- |
| memcached | 1.6.38 |
| Redis | v=8.0.2 sha=994bc96b:0 malloc=jemalloc-5.3.0 bits=64 build=29503de25b2919e1 |
| Valkey | v=8.1.1 sha=fcd8bc3e:0 malloc=jemalloc-5.3.0 bits=64 build=468c2bde4cf89187 |
| dragonfly | v1.30.3-a8c40e34757396a034e98b2c1c437dd568b50c8a |
| Garnet | 1.0.65+381bb797fb158d163cd74996f7b1cfff713069fe |

**All graphs are a [logarithmic scale](https://en.wikipedia.org/wiki/Logarithmic_scale).**

## Contents

**Pipeline 1** : [Throughput](#throughput), 
[Latency 50th Percentile](#latency-50th-percentile),
[Latency 90th Percentile](#latency-90th-percentile),
[Latency 99th Percentile](#latency-99th-percentile),
[Latency 99.9th Percentile](#latency-999th-percentile),
[Latency 99.99th Percentile](#latency-9999th-percentile),
[Latency MAX](#latency-max),
[CPU Cycles](#cpu-cycles)

**Pipeline 10**: [Throughput](#throughput-1),
[Latency 50th Percentile](#latency-50th-percentile-1),
[Latency 90th Percentile](#latency-90th-percentile-1),
[Latency 99th Percentile](#latency-99th-percentile-1),
[Latency 99.9th Percentile](#latency-999th-percentile-1),
[Latency 99.99th Percentile](#latency-9999th-percentile-1),
[Latency MAX](#latency-max-1),
[CPU Cycles](#cpu-cycles-1)

**Pipeline 25**: [Throughput](#throughput-2),
[Latency 50th Percentile](#latency-50th-percentile-2),
[Latency 90th Percentile](#latency-90th-percentile-2),
[Latency 99th Percentile](#latency-99th-percentile-2),
[Latency 99.9th Percentile](#latency-999th-percentile-2),
[Latency 99.99th Percentile](#latency-9999th-percentile-2),
[Latency MAX](#latency-max-2),
[CPU Cycles](#cpu-cycles-2)

**Pipeline 50**: [Throughput](#throughput-3),
[Latency 50th Percentile](#latency-50th-percentile-3),
[Latency 90th Percentile](#latency-90th-percentile-3),
[Latency 99th Percentile](#latency-99th-percentile-3),
[Latency 99.9th Percentile](#latency-999th-percentile-3),
[Latency 99.99th Percentile](#latency-9999th-percentile-3),
[Latency MAX](#latency-max-3),
[CPU Cycles](#cpu-cycles-3)


## Pipeline 1

### Throughput

![Alt text](results/graphs/graph_opsec-which_sets-pipeline_1-kind_median.png)
---
![Alt text](results/graphs/graph_opsec-which_gets-pipeline_1-kind_median.png)
---

### Latency 50th Percentile

![Alt text](results/graphs/graph_latency_p50_00-which_sets-pipeline_1-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p50_00-which_gets-pipeline_1-kind_median.png)
---

### Latency 90th Percentile

![Alt text](results/graphs/graph_latency_p90_00-which_sets-pipeline_1-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p90_00-which_gets-pipeline_1-kind_median.png)
---

### Latency 99th Percentile

![Alt text](results/graphs/graph_latency_p99_00-which_sets-pipeline_1-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_00-which_gets-pipeline_1-kind_median.png)
---

### Latency 99.9th Percentile

![Alt text](results/graphs/graph_latency_p99_90-which_sets-pipeline_1-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_90-which_gets-pipeline_1-kind_median.png)
---

### Latency 99.99th Percentile

![Alt text](results/graphs/graph_latency_p99_99-which_sets-pipeline_1-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_99-which_gets-pipeline_1-kind_median.png)
---

### Latency MAX

![Alt text](results/graphs/graph_latency_max-which_sets-pipeline_1-kind_median.png)
---
![Alt text](results/graphs/graph_latency_max-which_gets-pipeline_1-kind_median.png)
---

### CPU Cycles

![Alt text](results/graphs/graph_cpucycles-pipeline_1-kind_median.png)
---

## Pipeline 10

### Throughput

![Alt text](results/graphs/graph_opsec-which_sets-pipeline_10-kind_median.png)
---
![Alt text](results/graphs/graph_opsec-which_gets-pipeline_10-kind_median.png)
---

### Latency 50th Percentile

![Alt text](results/graphs/graph_latency_p50_00-which_sets-pipeline_10-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p50_00-which_gets-pipeline_10-kind_median.png)
---

### Latency 90th Percentile

![Alt text](results/graphs/graph_latency_p90_00-which_sets-pipeline_10-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p90_00-which_gets-pipeline_10-kind_median.png)
---

### Latency 99th Percentile

![Alt text](results/graphs/graph_latency_p99_00-which_sets-pipeline_10-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_00-which_gets-pipeline_10-kind_median.png)
---

### Latency 99.9th Percentile

![Alt text](results/graphs/graph_latency_p99_90-which_sets-pipeline_10-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_90-which_gets-pipeline_10-kind_median.png)
---

### Latency 99.99th Percentile

![Alt text](results/graphs/graph_latency_p99_99-which_sets-pipeline_10-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_99-which_gets-pipeline_10-kind_median.png)
---

### Latency MAX

![Alt text](results/graphs/graph_latency_max-which_sets-pipeline_10-kind_median.png)
---
![Alt text](results/graphs/graph_latency_max-which_gets-pipeline_10-kind_median.png)
---

### CPU Cycles

![Alt text](results/graphs/graph_cpucycles-pipeline_10-kind_median.png)
---

## Pipeline 25

### Throughput

![Alt text](results/graphs/graph_opsec-which_sets-pipeline_25-kind_median.png)
---
![Alt text](results/graphs/graph_opsec-which_gets-pipeline_25-kind_median.png)
---

### Latency 50th Percentile

![Alt text](results/graphs/graph_latency_p50_00-which_sets-pipeline_25-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p50_00-which_gets-pipeline_25-kind_median.png)
---

### Latency 90th Percentile

![Alt text](results/graphs/graph_latency_p90_00-which_sets-pipeline_25-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p90_00-which_gets-pipeline_25-kind_median.png)
---

### Latency 99th Percentile

![Alt text](results/graphs/graph_latency_p99_00-which_sets-pipeline_25-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_00-which_gets-pipeline_25-kind_median.png)
---

### Latency 99.9th Percentile

![Alt text](results/graphs/graph_latency_p99_90-which_sets-pipeline_25-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_90-which_gets-pipeline_25-kind_median.png)
---

### Latency 99.99th Percentile

![Alt text](results/graphs/graph_latency_p99_99-which_sets-pipeline_25-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_99-which_gets-pipeline_25-kind_median.png)

### Latency MAX

![Alt text](results/graphs/graph_latency_max-which_sets-pipeline_25-kind_median.png)
---
![Alt text](results/graphs/graph_latency_max-which_gets-pipeline_25-kind_median.png)

### CPU Cycles

![Alt text](results/graphs/graph_cpucycles-pipeline_25-kind_median.png)
---

## Pipeline 50

### Throughput

![Alt text](results/graphs/graph_opsec-which_sets-pipeline_50-kind_median.png)
---
![Alt text](results/graphs/graph_opsec-which_gets-pipeline_50-kind_median.png)
---

### Latency 50th Percentile

![Alt text](results/graphs/graph_latency_p50_00-which_sets-pipeline_50-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p50_00-which_gets-pipeline_50-kind_median.png)
---

### Latency 90th Percentile

![Alt text](results/graphs/graph_latency_p90_00-which_sets-pipeline_50-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p90_00-which_gets-pipeline_50-kind_median.png)
---

### Latency 99th Percentile

![Alt text](results/graphs/graph_latency_p99_00-which_sets-pipeline_50-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_00-which_gets-pipeline_50-kind_median.png)
---

### Latency 99.9th Percentile

![Alt text](results/graphs/graph_latency_p99_90-which_sets-pipeline_50-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_90-which_gets-pipeline_50-kind_median.png)
---

### Latency 99.99th Percentile

![Alt text](results/graphs/graph_latency_p99_99-which_sets-pipeline_50-kind_median.png)
---
![Alt text](results/graphs/graph_latency_p99_99-which_gets-pipeline_50-kind_median.png)
---

### Latency MAX

![Alt text](results/graphs/graph_latency_max-which_sets-pipeline_50-kind_median.png)
---
![Alt text](results/graphs/graph_latency_max-which_gets-pipeline_50-kind_median.png)
---

### CPU Cycles

![Alt text](results/graphs/graph_cpucycles-pipeline_50-kind_median.png)
---








