package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

var (
	//go:embed assets.gogo
	templateContents string
)

type Data struct {
	PackageName, BaseUrl string
	Assets               []DataAsset
}

type DataAsset struct {
	Variable, Parent, Name string
}

type H map[string]H

var (
	input      string
	output     string
	baseUrl    string
	pkgName    string
	sourceData H
	assetsData Data
)

func main() {
	parseFlag()
	parseSource()
	prepareAssets()
	generateAssets()
}

func generateAssets() {
	file, createErr := os.Create(output)
	if createErr != nil {
		panic(createErr)
	}
	defer file.Close()

	var buf bytes.Buffer
	populateTemplate(assetsData, templateContents, &buf)
	formatted, fmtErr := format.Source(buf.Bytes())
	if fmtErr != nil {
		panic(fmtErr)
	}
	if _, writeErr := file.Write(formatted); writeErr != nil {
		panic(writeErr)
	}
}

func populateTemplate(data any, tmpl string, output io.Writer) {
	t, err := template.New("model").Funcs(template.FuncMap{
		"toSnake":      strcase.ToSnake,
		"toCamel":      strcase.ToCamel,
		"toLowerCamel": strcase.ToLowerCamel,
		"join":         strings.Join,
	}).Parse(tmpl)
	if err != nil {
		panic(err)
	}

	err = t.Execute(output, data)
	if err != nil {
		panic(err)
	}
}

func prepareAssets() {
	assetsData.PackageName = pkgName
	assetsData.BaseUrl = baseUrl
	assetsData.Assets = flatten(sourceData, nil)

	sort.Slice(assetsData.Assets, func(i, j int) bool {
		return assetsData.Assets[i].Variable < assetsData.Assets[j].Variable
	})
}

func flatten(data H, parent *DataAsset) []DataAsset {
	var assets []DataAsset
	for key, dict := range data {
		asset := DataAsset{
			Name: key,
		}
		if parent == nil {
			asset.Variable = strcase.ToCamel(key)
			asset.Parent = "nil"
		} else {
			asset.Variable = parent.Variable + "_" + strcase.ToCamel(key)
			asset.Parent = "&" + parent.Variable
		}

		assets = append(assets, asset)
		assets = append(assets, flatten(dict, &asset)...)
	}
	return assets
}

func parseSource() {
	jsonData, readErr := os.ReadFile(input)
	if readErr != nil {
		panic(readErr)
	}

	unmarshalErr := json.Unmarshal(jsonData, &sourceData)
	if unmarshalErr != nil {
		panic(unmarshalErr)
	}
}

func parseFlag() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate assets from json")
		flag.PrintDefaults()
	}

	flag.StringVar(&input, "i", "", "input json file")
	flag.StringVar(&output, "o", "", "output go file")
	flag.StringVar(&pkgName, "p", "", "package name")
	flag.StringVar(&baseUrl, "b", "", "base url for assets")
	flag.Parse()

	if !strings.HasSuffix(input, ".json") {
		panic("require input .json file")
	}

	baseUrl = strings.TrimSuffix(baseUrl, "/")

	if len(output) == 0 {
		output = input
		output = strings.TrimSuffix(output, ".json") + ".go"
	}

	if !strings.HasSuffix(output, ".go") {
		panic("require output .go file")
	}

	if len(pkgName) == 0 {
		pkgName = "s3"
	}
}

// cd ~/Developer/testing/test-lib/internal/gen-static-assets;
// go build -o gen-static;
// cd ~/Developer/testing/test-lib;
// internal/gen-static-assets/gen-static -i assets.json -b "https://partying.oss-ap-southeast-1.aliyuncs.com"

// cd ~/Developer/testing/test-lib/internal/gen-static-assets; go build -o gen-static; cd ~/Developer/testing/test-lib; internal/gen-static-assets/gen-static -i assets.json -b "https://partying.oss-ap-southeast-1.aliyuncs.com"
