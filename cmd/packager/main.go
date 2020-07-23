package main

import (
	"log"
	"net/http"
	"os"

	"github.com/shurcooL/vfsgen"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Println("error getting working directory:", err)
	} else {
		log.Println("working directory:", wd)
	}
	static := http.Dir("../../web/static/dist")
	err = vfsgen.Generate(static, vfsgen.Options{
		PackageName:  "web",
		VariableName: "Assets",
		Filename:     "../../internal/web/generated_assets.go",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
