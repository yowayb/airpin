package airpin

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Pinterest struct {
	token  string
	client *http.Client
	base   string
	start  string
	end    string
}

func NewPinterest(token, start, end string) *Pinterest {
	base := "https://api.pinterest.com/v5/"
	return &Pinterest{token, &http.Client{}, base, start, end}
}

func (p Pinterest) get(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+p.token)
	return req
}

func (p Pinterest) post(url string, body io.Reader) *http.Request {
	req, err := http.NewRequest("POST", url, body)
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

type Accounts struct {
	Items []Account
}

type Account struct {
	Id          string
	Name        string
	Permissions [1]string
}

type Report struct {
	AccountId string
	Records   [][]string
}

func (p Pinterest) Reports() []Report {
	res := p.do(p.get(p.base + "ad_accounts"))
	dec := json.NewDecoder(res)
	var accounts Accounts
	if err := dec.Decode(&accounts); err != nil {
		panic(err)
	}
	res.Close()
	var rs []Report
	for _, account := range accounts.Items {
		if account.Permissions[0] == "ADMIN" {
			r := Report{account.Id, p.reportFor(account)}
			rs = append(rs, r)
		}
	}
	return rs
}

//go:embed config.json
var cfg string

type CampaignReport struct {
	CampaignID                            int64   `json:"CAMPAIGN_ID"`
	CampaignName                          string  `json:"CAMPAIGN_NAME"`
	CampaignStatus                        string  `json:"CAMPAIGN_STATUS"`
	SpendInMicroDollar                    int     `json:"SPEND_IN_MICRO_DOLLAR"`
	TotalImpression                       int     `json:"TOTAL_IMPRESSION"`
	CpmInDollar                           float64 `json:"CPM_IN_DOLLAR"`
	EengagementRate                       float64 `json:"EENGAGEMENT_RATE"`
	Ectr                                  float64 `json:"ECTR"`
	EcpcInDollar                          float64 `json:"ECPC_IN_DOLLAR"`
	OutboundClick1                        int     `json:"OUTBOUND_CLICK_1"`
	CampaignDailySpendCap                 int     `json:"CAMPAIGN_DAILY_SPEND_CAP"`
	CheckoutRoas                          float64 `json:"CHECKOUT_ROAS"`
	TotalCheckout                         int     `json:"TOTAL_CHECKOUT"`
	TotalCheckoutValueInMicroDollar       int     `json:"TOTAL_CHECKOUT_VALUE_IN_MICRO_DOLLAR"`
	TotalClickAddToCart                   int     `json:"TOTAL_CLICK_ADD_TO_CART"`
	TotalClickAddToCartValueInMicroDollar int64   `json:"TOTAL_CLICK_ADD_TO_CART_VALUE_IN_MICRO_DOLLAR"`
	Date                                  string  `json:"DATE"`
}

func (p Pinterest) reportFor(account Account) [][]string {
	url := p.base + fmt.Sprintf("ad_accounts/%s/reports", account.Id)
	cfg = strings.Replace(cfg, "<START_DATE>", p.start, 1)
	cfg = strings.Replace(cfg, "<END_DATE>", p.end, 1)
	req := p.post(url, bytes.NewBufferString(cfg))
	res := p.do(req)

	dec := json.NewDecoder(res)
	var objmap map[string]interface{}
	if err := dec.Decode(&objmap); err != nil {
		panic(err)
	}
	res.Close()
	token := objmap["token"].(string)

	reportUrl := ""
	backoff := 1 * time.Second
	timeout := 60 * time.Second
	for backoff <= timeout {
		time.Sleep(backoff)
		backoff *= 2

		s := p.reportStatus(account, token)
		if s.ReportStatus == "FINISHED" {
			reportUrl = s.URL
			break
		}
	}

	// Requesting from S3 so leave out the Authorization header
	req, err := http.NewRequest("GET", reportUrl, nil)
	res = p.do(req)
	cr := csv.NewReader(res)
	rs, err := cr.ReadAll()
	if err != nil {
		panic(err)
	}
	res.Close()
	return rs
}

type status struct {
	ReportStatus string `json:"report_status"`
	URL          string `json:"url"`
	Size         int    `json:"size"`
}

func (p Pinterest) reportStatus(account Account, token string) status {
	url := p.base + fmt.Sprintf("ad_accounts/%s/reports?token=%s", account.Id, url.QueryEscape(token))
	req := p.get(url)
	res := p.do(req)

	dec := json.NewDecoder(res)
	var s status
	if err := dec.Decode(&s); err != nil {
		panic(err)
	}
	res.Close()
	return s
}
