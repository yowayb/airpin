package airpin

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Period int

const (
	Week = iota
	Month
)

type Pinterest struct {
	token  string
	client *http.Client
	base   string
	period Period
}

func NewPinterest(token string, period Period) *Pinterest {
	root := "https://api.pinterest.com/v5/ad_accounts/"
	return &Pinterest{token, &http.Client{}, root, period}
}

func (p Pinterest) get(path string) *http.Request {
	req, err := http.NewRequest("GET", p.base+path, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+p.token)
	return req
}

func (p Pinterest) post(path string, body io.Reader) *http.Request {
	req, err := http.NewRequest("POST", p.base+path, body)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+p.token)
	req.Header.Add("Content-Type", "application/json")
	return req
}

func (p Pinterest) do(req *http.Request) io.ReadCloser {
	res, err := p.client.Do(req)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err) // don't recover; log and notify
		}
		panic(res.Status + ": " + string(body))
	}
	return res.Body
}

func (p Pinterest) Reports(account_id string) [][]string {
	token := p.requestReport(account_id)
	url := p.waitForReport(account_id, token)
	return p.getRecords(url)
}

//go:embed "config.json"
var cfg string

func (p Pinterest) requestReport(account_id string) string {
	var start_date, end_date string
	switch p.period {
	case Week:
		start_date = formatDaysAgo(7)
		end_date = formatDaysAgo(1)
	case Month:
		y, m, _ := time.Now().Date()
		_, m, d := time.Date(y, m, 0, 0, 0, 0, 0, time.UTC).Date()
		start_date = fmt.Sprintf("%v-%02d-01", y, int(m))
		end_date = fmt.Sprintf("%v-%02d-%v", y, int(m), d)
	}
	path := fmt.Sprintf("%s/reports", account_id)
	cfg = strings.Replace(cfg, "<START_DATE>", start_date, 1)
	cfg = strings.Replace(cfg, "<END_DATE>", end_date, 1)
	reader := strings.NewReader(cfg)
	req := p.post(path, reader)
	res := p.do(req)

	dec := json.NewDecoder(res)
	var objmap map[string]interface{}
	if err := dec.Decode(&objmap); err != nil {
		panic(err)
	}
	res.Close()

	token := objmap["token"].(string)
	return token
}

func (p Pinterest) waitForReport(account_id, token string) string {
	reportUrl := ""
	backoff := 1 * time.Second
	timeout := 60 * time.Second
	for backoff <= timeout {
		time.Sleep(backoff)
		backoff *= 2

		s := p.reportStatus(account_id, token)
		if s.ReportStatus == "FINISHED" {
			reportUrl = s.URL
			break
		}
	}
	return reportUrl
}

func (p Pinterest) getRecords(url string) [][]string {
	// S3 doesn't require authorization
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	res := p.do(req)
	// DEBUG (do not delete; check in to repo)
	// data, err := io.ReadAll(res)
	// if err != nil {
	// 	panic(err)
	// }

	csvReader := csv.NewReader(res)
	records, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	// DEBUG (do not delete; check in to repo)
	// header := records[0]

	// DEBUG (do not delete; check in to repo)
	// fmt.Printf("\n%#v\n")

	// DEBUG (do not delete; check in to repo)
	// for i, v := range header {
	// 	fmt.Println(i, v)
	// }

	// DEBUG (do not delete; check in to repo)
	// writeToFile("../test.csv", data)

	// DEBUG (do not delete; check in to repo)
	// os.Exit(0)

	return records
}

type status struct {
	ReportStatus string `json:"report_status"`
	URL          string `json:"url"`
	Size         int    `json:"size"`
}

func (p Pinterest) reportStatus(account_id, token string) status {
	path := fmt.Sprintf("%s/reports?token=%s", account_id, url.QueryEscape(token))
	req := p.get(path)
	res := p.do(req)

	dec := json.NewDecoder(res)
	var s status
	if err := dec.Decode(&s); err != nil {
		panic(err)
	}
	res.Close()
	return s
}

// Write the bytes to a file for inspection.  On GCP, such files will probably
// need to be created in GCS.
func writeToFile(filename string, bytes []byte) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = file.Write(bytes)
	if err != nil {
		panic(err)
	}
}
