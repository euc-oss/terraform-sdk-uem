// smart-groups lists smart groups available in the tenant.
//
// Note: at this SDK version, Smart Groups support is read-only via Search.
// Full CRUD is on the roadmap.
//
// The SmartGroupsService.Search method returns a SmartGroupSearchResultV1 with
// a SmartGroups []SmartGroupSearchModelV1 field. Each entry carries SmartGroupID,
// Name, ManagedByOrganizationGroupName, Devices, Assignments, and Exclusions.
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pageSize := 25
	svc := wsone.NewSmartGroupsService(client)
	_, resp, err := svc.Search(ctx, &wsone.SmartGroupsSearchOptions{
		PageSize: &pageSize,
	})
	if err != nil {
		log.Fatalf("search smart groups: %v", err)
	}

	total := 0
	if resp.Total != nil {
		total = *resp.Total
	}
	fmt.Printf("Found %d smart groups (total=%d):\n", len(resp.SmartGroups), total)
	for _, g := range resp.SmartGroups {
		id := 0
		if g.SmartGroupID != nil {
			id = *g.SmartGroupID
		}
		fmt.Printf("  %d  %s  (managed by: %s)\n", id, g.Name, g.ManagedByOrganizationGroupName)
	}
}
