package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

var static = http.Dir("../../web/static/dist")

func main() {
	err := vfsgen.Generate(static, vfsgen.Options{
		PackageName:  "web",
		VariableName: "Assets",
		Filename:     "../../internal/web/generated_assets.go",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
