package main

import (
	"log"
	"net/http"

	"github.com/shurcooL/vfsgen"
)

func main() {
	static := http.Dir("../../web/static/dist")
	err := vfsgen.Generate(static, vfsgen.Options{
		PackageName:  "web",
		VariableName: "Assets",
		Filename:     "../../internal/web/generated_assets.go",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
