// error-handling demonstrates how to inspect SDK errors.
//
// API errors are returned as *wsone.APIError. You can type-assert with
// errors.As to inspect the HTTP status code, error message, and structured
// error code. The IsRetryable() method tells you whether the error is one
// the SDK's retry layer covers (429, 500, 502, 503, 504); a returned error
// with IsRetryable()==true means the SDK already retried and exhausted its
// budget.
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
	"errors"
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deliberately request a profile ID that does not exist to trigger an error.
	const nonExistentProfileID = -1
	_, err = wsone.GetProfile(ctx, client, nonExistentProfileID, wsone.PlatformAppleOsX)
	if err == nil {
		fmt.Println("Unexpected success — the SDK returned a profile for ID=-1.")
		return
	}

	var apiErr *wsone.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("APIError received:\n")
		fmt.Printf("  StatusCode: %d\n", apiErr.StatusCode)
		fmt.Printf("  Message:    %s\n", apiErr.Message)
		if apiErr.ErrorCode != "" {
			fmt.Printf("  ErrorCode:  %s\n", apiErr.ErrorCode)
		}
		fmt.Printf("  IsRetryable: %v\n", apiErr.IsRetryable())
		return
	}

	fmt.Printf("Non-API error: %v\n", err)
}
