package main

import (
	"flag"
	"os"
	"strings"
)

var path string

func main() {
	flag.StringVar(&path, "path", path, "path")
	flag.Parse()

	fis, err := os.ReadDir(path + "/runs")
	if err != nil {
		panic(err)
	}
	out := "[\n"
	var count int
	for _, fi := range fis {
		if strings.Contains(fi.Name(), "average.json") ||
			strings.Contains(fi.Name(), "best.json") ||
			strings.Contains(fi.Name(), "median.json") ||
			strings.Contains(fi.Name(), "worst.json") {
			data, err := os.ReadFile(path + "/runs/" + fi.Name())
			if err != nil {
				panic(err)
			}

			json := string(data)
			json = strings.Replace(json, "  ", "    ", -1)
			if strings.HasSuffix(json, "\n}\n") {
				json = json[:len(json)-3] + "\n  }\n"
			}

			if count > 0 {
				out += ","
			}
			out += "{\n" + `  "file": "` + fi.Name() + `",` + "\n" +
				`  "data": `
			out += json
			out += `}`
			count++
		}
	}
	out += "\n]\n"
	os.WriteFile(path+"/output.json", []byte(out), 0666)
}
