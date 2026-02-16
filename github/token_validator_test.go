package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateTokenScopes_ClassicPAT_AllScopes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-OAuth-Scopes", "repo, project")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts, err := validateTokenScopesWithURL(context.Background(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.HasRepo {
		t.Error("HasRepo = false, want true")
	}
	if !ts.HasProject {
		t.Error("HasProject = false, want true")
	}
	if !ts.IsClassicPAT {
		t.Error("IsClassicPAT = false, want true")
	}
	if ts.IsFineGrained {
		t.Error("IsFineGrained = true, want false")
	}
	if err := ts.Validate(); err != nil {
		t.Errorf("Validate() returned unexpected error: %v", err)
	}
}

func TestValidateTokenScopes_ClassicPAT_MissingProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-OAuth-Scopes", "repo")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts, err := validateTokenScopesWithURL(context.Background(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.HasRepo {
		t.Error("HasRepo = false, want true")
	}
	if ts.HasProject {
		t.Error("HasProject = true, want false")
	}

	missing := ts.MissingScopes()
	found := false
	for _, m := range missing {
		if m == "project or read:org" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("MissingScopes() = %v, want to contain 'project or read:org'", missing)
	}

	if err := ts.Validate(); err == nil {
		t.Error("Validate() returned nil, want error for missing project scope")
	}
}

func TestValidateTokenScopes_ClassicPAT_MissingRepo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-OAuth-Scopes", "project")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts, err := validateTokenScopesWithURL(context.Background(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	missing := ts.MissingScopes()
	found := false
	for _, m := range missing {
		if m == "repo" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("MissingScopes() = %v, want to contain 'repo'", missing)
	}

	if err := ts.Validate(); err == nil {
		t.Error("Validate() returned nil, want error for missing repo scope")
	}
}

func TestValidateTokenScopes_FineGrainedPAT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No X-OAuth-Scopes header — indicates a fine-grained PAT
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts, err := validateTokenScopesWithURL(context.Background(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.IsFineGrained {
		t.Error("IsFineGrained = false, want true")
	}
	if ts.IsClassicPAT {
		t.Error("IsClassicPAT = true, want false")
	}
	if err := ts.Validate(); err != nil {
		t.Errorf("Validate() returned unexpected error for fine-grained PAT: %v", err)
	}

	missing := ts.MissingScopes()
	if missing != nil {
		t.Errorf("MissingScopes() = %v, want nil for fine-grained PAT", missing)
	}
}

func TestValidateTokenScopes_InvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := validateTokenScopesWithURL(context.Background(), "bad-token", server.URL)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf("error = %q, want to contain 'HTTP 401'", err.Error())
	}
}

func TestValidateTokenScopes_RepoImpliesActions(t *testing.T) {
	// The repo scope grants access to Actions (GitHub bundles Actions under repo).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-OAuth-Scopes", "repo")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts, err := validateTokenScopesWithURL(context.Background(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.HasRepo {
		t.Error("HasRepo = false, want true — repo scope should grant Actions access")
	}
	if !ts.IsClassicPAT {
		t.Error("IsClassicPAT = false, want true")
	}
}

func TestValidateTokenScopes_ReadOrgImpliesProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-OAuth-Scopes", "repo, read:org")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts, err := validateTokenScopesWithURL(context.Background(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.HasProject {
		t.Error("HasProject = false, want true — read:org scope should imply project access")
	}
}

func TestValidateTokenScopes_AdminOrgImpliesProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-OAuth-Scopes", "repo, admin:org")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ts, err := validateTokenScopesWithURL(context.Background(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.HasProject {
		t.Error("HasProject = false, want true — admin:org scope should imply project access")
	}
}

func TestValidateTokenScopes_SendsBearerToken(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("X-OAuth-Scopes", "repo, project")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := validateTokenScopesWithURL(context.Background(), "my-secret-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedAuth != "Bearer my-secret-token" {
		t.Errorf("Authorization header = %q, want %q", receivedAuth, "Bearer my-secret-token")
	}
}

// validateTokenScopesWithURL is a test helper that calls the validation logic
// against a custom URL (e.g., httptest server) instead of api.github.com.
func validateTokenScopesWithURL(ctx context.Context, token, baseURL string) (*TokenScopes, error) {
	return validateTokenScopesInternal(ctx, token, baseURL+"/user")
}
