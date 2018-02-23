//+build generate

package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {
	var fs http.FileSystem = http.Dir("../third_party/OpenAPI/")

	err := vfsgen.Generate(fs, vfsgen.Options{
		Filename:     "static.go",
		PackageName:  "static",
		VariableName: "Assets",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
