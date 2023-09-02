package airpin

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Airtable struct {
	token  string
	client *http.Client
	base   string
	start  string
	end    string
}

// TODO make a toggle in GCP for this
// TEST bases
var _ = map[string]string{
	"...": "...", // TEST URBAN
	"...": "...", // TEST OPTIMUM
}

// Update when Julie creates bases
var bases = map[string]string{
	"...": "...", // Urban Natural Home Furnishings
	// "...": "",                  // Goodie Goodie (Make Cake)
}

type records struct {
	Records  []record `json:"records"`
	Typecast bool     `json:"typecast"`
}

type record struct {
	ID     string `json:"id,omitempty"`
	Fields fields `json:"fields,omitempty"`
}

type fields struct {
	ReportDate         string    `json:"Report Date"`
	Start              string    `json:"Start"`
	End                string    `json:"End"`
	CampaignID         string    `json:"Campaign ID"`
	CampaignName       string    `json:"Campaign Name"`
	CampaignStatus     string    `json:"Campaign Status"`
	DailyBudget        JSONFloat `json:"Daily Budget"`
	Spend              JSONFloat `json:"Spend"`
	Impressions        int       `json:"Impressions"`
	Cpm                JSONFloat `json:"CPM"`
	EngagementRatePINS JSONFloat `json:"Engagement Rate (PINS)"`
	CTRPINS            JSONFloat `json:"CTR (PINS)"`
	OutboundClicks     int       `json:"Outbound Clicks"`
	Cpc                JSONFloat `json:"CPC"`
	ClickRatioPINS     JSONFloat `json:"Click Ratio (PINS)"`
	Checkouts          int       `json:"Checkouts"`
	OrderValueCheckout JSONFloat `json:"Order Value (Checkout)"`
	ConversionRatePINS JSONFloat `json:"Conversion Rate (PINS)"`
	Cpa                JSONFloat `json:"CPA"`
	Roas               JSONFloat `json:"ROAS"`
	ATCs               int       `json:"ATCs"`
	OrderValueATCs     JSONFloat `json:"Order Value (ATCs)"`
	Reach              JSONFloat
	Frequency          JSONFloat
	Leads              JSONFloat
}

func NewAirtable(token, start, end string) *Airtable {
	base := "https://api.airtable.com/v0/"
	return &Airtable{token, &http.Client{}, base, start, end}
}

func (a Airtable) UpdateAll(rs []Report) {
	for _, r := range rs {
		if baseId := bases[r.AccountId]; baseId != "" {
			a.update(baseId, r.Records)
		}
	}
}

func (a Airtable) update(baseId string, rs [][]string) {
	// for i := 0; i < 2; i++ {
	// 	for idx, r := range rs[i] {
	// 		fmt.Println(idx, r)
	// 	}
	// }
	// os.Exit(0)
	rs = rs[1:] // skip header
	fullChunkCount := len(rs) / 10
	for i := 0; i < fullChunkCount; i++ {
		a.post(baseId, a.chunk(rs[i*10:(i+1)*10]))
	}
	a.post(baseId, a.chunk(rs[fullChunkCount*10:]))
}

func (a Airtable) post(baseId string, chunk []byte) {
	url := a.base + baseId + "/" + url.PathEscape("Ad KPIs")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(chunk))
	req.Header.Add("Authorization", "Bearer "+a.token)
	req.Header.Add("Content-Type", "application/json")
	res, err := a.client.Do(req)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		panic(res.Status + ": " + string(body))
	}
}

