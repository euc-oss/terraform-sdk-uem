// auth-basic demonstrates Basic authentication.
//
// Basic auth sends username + password on every request; the tenant code goes
// on the Config (not the auth constructor). Use Basic only for legacy on-prem
// deployments where OAuth2 is not configured. OAuth2 is the recommended
// provider for new integrations.
//
// Required environment variables:
//
//	WSONE_API_URL     Base URL of the Workspace ONE UEM API
//	WSONE_TENANT_CODE UEM tenant code
//	WSONE_USERNAME    UEM admin username
//	WSONE_PASSWORD    UEM admin password
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
	for _, k := range []string{"WSONE_API_URL", "WSONE_TENANT_CODE", "WSONE_USERNAME", "WSONE_PASSWORD"} {
		if os.Getenv(k) == "" {
			log.Fatalf("missing required env var %s", k)
		}
	}

	auth := wsone.NewBasicAuth(
		os.Getenv("WSONE_USERNAME"),
		os.Getenv("WSONE_PASSWORD"),
	)

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
	fmt.Printf("Basic auth succeeded; %d profiles visible\n", len(profiles))
}
