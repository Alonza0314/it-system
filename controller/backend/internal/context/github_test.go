package context

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/free5gc/amf/pulls/242":
			_, _ = w.Write([]byte(`{"number":242,"title":"open PR","state":"open","merged_at":null}`))
		case "/repos/free5gc/amf/pulls/100":
			_, _ = w.Write([]byte(`{"number":100,"title":"merged PR","state":"closed","merged_at":"2026-01-02T03:04:05Z"}`))
		case "/repos/free5gc/amf/pulls/101":
			_, _ = w.Write([]byte(`{"number":101,"title":"closed PR","state":"closed","merged_at":null}`))
		case "/repos/free5gc/amf/pulls/403":
			w.WriteHeader(http.StatusForbidden)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ctx := &githubContext{pullsApiUrl: server.URL + "/repos/free5gc/%s/pulls"}

	for _, tc := range []struct {
		number        int
		expectedTitle string
		expectedState string
	}{
		{242, "open PR", "open"},
		{100, "merged PR", "merged"},
		{101, "closed PR", "closed"},
	} {
		p, err := ctx.getPr("amf", tc.number)
		if err != nil || p == nil {
			t.Fatalf("PR #%d: expected a PR, got %v, error %v", tc.number, p, err)
		}
		if p.Number() != tc.number || p.Title() != tc.expectedTitle || p.State() != tc.expectedState {
			t.Errorf("PR #%d: got number %d, title %q, state %q", tc.number, p.Number(), p.Title(), p.State())
		}
	}

	if p, err := ctx.getPr("amf", 99999); p != nil || err != nil {
		t.Errorf("Expected no PR and no error for a missing PR, got %v, %v", p, err)
	}

	if _, err := ctx.getPr("amf", 403); err == nil {
		t.Errorf("Expected an error when GitHub rejects the request")
	}
}
