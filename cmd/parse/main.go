package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/russross/blackfriday"

	. "github.com/arnehilmann/goutils"
	. "github.com/arnehilmann/readlater"
)

func fetchRawMessages(dir string, result chan string) {
	defer close(result)
	fileinfos, _ := ioutil.ReadDir(dir)
	for nr, fileinfo := range fileinfos {
		if fileinfo.Name() != "8642" {
			//continue
		}
		if nr > 100000 {
			break
		}
		if fileinfo.IsDir() || strings.Contains(fileinfo.Name(), ".") {
			continue
		}
		filepath := path.Join(dir, fileinfo.Name())
		result <- filepath
	}
}

func main() {
	renderedDir := "out/rendered2"

	rawFiles := make(chan string)
	articleList := []Article{}

	go fetchRawMessages("out/fetched", rawFiles)

	os.RemoveAll(renderedDir)
	os.MkdirAll(renderedDir, 0755)

	var wg sync.WaitGroup
	for filepath := range rawFiles {
		wg.Add(1)
		go func(filepath string) {
			defer wg.Done()
			article, err := ParseMessage(filepath, renderedDir)
			WarnIf(err)
			if err == nil {
				articleList = append(articleList, article)
			}
		}(filepath)
	}
	wg.Wait()
	sort.Sort(ById(articleList))

	articleStrings := []string{}
	for nr, article := range articleList {
		articleStrings = append(articleStrings, article.Markdown())
		log.Println(nr, article)
	}

	var err error
	markdown := strings.Join(articleStrings, "\n")
	err = ioutil.WriteFile(path.Join(renderedDir, "index.md"), []byte(markdown), 0644)
	PanicIf(err)

	const html = `<html>
    <head>
        <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
        <title>Articles</title>
        <link rel="stylesheet" href="pandoc.css">
        <link rel="stylesheet" href="custom.css">
    </head>
    <body>
        {{.}}
    </body>
    </html>`
	t := template.Must(template.New("html").Parse(html))
	output := blackfriday.MarkdownCommon([]byte(markdown))

	f, err := os.Create(path.Join(renderedDir, "index.html"))
	PanicIf(err)
	defer f.Close()

	err = t.Execute(f, string(output))
	PanicIf(err)

	Copy(path.Join(renderedDir, "pandoc.css"), "res/pandoc.css")
	Copy(path.Join(renderedDir, "custom.css"), "res/custom.css")
}
