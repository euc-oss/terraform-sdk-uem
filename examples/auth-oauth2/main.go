// auth-oauth2 demonstrates OAuth2 authentication for both cloud and on-premises Workspace ONE UEM deployments.
//
// Cloud:    Token URL is https://na.uemauth.workspaceone.com/connect/token (a separate auth host).
// On-prem:  Token URL is typically https://<your-host>/oauth/token.
//
// If WSONE_TOKEN_URL is not set, this example prints the cloud-default value and uses it.
//
// Required environment variables:
//
//	WSONE_API_URL       Base URL of the Workspace ONE UEM API
//	WSONE_TENANT_CODE   UEM tenant code
//	WSONE_CLIENT_ID     OAuth2 client ID
//	WSONE_CLIENT_SECRET OAuth2 client secret
//	WSONE_TOKEN_URL     (optional) override the default token endpoint
//
// Run:
//
//	go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	wsone "github.com/euc-oss/terraform-sdk-uem"
)

func main() {
	for _, k := range []string{"WSONE_API_URL", "WSONE_TENANT_CODE", "WSONE_CLIENT_ID", "WSONE_CLIENT_SECRET"} {
		if os.Getenv(k) == "" {
			log.Fatalf("missing required env var %s", k)
		}
	}

	tokenURL := os.Getenv("WSONE_TOKEN_URL")
	if tokenURL == "" {
		tokenURL = "https://na.uemauth.workspaceone.com/connect/token"
		fmt.Printf("WSONE_TOKEN_URL not set; using cloud default: %s\n", tokenURL)
	}

	auth, err := wsone.NewOAuth2Auth(wsone.OAuth2Config{
		ClientID:     os.Getenv("WSONE_CLIENT_ID"),
		ClientSecret: os.Getenv("WSONE_CLIENT_SECRET"),
		TokenURL:     tokenURL,
	})
	if err != nil {
		log.Fatalf("auth: %v", err)
	}

	client, err := wsone.NewClient(wsone.Config{
		BaseURL:    os.Getenv("WSONE_API_URL"),
		TenantCode: os.Getenv("WSONE_TENANT_CODE"),
		Auth:       auth,
	})
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	profiles, err := wsone.ListProfiles(ctx, client, nil)
	if err != nil {
		log.Fatalf("list profiles: %v", err)
	}
	fmt.Printf("OAuth2 succeeded; %d profiles visible\n", len(profiles))
}
