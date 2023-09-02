package airpin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/caarlos0/env/v8"
	"github.com/cloudevents/sdk-go/v2/event"
)

// TODO just use Secret Manager
type config struct {
	AirtableToken  string `env:"AIRTABLE_TOKEN,required"`
	PinterestToken string `env:"PINTEREST_TOKEN,required"`
}

type data struct {
	Message struct {
		Data string
	}
}

func init() {
	functions.CloudEvent("airpin", airpin)
}

func airpin(ctx context.Context, e event.Event) error {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalln(err)
	}

	var d data
	json.Unmarshal(e.Data(), &d)
	data, err := base64.StdEncoding.DecodeString(d.Message.Data)
	if err != nil {
		panic(err)
	}
	var start, end string
	switch period := string(data); period {
	case "7-day":
		// TODO parameterize this in GCP
		start = formatDaysAgo(7)
		end = formatDaysAgo(1)
		// start = "2023-08-04"
		// end = "2023-08-10"
	case "month":
		y, m, _ := time.Now().Date()
		_, m, d := time.Date(y, m, 0, 0, 0, 0, 0, time.UTC).Date()
		start = fmt.Sprintf("%v-%02d-01", y, int(m))
		end = fmt.Sprintf("%v-%02d-%v", y, int(m), d)
	default:
		panic("invalid period")
	}
	pinterest := NewPinterest(cfg.PinterestToken, start, end)
	ars := pinterest.Reports()

	airtable := NewAirtable(cfg.AirtableToken, start, end)
	airtable.UpdateAll(ars)
	return nil
}

func formatDaysAgo(days int) string {
	const YYYYMMDD = "2006-01-02"
	now := time.Now()
	ago := now.AddDate(0, 0, -days)
	return ago.Format(YYYYMMDD)
}
