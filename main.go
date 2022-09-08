package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"w.ryan.jung/goscrapper/crawler"
)

func crawlHnadler(w http.ResponseWriter, r *http.Request) {

	job := r.PostFormValue("job")

	if len(job) > 0 {
		crawler.Crawl(strings.ToLower(job), "uk")
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote("jobs.csv"))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, "jobs.csv")

	defer os.Remove("jobs.csv")
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("public")))
	http.HandleFunc("/crawl", crawlHnadler)
	http.ListenAndServe(":3000", nil)
}
