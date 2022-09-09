package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"

	"w.ryan.jung/goscrapper/crawler"
)

type Data struct {
	Items []crawler.ExtractedJob
}

func crawlHnadler(w http.ResponseWriter, r *http.Request) {

	job := r.PostFormValue("job")
	location := r.PostFormValue("location")
	c := make(chan []crawler.ExtractedJob)
	data := Data{}

	if len(job) > 0 {
		go crawler.Crawl(strings.ToLower(job), strings.ToLower(location), c)
	}

	data.Items = <-c
	tmpl, _ := template.ParseFiles("resources/templates/tpl.gohtml")
	err := tmpl.Execute(w, data)
	if err != nil {
		fmt.Println(err)
	}
}

func downloadHnadler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote("jobs.csv"))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, "jobs.csv")
	defer os.Remove("jobs.csv")
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("public")))
	http.Handle("resources/css/", http.StripPrefix("resources/css/", http.FileServer(http.Dir("resources/css"))))
	http.HandleFunc("/crawl", crawlHnadler)
	http.HandleFunc("/download", downloadHnadler)

	http.ListenAndServe(":3000", nil)
}
