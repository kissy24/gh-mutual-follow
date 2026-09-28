// Package github accesses only GitHub.com using GitHub CLI credentials.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gh-mutual-follow/internal/domain"
)

const apiHost = "api.github.com"

// Client is used sequentially for one invocation, keeping the same credential throughout.
type Client struct {
	http        *http.Client
	baseURL     string
	tokenSource func(context.Context) (string, error)
	token       string
	timeout     time.Duration
}

func NewClient() *Client {
	return &Client{
		http:    &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		baseURL: "https://" + apiHost, tokenSource: resolveToken, timeout: 30 * time.Second,
	}
}

func resolveToken(ctx context.Context) (string, error) {
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := os.Getenv(key); token != "" {
			return token, nil
		}
	}
	path, err := exec.LookPath("gh")
	if err != nil {
		return "", &domain.Error{Code: "authentication_failed", Message: "Install GitHub CLI, then run gh auth login --hostname github.com."}
	}
	// No shell, no token in command arguments, no stderr or raw process error exposed.
	cmd := exec.CommandContext(ctx, path, "auth", "token", "--hostname", "github.com")
	output, err := cmd.Output()
	if err != nil {
		return "", &domain.Error{Code: "authentication_failed", Message: "Run gh auth login --hostname github.com, or supply GH_TOKEN."}
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *Client) request(ctx context.Context, method, path string) ([]byte, http.Header, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if c.token == "" {
		token, err := c.tokenSource(ctx)
		if err != nil {
			return nil, nil, err
		}
		if token == "" || strings.ContainsAny(token, "\r\n") {
			return nil, nil, &domain.Error{Code: "authentication_failed", Message: "Missing or invalid GitHub credential."}
		}
		c.token = token
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return nil, nil, &domain.Error{Code: "api_error", Message: "Could not construct GitHub request."}
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "gh-mutual-follow")
	resp, err := c.http.Do(req)
	if err != nil {
		code := "api_error"
		message := "Could not read GitHub data. Check your connection and retry."
		if method != "GET" {
			code = "outcome_unknown"
			message = "The change could not be confirmed. Check list before retrying."
		}
		return nil, nil, &domain.Error{Code: code, Message: message}
	}
	defer resp.Body.Close()
	expected := http.StatusOK
	if method != "GET" {
		expected = http.StatusNoContent
	}
	if resp.StatusCode != expected {
		return nil, nil, responseError(resp, method != "GET")
	}
	if method != "GET" {
		return nil, resp.Header, nil
	}
	const limit = 8 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(body) > limit {
		return nil, nil, &domain.Error{Code: "api_error", Message: "Incomplete or oversized GitHub response; no changes made."}
	}
	return body, resp.Header, nil
}

func responseError(resp *http.Response, mutation bool) *domain.Error {
	e := &domain.Error{Code: "api_error", Message: fmt.Sprintf("GitHub returned HTTP %d; no further changes attempted.", resp.StatusCode)}
	switch {
	case resp.StatusCode == 401:
		e.Code = "authentication_failed"
		e.Message = "GitHub authentication failed. Run gh auth login --hostname github.com or replace GH_TOKEN."
	case resp.StatusCode == 429 || (resp.StatusCode == 403 && (resp.Header.Get("X-RateLimit-Remaining") == "0" || resp.Header.Get("Retry-After") != "")):
		e.Code = "rate_limited"
		e.Message = "GitHub rate limit reached; processing stopped."
		if n, err := strconv.ParseUint(resp.Header.Get("Retry-After"), 10, 32); err == nil {
			e.Message += fmt.Sprintf(" Retry after at least %d seconds.", n)
		} else if n, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil && n > 0 {
			e.Message += " Rate limit resets at " + time.Unix(n, 0).UTC().Format(time.RFC3339) + "."
		}
	case resp.StatusCode == 403:
		e.Code = "permission_denied"
		e.Message = "Permission denied. For OAuth/classic tokens grant user:follow (gh auth refresh --hostname github.com --scopes user:follow); for fine-grained tokens grant Followers read/write permissions."
	case mutation && (resp.StatusCode >= 500 || resp.StatusCode == 408 || (resp.StatusCode >= 200 && resp.StatusCode < 300)):
		e.Code = "outcome_unknown"
		e.Message = "GitHub did not confirm the change. Check list before retrying."
	}
	return e
}

type apiUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func (u apiUser) user() (domain.User, error) {
	if u.ID <= 0 || !domain.ValidLogin(u.Login) {
		return domain.User{}, &domain.Error{Code: "api_error", Message: "GitHub returned an invalid user record; no changes made."}
	}
	return domain.User{ID: u.ID, Login: u.Login, URL: "https://github.com/" + u.Login}, nil
}
func invalidJSON() error {
	return &domain.Error{Code: "api_error", Message: "GitHub returned invalid JSON; no changes made."}
}

func (c *Client) User(ctx context.Context) (domain.User, error) {
	body, _, err := c.request(ctx, "GET", "/user")
	if err != nil {
		return domain.User{}, err
	}
	var u apiUser
	if json.Unmarshal(body, &u) != nil {
		return domain.User{}, invalidJSON()
	}
	return u.user()
}
func (c *Client) Following(ctx context.Context) ([]domain.User, error) {
	return c.list(ctx, "/user/following")
}
func (c *Client) Followers(ctx context.Context) ([]domain.User, error) {
	return c.list(ctx, "/user/followers")
}
func (c *Client) list(ctx context.Context, path string) ([]domain.User, error) {
	users := []domain.User{}
	for page := 1; ; {
		body, headers, err := c.request(ctx, "GET", fmt.Sprintf("%s?per_page=100&page=%d", path, page))
		if err != nil {
			return nil, err
		}
		var batch []apiUser
		if json.Unmarshal(body, &batch) != nil || batch == nil {
			return nil, invalidJSON()
		}
		for _, raw := range batch {
			u, err := raw.user()
			if err != nil {
				return nil, err
			}
			users = append(users, u)
		}
		next, err := nextPage(headers.Get("Link"), path, page)
		if err != nil {
			return nil, err
		}
		if next == 0 {
			return users, nil
		}
		page = next
	}
}

// Validate pagination destinations, but construct requests locally so credentials
// never follow URLs supplied by the API to another host or endpoint.
func nextPage(link, path string, current int) (int, error) {
	for _, part := range strings.Split(link, ",") {
		target, params, ok := strings.Cut(strings.TrimSpace(part), ";")
		if !ok || !strings.Contains(params, `rel="next"`) {
			continue
		}
		u, err := url.Parse(strings.Trim(strings.TrimSpace(target), "<>"))
		if err != nil {
			return 0, invalidJSON()
		}
		n, err := strconv.Atoi(u.Query().Get("page"))
		if err != nil || n <= current || u.Scheme != "https" || u.Host != apiHost || u.Path != path || u.User != nil {
			return 0, &domain.Error{Code: "api_error", Message: "GitHub returned unsafe pagination; no changes made."}
		}
		return n, nil
	}
	return 0, nil
}

func (c *Client) Change(ctx context.Context, login string, follow bool) error {
	if !domain.ValidLogin(login) {
		return &domain.Error{Code: "invalid_target", Message: "Invalid GitHub login; no changes made."}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	method := "DELETE"
	if follow {
		method = "PUT"
	}
	_, _, err := c.request(ctx, method, "/user/following/"+url.PathEscape(login))
	// A request cancelled after transmission may already have changed server state.
	if err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		return &domain.Error{Code: "outcome_unknown", Message: "The change could not be confirmed. Check list before retrying."}
	}
	return err
}
