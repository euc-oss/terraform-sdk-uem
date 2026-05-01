// profile-crud walks Create → Read → Update → Delete on a macOS profile.
//
// This example creates a real profile in your tenant, reads it, updates its
// description, and deletes it. If the program exits mid-flow with an error,
// you may need to clean up the partial profile manually via the UEM console.
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
	"github.com/euc-oss/terraform-sdk-uem/models"
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

	platform := wsone.PlatformAppleOsX
	name := fmt.Sprintf("example-crud-%d", time.Now().Unix())

	// Create
	createReq := &models.ProfileCreateRequest{
		"General": map[string]interface{}{
			"Name":        name,
			"Description": "Created by go-wsone-sdk profile-crud example",
		},
	}
	created, err := wsone.CreateProfile(ctx, client, platform, createReq)
	if err != nil {
		log.Fatalf("create: %v", err)
	}
	id := created.GetProfileID()
	fmt.Printf("Created profile id=%d name=%s\n", id, name)

	// Read
	got, err := wsone.GetProfile(ctx, client, id, platform)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("Read profile id=%d name=%s\n", got.GetProfileID(), got.GetName())

	// Update
	updateReq := &models.ProfileUpdateRequest{
		"General": map[string]interface{}{
			"ProfileId":        id,
			"Name":             name,
			"Description":      "Updated by go-wsone-sdk profile-crud example",
			"CreateNewVersion": false,
		},
	}
	if _, err := wsone.UpdateProfile(ctx, client, platform, id, updateReq); err != nil {
		log.Fatalf("update: %v", err)
	}
	fmt.Printf("Updated profile id=%d\n", id)

	// Delete
	if err := wsone.DeleteProfile(ctx, client, id); err != nil {
		log.Fatalf("delete: %v", err)
	}
	fmt.Printf("Deleted profile id=%d\n", id)
}
