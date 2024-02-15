package airpin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

func init() {
	functions.HTTP("exchangeCodeForTokens", exchangeCodeForTokens)
	functions.HTTP("saveTokens", saveTokens)
	functions.CloudEvent("refreshAccessToken", refreshAccessToken)
}

type response struct {
	AccessToken  string `json:"access_token"`
	ResponseType string `json:"response_type"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// A job must be scheduled in Google Cloud Scheduler to run this every 30 days,
// which is the expiry period set by Pinterest.  Google Cloud Scheduler uses a
// crontab format, which is stateless, so it's not possible to configure it to
// execute "every 30 days".  Instead, "0 0 */30 * *" will run on the first of
// each month, as well as the 31st of each month, so there's an unnecessary
// extra refresh in months that have more than 30 days.
func refreshAccessToken(ctx context.Context, e event.Event) error {
	smc, err := secretmanager.NewClient(ctx)
	if err != nil {
		panic(err)
	}
	defer smc.Close()
	rt := getRefreshToken(ctx, smc)
	uri := "https://api.pinterest.com/v5/oauth/token"
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", rt)
	data.Set("scopes", "ads:read,pins:read,user_accounts:read")
	req, err := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	b64client := getB64Client(ctx, smc)
	req.Header.Add("Authorization", "Basic "+b64client)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != http.StatusOK {
		panic(res.Status + ": " + string(body))
	}
	var r response
	if err = json.Unmarshal(body, &r); err != nil {
		panic(err)
	}
	saveAccessToken(ctx, smc, r.AccessToken)
	// TODO disable last access token
	return nil
}

func getB64Client(ctx context.Context, client *secretmanager.Client) string {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: "projects/YOUR_PROJECT/secrets/b64client/versions/latest",
	}
	res, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		panic(err)
	}
	return string(res.Payload.Data)
}

func exchangeCodeForTokens(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	smc, err := secretmanager.NewClient(ctx)
	if err != nil {
		panic(err)
	}
	defer smc.Close()
	code := r.URL.Query().Get("code")
	uri := "https://api.pinterest.com/v5/oauth/token"
	redirectUri := "https://YOUR_PROJECT_SUBDOMAIN.cloudfunctions.net/saveTokens"
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectUri)
	req, err := http.NewRequest("POST", uri, strings.NewReader(data.Encode()))
	if err != nil {
		panic(err)
	}
	b64client := getB64Client(ctx, smc)
	req.Header.Add("Authorization", "Basic "+b64client)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != http.StatusOK {
		panic(res.Status + ": " + string(body))
	}
	var d response
	if err = json.Unmarshal(body, &d); err != nil {
		panic(err)
	}

}

func saveTokens(w http.ResponseWriter, r *http.Request) {

}

func getRefreshToken(ctx context.Context, client *secretmanager.Client) string {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: "projects/YOUR_PROJECT/secrets/pinterest-refresh-token/versions/latest",
	}
	res, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		panic(err)
	}
	return string(res.Payload.Data)
}

func saveAccessToken(ctx context.Context, client *secretmanager.Client, token string) {
	req := &secretmanagerpb.AddSecretVersionRequest{
		Parent: "projects/YOUR_PROJECT/secrets/pinterest-token",
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(token),
		},
	}
	if _, err := client.AddSecretVersion(ctx, req); err != nil {
		panic(err)
	}
}
