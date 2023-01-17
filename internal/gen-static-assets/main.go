package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

var (
	//go:embed assets.gogo
	assetsTemplateContents string
)

type AssetsTemplateData struct {
	Assets []*DataAsset
}

type DataAsset struct {
	Variable, String                    string
	variableWithExt, variableWithoutExt string
}

type H map[string]H

var (
	input      string
	outputDir  string
	baseUrl    string
	sourceDict H
)

func main() {
	parseFlag()
	parseSource()
	generateAssets()
}

func constructPackagePath(keys []string) string {
	packageNames := make([]string, len(keys))
	for i := range keys {
		packageNames[i] = keyAsPathName(keys[i])
	}
	packagePath, joinErr := url.JoinPath("", packageNames...)
	if joinErr != nil {
		panic(joinErr)
	}
	return packagePath
}

func constructAssetPath(keys []string) string {
	assetPath, joinErr := url.JoinPath(baseUrl, keys...)
	if joinErr != nil {
		panic(joinErr)
	}
	return assetPath
}

func keyAsPathName(key string) string {
	name := strcase.ToLowerCamel(key)
	if len(name) == 0 || !regexp.MustCompile("^[a-z].*$").MatchString(name) {
		name = "k" + name
	}
	return name
}

func keyAsVariable(key string) string {
	name := strcase.ToCamel(key)
	if len(name) == 0 || !regexp.MustCompile("^[A-Z].*$").MatchString(name) {
		name = "K" + name
	}
	return name
}

func prepareAssets(prevKeys []string, dict H) map[string][]*DataAsset {
	packages := make(map[string][]*DataAsset)
	for key, nextDict := range dict {
		allKeys := append(prevKeys, key)
		if len(nextDict) > 0 {
			// folder
			folderPackages := prepareAssets(allKeys, nextDict)

			// combine packages
			for pkgPath, pkgAssets := range folderPackages {
				packages[pkgPath] = append(packages[pkgPath], pkgAssets...)
			}
		} else {
			// asset
			packagePath := constructPackagePath(prevKeys)
			assetPath := constructAssetPath(allKeys)
			assetName := strings.TrimSuffix(key, filepath.Ext(key))

			// add to packages
			packages[packagePath] = append(packages[packagePath], &DataAsset{
				Variable:           "",
				String:             assetPath,
				variableWithExt:    strcase.ToCamel(key),
				variableWithoutExt: strcase.ToCamel(assetName),
			})
		}
	}

	return packages
}

func formatPackageAssets(packageAssets map[string][]*DataAsset) {
	for _, assets := range packageAssets {
		// sort the assets by url
		sort.Slice(assets, func(i, j int) bool {
			return assets[i].String < assets[j].String
		})

		// use variable without extension if possible
		assetName := make(map[string]int)

		// find all potential variable naming
		for _, asset := range assets {
			if len(asset.variableWithoutExt) == 0 {
				assetName[asset.variableWithExt]++
			} else {
				assetName[asset.variableWithoutExt]++
			}
		}

		// assign variable based on count
		for _, asset := range assets {
			if len(asset.variableWithoutExt) == 0 {
				// simply use variable with extension
				asset.Variable = asset.variableWithExt
			} else {
				// use variable without extension, must check if has conflict
				if assetName[asset.variableWithoutExt] > 1 {
					// conflicts! use variable with extension
					asset.Variable = asset.variableWithExt
				} else {
					// no issue using variable without extension
					asset.Variable = asset.variableWithoutExt
				}
			}
			// just in case Variable is empty string
			asset.Variable = keyAsVariable(asset.Variable)
		}

		// de-conflict variable names by extending conflicted names
		nameConflict := make(map[string]bool)
		for _, asset := range assets {
			for nameConflict[asset.Variable] {
				// has conflict! extend conflict variable name until it does not conflict
				asset.Variable = asset.Variable + asset.Variable[len(asset.Variable)-1:]
			}
			nameConflict[asset.Variable] = true
		}
	}
}

func generateAssets() {
	// construct package hierarchies from source dict
	packageAssets := prepareAssets(nil, sourceDict)
	formatPackageAssets(packageAssets)

	// recreate root folder for assets
	rootPath := filepath.Join(outputDir, "assets")
	deleteFolder(rootPath)
	generateFolder(rootPath)

	// create assets inside root folder
	for pkgPath, assets := range packageAssets {
		// generate assets folder
		assetsFolderPath := filepath.Join(rootPath, pkgPath)
		generateFolder(assetsFolderPath)

		// prepare assets template file data
		data := AssetsTemplateData{
			Assets: assets,
		}

		// generate assets file
		assetsPath := filepath.Join(rootPath, pkgPath, "assets.go")
		generateFile(assetsPath, assetsTemplateContents, data)
	}
}

func deleteFolder(dirPath string) {
	if rmErr := os.RemoveAll(dirPath); rmErr != nil {
		panic(rmErr)
	}
}

func generateFolder(dirPath string) {
	if dirErr := os.MkdirAll(dirPath, 0755); dirErr != nil {
		panic(dirErr)
	}
}

func generateFile(path, templateContents string, data any) {
	file, createErr := os.Create(path)
	if createErr != nil {
		panic(createErr)
	}
	defer file.Close()

	var buf bytes.Buffer
	populateTemplate(data, templateContents, &buf)
	formatted, fmtErr := format.Source(buf.Bytes())
	if fmtErr != nil {
		fmt.Println(buf.String())
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

func parseSource() {
	jsonData, readErr := os.ReadFile(input)
	if readErr != nil {
		panic(readErr)
	}

	unmarshalErr := json.Unmarshal(jsonData, &sourceDict)
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
	flag.StringVar(&outputDir, "od", "", "output base directory")
	flag.StringVar(&baseUrl, "b", "", "base url for assets")
	flag.Parse()

	if !strings.HasSuffix(input, ".json") {
		panic("require input .json file")
	}

	baseUrl = strings.TrimSuffix(baseUrl, "/")
}

// cd ~/Developer/testing/test-lib/internal/gen-static-assets;
// go build -o gen-static;
// cd ~/Developer/testing/test-lib;
// internal/gen-static-assets/gen-static -i assets.json -b "https://partying.oss-ap-southeast-1.aliyuncs.com"

// cd ~/Developer/testing/test-lib/internal/gen-static-assets; go build -o gen-static; cd ~/Developer/testing/test-lib; internal/gen-static-assets/gen-static -i assets.json -b "https://partying.oss-ap-southeast-1.aliyuncs.com"
