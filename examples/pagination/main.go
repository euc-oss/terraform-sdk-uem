// pagination shows how to iterate through paginated list responses.
//
// The Workspace ONE UEM API returns up to 500 records per page by default.
// For tenants with more than 500 profiles you must page explicitly. This
// example uses wsone.ListOptions to control pagination and walks through
// every page until no more records are returned.
//
// Required environment variables:
//
//	WSONE_API_URL       Base URL of the Workspace ONE UEM API
//	WSONE_TENANT_CODE   UEM tenant code
//	WSONE_CLIENT_ID     OAuth2 client ID
//	WSONE_CLIENT_SECRET OAuth2 client secret
//	WSONE_TOKEN_URL     OAuth2 token endpoint URL
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
	for _, k := range []string{"WSONE_API_URL", "WSONE_TENANT_CODE", "WSONE_CLIENT_ID", "WSONE_CLIENT_SECRET", "WSONE_TOKEN_URL"} {
		if os.Getenv(k) == "" {
			log.Fatalf("missing required env var %s", k)
		}
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
	})
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const pageSize = 100
	page := 0
	total := 0
	for {
		opts := &wsone.ListOptions{
			Page:     page,
			PageSize: pageSize,
		}
		profiles, err := wsone.ListProfiles(ctx, client, opts)
		if err != nil {
			log.Fatalf("list page %d: %v", page, err)
		}
		fmt.Printf("Page %d: %d profiles\n", page, len(profiles))
		total += len(profiles)

		// Last page when fewer than pageSize records returned.
		if len(profiles) < pageSize {
			break
		}
		page++
	}
	fmt.Printf("Total profiles seen across pages: %d\n", total)
}
