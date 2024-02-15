package airpin

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/caarlos0/env/v8"
	"github.com/cloudevents/sdk-go/v2/event"
)

type Credentials struct {
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
	period := getPeriod(e)
	creds := Credentials{}
	if err := env.Parse(&creds); err != nil {
		panic(err)
	}
	pin := NewPinterest(creds.PinterestToken, period)
	air := NewAirtable(creds.AirtableToken, period)

	for account_id := range bases {
		air.Update(account_id, pin.Reports(account_id))
	}

	return nil
}

func getPeriod(e event.Event) Period {
	var d data
	json.Unmarshal(e.Data(), &d)
	data, err := base64.StdEncoding.DecodeString(d.Message.Data)
	if err != nil {
		panic(err)
	}

	var period Period
	switch string(data) {
	case "7-day":
		period = Week
	case "month":
		period = Month
	default:
		panic("invalid period")
	}
	return period
}
