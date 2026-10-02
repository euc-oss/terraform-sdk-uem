// Package resources contains hand-coded SDK surfaces for workflows that
// codegen cannot express as a single HTTP call.
//
// BaselineCreator is the multipart-create helper for the Baselines V1
// API. It exists because POST /api/mdm/groups/{ogUuid}/baselines
// accepts a multipart/form-data body whose Content-Type carries a
// dynamically-generated boundary string. The SDK's codegen template
// hardcodes a per-operation Content-Type at generation time, so a
// generated method cannot accept the boundary at call time. This
// helper is the smallest-blast-radius fallback to the codegen
// multipart template extension.
//
// Retirement criterion: when the codegen template learns to accept a
// runtime Content-Type override (allowing a multipart boundary to be
// inserted per-call), this helper can be retired. Callers would then
// use a generated CreateBaselineAsync method directly. Until then,
// this helper is the sanctioned path.
//
// All other Baselines operations (Get, List, Update, Delete, Clone,
// Status, Assignments, Devices, DevicePolicies) are pure codegen-emitted
// methods on sdk.BaselinesV1Service. Use this helper ONLY for creating
// a new baseline.
package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/euc-oss/terraform-sdk-uem/client"
	mdmv1 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v1"
)

// baselineCreateEndpoint is the canonical multipart-create path for the
// Baselines V1 API. The single occurrence here keeps the helper's
// reference clear; the rest of the Baselines surface uses the generated
// service's hardcoded endpoint constants.
const baselineCreateEndpoint = "/api/mdm/groups/%s/baselines"

// BaselineCreator assembles the multipart/form-data body required by
// POST /api/mdm/groups/{organizationGroupUuid}/baselines and submits
// it via the SDK client. The endpoint takes a JSON-serialized
// CreateBaselineRequestV1Model in form field "baseline" plus an
// optional zip file in form field "customFile".
type BaselineCreator struct {
	client *client.Client
}

// NewBaselineCreator constructs a BaselineCreator. Pass the same
// client.Client used by the rest of the SDK.
func NewBaselineCreator(c *client.Client) *BaselineCreator {
	return &BaselineCreator{client: c}
}

// Create assembles the multipart body, submits the request, and
// returns the new baseline plus the response headers. Pass a nil
// customFile if no custom-template zip payload is being uploaded.
func (b *BaselineCreator) Create(
	ctx context.Context,
	organizationGroupUuid string,
	body *mdmv1.CreateBaselineRequestV1Model,
	customFile io.Reader,
) (http.Header, *mdmv1.BaselineV1Model, error) {
	if organizationGroupUuid == "" {
		return nil, nil, fmt.Errorf("BaselineCreator.Create: organizationGroupUuid must not be empty")
	}
	if body == nil {
		return nil, nil, fmt.Errorf("BaselineCreator.Create: body must not be nil")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("BaselineCreator.Create: marshal body: %w", err)
	}
	if err := w.WriteField("baseline", string(bodyJSON)); err != nil {
		return nil, nil, fmt.Errorf("BaselineCreator.Create: write baseline field: %w", err)
	}

	if customFile != nil {
		part, err := w.CreateFormFile("customFile", "baseline.zip")
		if err != nil {
			return nil, nil, fmt.Errorf("BaselineCreator.Create: create customFile part: %w", err)
		}
		if _, err := io.Copy(part, customFile); err != nil {
			return nil, nil, fmt.Errorf("BaselineCreator.Create: copy customFile: %w", err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, nil, fmt.Errorf("BaselineCreator.Create: close multipart writer: %w", err)
	}

	endpoint := fmt.Sprintf(baselineCreateEndpoint, organizationGroupUuid)
	var response mdmv1.BaselineV1Model
	headers, err := b.client.DoRequest(
		ctx,
		http.MethodPost,
		endpoint,
		mdmv1.AcceptHeader,
		w.FormDataContentType(),
		buf.Bytes(),
		&response,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("BaselineCreator.Create: %w", err)
	}
	return headers, &response, nil
}
