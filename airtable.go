package airpin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Airtable struct {
	token  string
	client *http.Client
	base   string
	period Period
}

// Mapping from Pinterest Ad Account ID to Airtable Base ID
var bases = map[string]string{
	// "5497560xxxxx": "appa7SRvmSAhxxxxx", // TEST
	// "5497633xxxxx": "appR9q0V06C8xxxxx", // TEST
}

type records struct {
	Records  [1]record `json:"records"`
	Typecast bool      `json:"typecast"`
}

type record struct {
	ID     string `json:"id,omitempty"`
	Fields fields `json:"fields,omitempty"`
}

type fields struct {
	ReportDate     string    `json:"Report Date"`
	Start          string    `json:"Start"`
	End            string    `json:"End"`
	CampaignID     string    `json:"Campaign ID"`
	CampaignName   string    `json:"Campaign Name"`
	CampaignStatus string    `json:"Campaign Status"`
	Budget         JSONFloat `json:"Daily budget"`
	Spend          JSONFloat `json:"Spend"`
	Reach          JSONFloat `json:"Reach"`
	Freq           JSONFloat `json:"Freq"`
	IM             int       `json:"IM"`
	IM1            int       `json:"IM1"`
	IM2            int       `json:"IM2"`
	CPM            JSONFloat `json:"CPM"`
	PC             int       `json:"PC"`
	PC1            int       `json:"PC1"`
	PC2            int       `json:"PC2"`
	CTR            JSONFloat `json:"CTR"`
	CPC            JSONFloat `json:"CPC"`
	SV             int       `json:"SV"`
	SV1            int       `json:"SV1"`
	SV2            int       `json:"SV2"`
	SVR            JSONFloat `json:"SVR"`
	SVR1           JSONFloat `json:"SVR1"`
	SVR2           JSONFloat `json:"SVR2"`
	NGR            JSONFloat `json:"NGR"`
	NGR1           JSONFloat `json:"NGR1"`
	NG             int       `json:"NG"`
	NG1            int       `json:"NG1"`
	NG2            int       `json:"NG2"`
	OC             int       `json:"OC"`
	OC1            int       `json:"OC1"`
	OC2            int       `json:"OC2"`
	CTPV           int       `json:"CTPV"`
	NGPV           int       `json:"NGPV"`
	VTPV           int       `json:"VTPV"`
	ATC            int       `json:"ATC"`
	ATCV           JSONFloat `json:"ATCV"`
	Checkouts      int       `json:"Checkouts"`
	CTC            int       `json:"CTC"`
	NGC            int       `json:"NGC"`
	VTC            int       `json:"VTC"`
	Value          JSONFloat `json:"Value"`
	CTV            JSONFloat `json:"CTV"`
	NGV            JSONFloat `json:"NGV"`
	VTV            JSONFloat `json:"VTV"`
	CPA            JSONFloat `json:"CPA"`
	ROAS           JSONFloat `json:"ROAS"`
	Leads          int       `json:"Leads"`
	CR             JSONFloat `json:"CR"`
	ConvRate       JSONFloat `json:"Conv Rate"`
	OCPC           JSONFloat `json:"oCPC"`
}

func NewAirtable(token string, period Period) *Airtable {
	base := "https://api.airtable.com/v0/"
	return &Airtable{token, &http.Client{}, base, period}
}

