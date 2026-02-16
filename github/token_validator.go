package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// TokenScopes holds the parsed scope information from a GitHub PAT token.
type TokenScopes struct {
	// Scopes contains the raw scope strings from the X-OAuth-Scopes header.
	Scopes []string

	// HasRepo indicates whether the "repo" scope is present (includes Actions access).
	HasRepo bool

	// HasProject indicates whether "project", "read:org", or "admin:org" scope is present.
	HasProject bool

	// IsClassicPAT is true when the X-OAuth-Scopes header was present in the response.
	IsClassicPAT bool

	// IsFineGrained is true when the X-OAuth-Scopes header was empty or missing,
	// indicating a fine-grained PAT that cannot be validated via headers.
	IsFineGrained bool
}

// MissingScopes returns a list of missing required scopes for display.
// For fine-grained PATs, it returns nil since scope validation is not possible
// via the X-OAuth-Scopes header.
func (ts *TokenScopes) MissingScopes() []string {
	if ts.IsFineGrained {
		return nil
	}

	var missing []string
	if !ts.HasRepo {
		missing = append(missing, "repo")
	}
	if !ts.HasProject {
		missing = append(missing, "project or read:org")
	}
	return missing
}

// Validate checks that all required scopes are present. For classic PATs with
// missing scopes, it returns an error listing exactly which scopes are needed.
// For fine-grained PATs, it logs a warning but returns nil (graceful degradation).
func (ts *TokenScopes) Validate() error {
	missing := ts.MissingScopes()
	if len(missing) > 0 {
		return fmt.Errorf("missing required token scopes: %s\n\nYour token needs: repo, project (or read:org)\nSee: https://github.com/settings/tokens", strings.Join(missing, ", "))
	}
	if ts.IsFineGrained {
		log.Println("Warning: fine-grained PAT detected — scope validation skipped. If you encounter permission errors, verify your token permissions.")
	}
	return nil
}

// ValidateTokenScopes makes a request to the GitHub API to determine the
// scopes associated with the given token. It reads the X-OAuth-Scopes header
// to detect classic PAT scopes and degrades gracefully for fine-grained PATs.
func ValidateTokenScopes(ctx context.Context, token string) (*TokenScopes, error) {
	return validateTokenScopesInternal(ctx, token, "https://api.github.com/user")
}

// validateTokenScopesInternal contains the core validation logic. It accepts a
// URL parameter so tests can point to an httptest server instead of the real
// GitHub API.
func validateTokenScopesInternal(ctx context.Context, token, url string) (*TokenScopes, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create token validation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token validation failed: HTTP %d — check that your token is valid", resp.StatusCode)
	}

	ts := &TokenScopes{}

	scopeHeader := resp.Header.Get("X-OAuth-Scopes")
	if scopeHeader != "" {
		ts.IsClassicPAT = true
		ts.Scopes = parseScopes(scopeHeader)
		ts.HasRepo = hasScope(ts.Scopes, "repo")
		ts.HasProject = hasScope(ts.Scopes, "project") ||
			hasScope(ts.Scopes, "read:org") ||
			hasScope(ts.Scopes, "admin:org")
	} else {
		ts.IsFineGrained = true
	}

	return ts, nil
}

// parseScopes splits a comma-separated scope header into trimmed scope strings.
func parseScopes(header string) []string {
	parts := strings.Split(header, ",")
	scopes := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			scopes = append(scopes, s)
		}
	}
	return scopes
}

// hasScope checks whether a specific scope exists in the scope list.
func hasScope(scopes []string, target string) bool {
	for _, s := range scopes {
		if s == target {
			return true
		}
	}
	return false
}
