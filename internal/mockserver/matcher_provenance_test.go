package mockserver

import (
	"net/http/httptest"
	"testing"
)

func provenanceFixture(name, endpoint string, query map[string]string, ported bool) *MockResponse {
	r := &MockResponse{
		Metadata:   ResponseMetadata{Endpoint: endpoint, Method: "GET", Version: "2"},
		Request:    RequestSpec{QueryParams: query},
		Response:   ResponseSpec{StatusCode: 200},
		SourceFile: name,
	}
	if ported {
		r.Metadata.Provenance = &FixtureProvenance{Class: PortedFrom262}
	}
	return r
}

func TestMatchPreferring_PortNeverBeatsGenuine(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/mdm/profiles/search?page=1&pagesize=500", nil)
	req.Header.Set("Accept", "application/json;version=2")

	genuine := provenanceFixture("genuine.json", "/api/mdm/profiles/search", nil, false)
	// The port matches the query params too, so it outscores the genuine one.
	port := provenanceFixture("port.json", "/api/mdm/profiles/search", map[string]string{"page": "1", "pagesize": "500"}, true)

	if got := NewRequestMatcher([]*MockResponse{port, genuine}).Match(req); got != genuine {
		t.Errorf("genuine vs higher-scoring port: got %v, want the genuine fixture", got.SourceFile)
	}
	if got := NewRequestMatcher([]*MockResponse{port}).Match(req); got != port {
		t.Errorf("sole port: got %v, want the port served", got)
	}
	if got := NewRequestMatcher([]*MockResponse{genuine, port}).MatchPreferring(req, "port.json"); got != port {
		t.Errorf("preferred port: got %v, want the preferred port", got.SourceFile)
	}
	// A port whose exact literal path is more specific than the genuine
	// template still wins (e.g. a captured not-found for one id).
	byID := provenanceFixture("by-id.json", "/api/mdm/profiles/{id}", nil, false)
	literalPort := provenanceFixture("literal-port.json", "/api/mdm/profiles/68990", nil, true)
	idReq := httptest.NewRequest("GET", "/api/mdm/profiles/68990", nil)
	idReq.Header.Set("Accept", "application/json;version=2")
	if got := NewRequestMatcher([]*MockResponse{byID, literalPort}).Match(idReq); got != literalPort {
		t.Errorf("literal-path port vs genuine template: got %v, want the literal port", got.SourceFile)
	}
	// A genuine literal beats a port literal for the same path.
	genuineLiteral := provenanceFixture("genuine-literal.json", "/api/mdm/profiles/68990", map[string]string{}, false)
	if got := NewRequestMatcher([]*MockResponse{literalPort, genuineLiteral}).Match(idReq); got != genuineLiteral {
		t.Errorf("genuine literal vs port literal: got %v, want the genuine literal", got.SourceFile)
	}
	other := provenanceFixture("other.json", "/api/mdm/other", nil, false)
	if got := NewRequestMatcher([]*MockResponse{other, port}).Match(req); got != port {
		t.Errorf("non-matching genuine plus port: got %v, want the port", got)
	}
	if got := NewRequestMatcher([]*MockResponse{other}).Match(req); got != nil {
		t.Errorf("no match: got %v, want nil", got.SourceFile)
	}
}
