all:
	go build -o bench cmd/bench/main.go
	go build -o choose cmd/choose/main.go
	go build -o combine cmd/combine/main.go
	go build -o graph cmd/graph/main.go

clean:
	rm -f bench choose combine graph