func (a Airtable) Update(account_id string, rows [][]string) {
	var start_date, end_date string
	switch a.period {
	case Week:
		start_date = formatDaysAgo(7)
		end_date = formatDaysAgo(1)
	case Month:
		y, m, _ := time.Now().Date()
		_, m, d := time.Date(y, m, 0, 0, 0, 0, 0, time.UTC).Date()
		start_date = fmt.Sprintf("%v-%02d-01", y, int(m))
		end_date = fmt.Sprintf("%v-%02d-%v", y, int(m), d)
	}
	if baseId := bases[account_id]; baseId != "" {
		for _, row := range rows[1:] {
			// These metrics are in v4 of the Pinterest API but not v5 so
			// we have to calculate them here.
			total_impression := atoi(row[7])
			total_outbound_click := atoi(row[25]) + atoi(row[26])
			total_clickthrough := atoi(row[11])
			click_ratio := float64(total_outbound_click) / float64(total_clickthrough)
			total_checkout := atoi(row[34])
			conversion_rate := float64(total_checkout) / float64(total_impression)
			outbound_cost_per_click := float64(parseFloat(row[4])) / float64(total_outbound_click)
			rec := records{Records: [1]record{{Fields: fields{
				// The comments in ALL_CAPS are the Pinterest internal names
				// which appear in JSON returned by the Pinterest API.  When
				// a metric does not exist in the v5 API, the comment shows
				// how to calculate it.
				ReportDate:     formatDaysAgo(0),
				Start:          start_date,
				End:            end_date,
				CampaignID:     row[0],                                                                                // CAMPAIGN_ID
				CampaignName:   row[1],                                                                                // CAMPAIGN_NAME
				CampaignStatus: row[2],                                                                                // CAMPAIGN_STATUS
				Budget:         parseFloat(row[3]),                                                                    // CAMPAIGN_DAILY_SPEND_CAP
				Spend:          parseFloat(row[4]),                                                                    // SPEND_IN_MICRO_DOLLAR
				Reach:          parseFloat(row[5]),                                                                    // TOTAL_IMPRESSION_USER
				Freq:           parseFloat(row[6]),                                                                    // TOTAL_IMPRESSION_FREQUENCY
				IM:             total_impression,                                                                      // TOTAL_IMPRESSION
				IM1:            atoi(row[8]),                                                                          // IMPRESSION_1
				IM2:            atoi(row[9]),                                                                          // IMPRESSION_2
				CPM:            parseFloat(row[10]),                                                                   // CPM_IN_DOLLAR
				PC:             total_clickthrough,                                                                    // TOTAL_CLICKTHROUGH
				PC1:            atoi(row[12]),                                                                         // CLICKTHROUGH_1
				PC2:            atoi(row[13]),                                                                         // CLICKTHROUGH_2
				CTR:            parseFloat(row[14]),                                                                   // ECTR
				CPC:            parseFloat(row[15]),                                                                   // ECPC_IN_DOLLAR
				SV:             int(parseFloat(row[7]) * parseFloat(row[18])),                                         // TOTAL_REPIN = TOTAL_REPIN_RATE * TOTAL_IMPRESSION,
				SV1:            atoi(row[16]),                                                                         // REPIN_1
				SV2:            atoi(row[17]),                                                                         // REPIN_2
				SVR:            parseFloat(row[18]),                                                                   // TOTAL_REPIN_RATE
				SVR1:           parseFloat(row[19]),                                                                   // REPIN_RATE
				SVR2:           parseFloat(row[17]) / parseFloat(row[7]),                                              // REPIN_RATE_2 = REPIN_2 / TOTAL_IMPRESSION
				NGR:            parseFloat(row[20]),                                                                   // EENGAGEMENT_RATE
				NGR1:           parseFloat(row[21]),                                                                   // ENGAGEMENT_RATE
				NG:             atoi(row[22]),                                                                         // TOTAL_ENGAGEMENT
				NG1:            atoi(row[23]),                                                                         // ENGAGEMENT_1
				NG2:            atoi(row[24]),                                                                         // ENGAGEMENT_2
				OC:             total_outbound_click,                                                                  // TOTAL_OUTBOUND_CLICK = OUTBOUND_CLICK_1 + OUTBOUND_CLICK_2
				OC1:            atoi(row[25]),                                                                         // OUTBOUND_CLICK_1
				OC2:            atoi(row[26]),                                                                         // OUTBOUND_CLICK_2
				CTPV:           atoi(row[27]),                                                                         // TOTAL_CLICK_PAGE_VISIT
				NGPV:           atoi(row[28]),                                                                         // TOTAL_ENGAGEMENT_PAGE_VISIT
				VTPV:           atoi(row[29]),                                                                         // TOTAL_VIEW_PAGE_VISIT
				ATC:            int(parseFloat(row[30]) * parseFloat(row[7])),                                         // TOTAL_ADD_TO_CART = TOTAL_ADD_TO_CART_CONVERSION_RATE * TOTAL_IMPRESSION
				ATCV:           parseFloat(row[31]) + parseFloat(row[32]) + parseFloat(row[33]),                       // TOTAL_ADD_TO_CART_VALUE_IN_MICRO_DOLLAR = TOTAL_CLICK_ADD_TO_CART_VALUE_IN_MICRO_DOLLAR + TOTAL_ENGAGEMENT_ADD_TO_CART_VALUE_IN_MICRO_DOLLAR + TOTAL_VIEW_ADD_TO_CART_VALUE_IN_MICRO_DOLLAR
				Checkouts:      total_checkout,                                                                        // TOTAL_CHECKOUT
				CTC:            atoi(row[35]),                                                                         // TOTAL_CLICK_CHECKOUT
				NGC:            atoi(row[36]),                                                                         // TOTAL_ENGAGEMENT_CHECKOUT
				VTC:            atoi(row[37]),                                                                         // TOTAL_VIEW_CHECKOUT
				Value:          parseFloat(row[38]),                                                                   // TOTAL_CHECKOUT_VALUE_IN_MICRO_DOLLAR
				CTV:            parseFloat(row[39]),                                                                   // TOTAL_CLICK_CHECKOUT_VALUE_IN_MICRO_DOLLAR
				NGV:            parseFloat(row[40]),                                                                   // TOTAL_ENGAGEMENT_CHECKOUT_VALUE_IN_MICRO_DOLLAR
				VTV:            parseFloat(row[41]),                                                                   // TOTAL_VIEW_CHECKOUT_VALUE_IN_MICRO_DOLLAR
				CPA:            parseFloat(row[42]) + parseFloat(row[43]) + parseFloat(row[44]) + parseFloat(row[45]), // CHECKOUT_COST_PER_ACTION = INAPP_CHECKOUT_COST_PER_ACTION + OFFLINE_CHECKOUT_COST_PER_ACTION + PINTEREST_CHECKOUT_COST_PER_ACTION + WEB_CHECKOUT_COST_PER_ACTION
				ROAS:           parseFloat(row[46]),                                                                   // CHECKOUT_ROAS
				Leads:          atoi(row[47]),                                                                         // TOTAL_LEAD
				CR:             JSONFloat(click_ratio / 100),                                                          // TOTAL_OUTBOUND_CLICK / TOTAL_CLICKTHROUGH
				ConvRate:       JSONFloat(conversion_rate),                                                            // TOTAL_OUTBOUND_CLICK / TOTAL_CHECKOUT
				OCPC:           JSONFloat(outbound_cost_per_click),                                                    // SPEND_IN_MICRO_DOLLAR / TOTAL_OUTBOUND_CLICK
			}}}, Typecast: true}
			recJson, err := json.MarshalIndent(rec, "", "  ")
			if err != nil {
				panic(err)
			}
			// DEBUG (do not delete; check in to repo)
			// fmt.Println(string(recJson))

			a.post(baseId, recJson)

			// DEBUG (do not delete; check in to repo)
			// os.Exit(0)
		}
	}
}

func (a Airtable) post(baseId string, record []byte) {
	url := a.base + baseId + "/" + url.PathEscape("Ad KPIs")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(record))
	req.Header.Add("Authorization", "Bearer "+a.token)
	req.Header.Add("Content-Type", "application/json")
	res, err := a.client.Do(req)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		panic("baseId: " + baseId + ", " + res.Status + ": " + string(body))
	}
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return n
}

func parseFloat(s string) JSONFloat {
	// Remove scientific notation which Airtable doesn't accept
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return JSONFloat(n)
}

type JSONFloat float64

func (j JSONFloat) MarshalJSON() ([]byte, error) {
	v := float64(j)
	// Airtable doesn't accept infinity
	if math.IsInf(v, 0) {
		return []byte("0"), nil
	}
	return json.Marshal(v)
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

func formatDaysAgo(days int) string {
	const YYYYMMDD = "2006-01-02"
	now := time.Now()
	ago := now.AddDate(0, 0, -days)
	return ago.Format(YYYYMMDD)
}
