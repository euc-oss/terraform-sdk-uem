// custom-http-client demonstrates injecting a custom *http.Client.
//
// Use cases:
//   - Routing requests through a corporate proxy (HTTPS_PROXY env var)
//   - Adding a custom RootCAs pool for self-signed UEM on-prem certificates
//   - Wrapping the transport for telemetry or logging
//
// The SDK applies its own per-request Timeout (30s by default) over any
// timeout you set on the supplied http.Client. Use wsone.Config.Timeout to
// configure the per-request budget instead.
//
// Required environment variables:
//
//	WSONE_API_URL       Base URL of the Workspace ONE UEM API
//	WSONE_TENANT_CODE   UEM tenant code
//	WSONE_CLIENT_ID     OAuth2 client ID
//	WSONE_CLIENT_SECRET OAuth2 client secret
//	WSONE_TOKEN_URL     OAuth2 token endpoint URL
//	HTTPS_PROXY         (optional) proxy URL, e.g. http://proxy.corp.local:8080
//
// Run:
//
//	go run .
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	wsone "github.com/euc-oss/terraform-sdk-uem"
)

func main() {
	for _, k := range []string{"WSONE_API_URL", "WSONE_TENANT_CODE", "WSONE_CLIENT_ID", "WSONE_CLIENT_SECRET", "WSONE_TOKEN_URL"} {
		if os.Getenv(k) == "" {
			log.Fatalf("missing required env var %s", k)
		}
	}

	transport := &http.Transport{}
	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		u, err := url.Parse(proxy)
		if err != nil {
			log.Fatalf("invalid HTTPS_PROXY: %v", err)
		}
		transport.Proxy = http.ProxyURL(u)
		fmt.Printf("Using proxy: %s\n", proxy)
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	auth, err := wsone.NewOAuth2Auth(wsone.OAuth2Config{
		ClientID:     os.Getenv("WSONE_CLIENT_ID"),
		ClientSecret: os.Getenv("WSONE_CLIENT_SECRET"),
		TokenURL:     os.Getenv("WSONE_TOKEN_URL"),
	})
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	client, err := wsone.NewClient(wsone.Config{
		BaseURL:    os.Getenv("WSONE_API_URL"),
		TenantCode: os.Getenv("WSONE_TENANT_CODE"),
		Auth:       auth,
		HTTPClient: httpClient,
		Timeout:    60 * time.Second,
	})
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	profiles, err := wsone.ListProfiles(ctx, client, nil)
	if err != nil {
		log.Fatalf("list profiles: %v", err)
	}
	fmt.Printf("Custom HTTP client succeeded; %d profiles visible\n", len(profiles))
}