func (a Airtable) chunk(rs [][]string) []byte {
	var recs records
	for _, r := range rs {
		impressions := parseFloat(r[20]) + parseFloat(r[21])
		clicks := parseFloat(r[22]) + parseFloat(r[23])
		outbounds := atoi(r[9]) + atoi(r[10])
		checkouts := atoi(r[14])

		engagementRate := parseFloat(r[6])
		if engagementRate > 1 || engagementRate < 0 {
			log.Printf("EngagementRatePINS out of percent range: %v", engagementRate)
		}
		calculatedEngagementRate := (parseFloat(r[18]) + parseFloat(r[19])) / impressions
		if parseFloat(r[6]) != calculatedEngagementRate {
			log.Printf("EngagementRatePINS (%s) != calculated (%v)", r[6], calculatedEngagementRate)
		}

		ctr := clicks / impressions
		if ctr > 1 || ctr < 0 {
			log.Printf("CTRPINS out of percent range: %v", ctr)
		}
		if parseFloat(r[7]) != ctr {
			log.Printf("CTRPINS (%s) != calculated (%v)", r[7], ctr)
		}

		clickRatio := JSONFloat(outbounds) / clicks
		if clickRatio > 1 || clickRatio < 0 {
			log.Printf("ClickRatioPINS out of percent range: %v", ctr)
		}
		if math.IsNaN(float64(clickRatio)) {
			clickRatio = 0
		}

		conversionRate := float64(checkouts) / float64(outbounds)
		if math.IsNaN(conversionRate) {
			conversionRate = 0
		}

		countAtcs := atoi(r[26]) + atoi(r[16])
		valueAtcs := parseFloat(r[17]) + parseFloat(r[27])

		spend := parseFloat(r[3])
		calculatedCpa := spend / JSONFloat(checkouts)
		cpa := parseFloat(r[24])
		if cpa != calculatedCpa && checkouts != 0 {
			log.Printf("Cpa (%v) != calculatedCpa(%v)", cpa, calculatedCpa)
		}

		// TODO Take another look at mapping the JSON from the non-CSV endpoint
		rec := record{Fields: fields{
			ReportDate:         formatDaysAgo(0),
			Start:              a.start,
			End:                a.end,
			CampaignID:         r[0],
			CampaignName:       r[1],
			CampaignStatus:     r[2],
			DailyBudget:        parseFloat(r[11]),
			Spend:              spend,
			Impressions:        atoi(r[4]),
			Cpm:                parseFloat(r[5]),
			EngagementRatePINS: parseFloat(r[6]),
			CTRPINS:            parseFloat(r[7]),
			OutboundClicks:     outbounds,
			Cpc:                parseFloat(r[8]),
			ClickRatioPINS:     clickRatio,
			Checkouts:          checkouts,
			OrderValueCheckout: parseFloat(r[15]),
			ConversionRatePINS: JSONFloat(conversionRate),
			Cpa:                cpa,
			Roas:               parseFloat(r[13]),
			ATCs:               countAtcs,
			OrderValueATCs:     valueAtcs,
			Reach:              parseFloat(r[28]),
			Frequency:          parseFloat(r[29]),
			Leads:              parseFloat(r[30]),
		}}
		recs.Records = append(recs.Records, rec)
	}
	recs.Typecast = true // handle INVALID_MULTIPLE_CHOICE_OPTIONS error
	rsj, err := json.Marshal(recs)
	if err != nil {
		panic(err)
	}
	return rsj
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return n
}

// removes scientific notation which Airtable doesn't accept
func parseFloat(s string) JSONFloat {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	if strings.ContainsAny(s, "Ee") {
		log.Printf("Scientific notation: %s; parsed: %v", s, n)
	}
	return JSONFloat(n)
}

type JSONFloat float64

func (j JSONFloat) MarshalJSON() ([]byte, error) {
	v := float64(j)
	if math.IsInf(v, 0) {
		return []byte("0"), nil
	}
	return json.Marshal(v) // marshal result as standard float64
}

func (j *JSONFloat) UnsmarshalJSON(v []byte) error {
	if s := string(v); s == "+" || s == "-" {
		// if +/- indiciates infinity
		if s == "+" {
			*j = JSONFloat(math.Inf(1))
			return nil
		}
		*j = JSONFloat(math.Inf(-1))
		return nil
	}
	// just a regular float value
	var fv float64
	if err := json.Unmarshal(v, &fv); err != nil {
		return err
	}
	*j = JSONFloat(fv)
	return nil
}
