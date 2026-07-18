package client

import (
	"net/url"
	"strings"
	"testing"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/config"
)

func TestBuildAuthorizeURL_ThailandRegionalEndpoint(t *testing.T) {
	client := NewLazadaClient(&config.Config{
		LazadaRedirectURI: "https://dev-laco-invoice.laconic.co.th/stores/lazada/callback",
		LazadaCountry:     "th",
	}, 5).(*lazadaClientImpl)

	creds := Credentials{AppKey: "139411", AppSecret: "secret"}
	got := client.BuildAuthorizeURL(creds, "state-123")
	if !strings.HasPrefix(got, "https://api.lazada.co.th/oauth/authorize?") {
		t.Fatalf("expected Thailand regional authorize URL, got %q", got)
	}

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	query := parsed.Query()
	if query.Get("response_type") != "code" {
		t.Fatalf("response_type = %q", query.Get("response_type"))
	}
	if query.Get("client_id") != "139411" {
		t.Fatalf("client_id = %q", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "https://dev-laco-invoice.laconic.co.th/stores/lazada/callback" {
		t.Fatalf("redirect_uri = %q", query.Get("redirect_uri"))
	}
	if query.Get("country") != "th" {
		t.Fatalf("country = %q", query.Get("country"))
	}
	if query.Get("force_auth") != "" {
		t.Fatalf("force_auth should be omitted by default, got %q", query.Get("force_auth"))
	}
	if query.Get("state") != "state-123" {
		t.Fatalf("state = %q", query.Get("state"))
	}
}

func TestBuildAuthorizeURL_DefaultCentralEndpointWithForceAuth(t *testing.T) {
	client := NewLazadaClient(&config.Config{
		LazadaRedirectURI: "https://example.com/callback",
		LazadaForceAuth:   true,
	}, 5).(*lazadaClientImpl)

	creds := Credentials{AppKey: "139411", AppSecret: "secret"}
	got := client.BuildAuthorizeURL(creds, "")
	if !strings.HasPrefix(got, defaultAuthURL+"?") {
		t.Fatalf("expected central authorize URL, got %q", got)
	}

	query, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if query.Query().Get("force_auth") != "true" {
		t.Fatalf("force_auth = %q", query.Query().Get("force_auth"))
	}
}

func TestResolveAuthorizeBaseURL_ExplicitOverride(t *testing.T) {
	got := resolveAuthorizeBaseURL("https://custom.example/oauth/authorize", "th")
	if got != "https://custom.example/oauth/authorize" {
		t.Fatalf("got %q", got)
	}
}
