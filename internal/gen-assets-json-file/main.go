package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type H map[string]H

var (
	inputs StringSlice
	output string
	data   H
)

func main() {
	parseFlag()
	prepareData()
	generateOutput()
}

func generateOutput() {
	file, createErr := os.Create(output)
	if createErr != nil {
		panic(createErr)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "\t")
	encodeErr := encoder.Encode(data)
	if encodeErr != nil {
		panic(encodeErr)
	}
}

func prepareData() {
	data = H{}
	for _, input := range inputs {
		inputParts := strings.Split(input, string(os.PathSeparator))
		lastDir := inputParts[len(inputParts)-1]

		filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				panic(err)
			}

			parts := strings.Split(path, string(os.PathSeparator))
			for len(parts) > 0 && parts[0] != lastDir {
				parts = parts[1:]
			}

			dataPtr := data
			for _, part := range parts {
				if _, found := dataPtr[part]; !found {
					dataPtr[part] = H{}
				}
				dataPtr = dataPtr[part]
			}

			return nil
		})
	}
}

func parseFlag() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate json file of directory hierarchy")
		flag.PrintDefaults()
	}

	flag.Var(&inputs, "i", "input directory")
	flag.StringVar(&output, "o", "", "output json file")
	flag.Parse()

	if !strings.HasSuffix(output, ".json") {
		panic("require output .json file")
	}
}

type StringSlice []string

func (z StringSlice) String() string {
	return strings.Join(z, ",")
}

func (z *StringSlice) Set(value string) error {
	*z = append(*z, value)
	return nil
}

// cd ~/Developer/testing/test-lib/internal/gen-assets-json-file;
// go build -o gen-json;
// cd ~/Developer;
// testing/test-lib/internal/gen-assets-json-file/gen-json -i oss-veeka-assets/static -i oss-veeka-assets/veeka -o assets.json
