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
	fmt.Println("Crawl")

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
		fmt.Println("Job WellDone")
	}
}
func getJobs(page int, baseUrl string, mainC chan<- []ExtractedJob) {
	fmt.Println("getjobs")

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
	fmt.Println("extractJob")

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
	fmt.Println("getPages")

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
	fmt.Println("url:", url)

	req, err := http.NewRequest("GET", url, nil)
	checkErr(err)
	req.Header.Add("accept", `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`)
	req.Header.Add("user-agent", `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36`)

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
	fmt.Println("checkErr")

	if e != nil {
		fmt.Println(e)
		log.Fatalln(e)
	}
}
func checkCodeStatus(res *http.Response) {
	fmt.Println("checkCodeStatus")

	if res.StatusCode != 200 {
		fmt.Println(res.Status)
		log.Fatalln("Request faild with status code:", res.StatusCode)
	}
}

func writeJobs(jobs []ExtractedJob) bool {
	fmt.Println("writeJobs")

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
