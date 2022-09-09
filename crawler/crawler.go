package crawler

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type ExtractedJob struct {
	Link     string
	Title    string
	Company  string
	Location string
	MetaData string
	Summary  string
}

var viewPage string

func Crawl(search string, place string, result chan<- []ExtractedJob) {
	baseUrl := fmt.Sprintf("https://%v.indeed.com/jobs?q=%v&limit=50", place, search)
	viewPage = fmt.Sprintf("https://%s.indeed.com/viewjob?jk=", place)

	jobs := []ExtractedJob{}
	c := make(chan []ExtractedJob)
	pages := getPages(baseUrl)

	for i := 0; i < pages; i++ {
		go getJobs(i, baseUrl, c)
	}

	for i := 0; i < pages; i++ {
		extractedJobs := <-c
		jobs = append(jobs, extractedJobs...)
	}

	done := writeJobs(jobs)
	if done {
		result <- jobs
		fmt.Println("Done Job")
	}
}
func getJobs(page int, baseUrl string, mainC chan<- []ExtractedJob) {

	jobs := []ExtractedJob{}
	c := make(chan ExtractedJob)
	pageUrl := baseUrl + "&start=" + strconv.Itoa(page*50)

	fmt.Println(pageUrl)

	res := getHttpRes(pageUrl)
	checkCodeStatus(res)
	doc, err := goquery.NewDocumentFromReader(res.Body)
	defer res.Body.Close()
	checkErr(err)

	cards := doc.Find(".cardOutline")
	cards.Each(func(_ int, card *goquery.Selection) {
		go extractJob(card, c)
	})

	for i := 0; i < cards.Length(); i++ {
		job := <-c
		jobs = append(jobs, job)
	}

	mainC <- jobs
}

func extractJob(card *goquery.Selection, c chan<- ExtractedJob) {
	id, _ := card.Find(".resultContent").Find("a").Attr("data-jk")
	title := card.Find(".jobTitle").Find("a").Text()
	company := card.Find(".resultContent").Find(".companyName").Text()
	location := card.Find(".resultContent").Find(".companyLocation").Text()
	metaData := card.Find(".resultContent").Find(".metadataContainer").Text()
	summary := card.Find(".jobCardShelfContainer").Find(".underShelfFooter").Find(".job-snippet").Text()

	c <- ExtractedJob{
		Link:     viewPage + id,
		Title:    title,
		Company:  company,
		Location: location,
		MetaData: metaData,
		Summary:  strings.TrimSpace(summary),
	}
}

func getPages(baseUrl string) int {
	res := getHttpRes(baseUrl)
	pages := 0
	doc, err := goquery.NewDocumentFromReader(res.Body)
	defer res.Body.Close()
	doc.Find(".pagination").Each(func(_ int, s *goquery.Selection) {
		pages = s.Find("a").Length()
	})
	checkErr(err)

	return pages
}

func getHttpRes(url string) *http.Response {
	req, err := http.NewRequest("GET", url, nil)
	checkErr(err)
	req.Header.Add("Accept", `text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8`)
	req.Header.Add("User-Agent", `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.64 Safari/537.11`)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MaxVersion: tls.VersionTLS12,
		},
	}
	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	checkErr(err)
	checkCodeStatus(res)
	return res
}

func checkErr(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}
func checkCodeStatus(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request faild with status code:", res.StatusCode)
	}
}

func writeJobs(jobs []ExtractedJob) bool {
	done := false
	file, err := os.Create("jobs.csv")
	checkErr(err)

	w := csv.NewWriter(file)
	defer w.Flush()

	headers := []string{"ViewPage", "Title", "Company", "Location", "Metadata", "Summary"}
	wErr := w.Write(headers)
	checkErr(wErr)

	for _, job := range jobs {
		row := []string{job.Link, job.Title, job.Company, job.Location, job.MetaData, job.Summary}
		wErr := w.Write(row)
		checkErr(wErr)
		done = true
	}

	return done
}
