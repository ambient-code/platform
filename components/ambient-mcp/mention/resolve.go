package mention

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var mentionPattern = regexp.MustCompile(`@([a-zA-Z0-9_-]+)`)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type TokenFunc func() string

type Resolver struct {
	baseURL string
	tokenFn TokenFunc
	http    *http.Client
}

func NewResolver(baseURL string, tokenFn TokenFunc) (*Resolver, error) {
	if tokenFn == nil {
		return nil, fmt.Errorf("tokenFn must not be nil")
	}
	return &Resolver{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		tokenFn: tokenFn,
		http:    &http.Client{},
	}, nil
}

type agentSearchResult struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
	Total int `json:"total"`
}

func (r *Resolver) Resolve(ctx context.Context, projectID, identifier string) (string, error) {
	if uuidPattern.MatchString(strings.ToLower(identifier)) {
		path := r.baseURL + "/api/ambient/v1/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(identifier)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+r.tokenFn())
		resp, err := r.http.Do(req)
		if err != nil {
			return "", fmt.Errorf("lookup agent by ID: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("AGENT_NOT_FOUND")
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("AGENT_NOT_FOUND: HTTP %d", resp.StatusCode)
		}
		var a struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
			return "", fmt.Errorf("decode agent: %w", err)
		}
		return a.ID, nil
	}

	path := r.baseURL + "/api/ambient/v1/projects/" + url.PathEscape(projectID) + "/agents?search=name='" + url.QueryEscape(identifier) + "'"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+r.tokenFn())
	resp, err := r.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("search agent by name: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("MENTION_NOT_RESOLVED: HTTP %d", resp.StatusCode)
	}
	var result agentSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode agent list: %w", err)
	}
	switch result.Total {
	case 0:
		return "", fmt.Errorf("MENTION_NOT_RESOLVED: no agent named %q", identifier)
	case 1:
		return result.Items[0].ID, nil
	default:
		return "", fmt.Errorf("AMBIGUOUS_AGENT_NAME: %d agents match %q", result.Total, identifier)
	}
}

type Match struct {
	Token      string
	Identifier string
	AgentID    string
}

func Extract(text string) []Match {
	found := mentionPattern.FindAllStringSubmatch(text, -1)
	seen := make(map[string]bool)
	var matches []Match
	for _, m := range found {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		matches = append(matches, Match{Token: m[0], Identifier: m[1]})
	}
	return matches
}

func StripToken(text, token string) string {
	return strings.TrimSpace(strings.ReplaceAll(text, token, ""))
}
