package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gh-mutual-follow/internal/cli"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gh-mutual-follow/internal/domain"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := NewClient()
	c.baseURL = server.URL
	c.tokenSource = func(context.Context) (string, error) { return "test-credential", nil }
	return c
}

func TestIdentityAndAllPages(t *testing.T) {
	calls := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer test-credential" {
			t.Error("missing authentication")
		}
		if r.URL.Path == "/user" {
			fmt.Fprint(w, `{"id":1,"login":"owner"}`)
			return
		}
		if r.URL.Query().Get("per_page") != "100" {
			t.Error("wrong page size")
		}
		if r.URL.Query().Get("page") == "1" {
			w.Header().Set("Link", `<https://api.github.com/user/following?page=2>; rel="next"`)
			fmt.Fprint(w, "[")
			for i := 2; i <= 101; i++ {
				if i > 2 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, `{"id":%d,"login":"user%d"}`, i, i)
			}
			fmt.Fprint(w, "]")
		} else {
			fmt.Fprint(w, `[{"id":102,"login":"last-user"}]`)
		}
	})
	u, err := c.User(context.Background())
	if err != nil || u.Login != "owner" {
		t.Fatalf("identity: %+v %v", u, err)
	}
	users, err := c.Following(context.Background())
	if err != nil || len(users) != 101 || calls != 3 {
		t.Fatalf("pages: %d calls %d error %v", len(users), calls, err)
	}
	if users[100].URL != "https://github.com/last-user" {
		t.Fatal("wrong profile URL")
	}
}

func TestLaterPageFailureReturnsNoPartialList(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			w.Header().Set("Link", `<https://api.github.com/user/followers?page=2>; rel="next"`)
			fmt.Fprint(w, `[{"id":2,"login":"alice"}]`)
		} else {
			w.WriteHeader(500)
		}
	})
	users, err := c.Followers(context.Background())
	if err == nil || len(users) != 0 {
		t.Fatalf("partial data leaked: %+v %v", users, err)
	}
}

func TestChangeHTTPContract(t *testing.T) {
	for _, follow := range []bool{true, false} {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			want := "DELETE"
			if follow {
				want = "PUT"
			}
			if r.Method != want || r.URL.Path != "/user/following/alice" {
				t.Errorf("request %s %s", r.Method, r.URL)
			}
			if r.ContentLength != 0 {
				t.Error("unexpected body")
			}
			w.WriteHeader(204)
		})
		if err := c.Change(context.Background(), "alice", follow); err != nil {
			t.Fatal(err)
		}
	}
}

func TestErrorClassificationAndRedaction(t *testing.T) {
	for _, tt := range []struct {
		status                 int
		remaining, retry, code string
	}{
		{401, "", "", "authentication_failed"}, {403, "", "", "permission_denied"}, {403, "0", "", "rate_limited"},
		{429, "", "60", "rate_limited"}, {422, "", "", "api_error"}, {500, "", "", "outcome_unknown"},
	} {
		t.Run(fmt.Sprint(tt.status, tt.remaining), func(t *testing.T) {
			c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-RateLimit-Remaining", tt.remaining)
				w.Header().Set("Retry-After", tt.retry)
				w.WriteHeader(tt.status)
				fmt.Fprint(w, `{"message":"test-credential secret"}`)
			})
			err := c.Change(context.Background(), "alice", false)
			var e *domain.Error
			if !errors.As(err, &e) || e.Code != tt.code || strings.Contains(e.Message, "test-credential") {
				t.Fatalf("error: %v", err)
			}
		})
	}
}

func TestRejectUntrustedAPIData(t *testing.T) {
	for _, body := range []string{`{"id":0,"login":"alice"}`, `{"id":2,"login":"../alice"}`, `{"id":2,"login":"alice\u001b"}`, `null`, `{"id":2,"login":"alice"} {}`} {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
		if _, err := c.User(context.Background()); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRejectRedirectAndHostilePagination(t *testing.T) {
	for _, redirect := range []bool{false, true} {
		calls := 0
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if redirect {
				w.Header().Set("Location", "https://example.invalid/steal")
				w.WriteHeader(302)
			} else {
				w.Header().Set("Link", `<https://example.invalid/steal?page=2>; rel="next"`)
				fmt.Fprint(w, `[]`)
			}
		})
		if _, err := c.Following(context.Background()); err == nil || calls != 1 {
			t.Fatalf("unsafe follow-up: %v calls %d", err, calls)
		}
	}
}

func TestTimeoutAndInvalidTargets(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); <-r.Context().Done() })
	c.timeout = 10 * time.Millisecond
	err := c.Change(context.Background(), "alice", false)
	var e *domain.Error
	if !errors.As(err, &e) || e.Code != "outcome_unknown" {
		t.Fatalf("timeout: %v", err)
	}
	if err := c.Change(context.Background(), "../alice", false); err == nil || calls.Load() != 1 {
		t.Fatal("invalid target reached server")
	}
}

func TestEnvironmentTokenPrecedence(t *testing.T) {
	t.Setenv("GH_TOKEN", "first")
	t.Setenv("GITHUB_TOKEN", "second")
	got, err := resolveToken(context.Background())
	if err != nil || got != "first" {
		t.Fatalf("%q %v", got, err)
	}
	t.Setenv("GH_TOKEN", "")
	got, err = resolveToken(context.Background())
	if err != nil || got != "second" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestCLIStopsBeforeChangesWhenSnapshotFails(t *testing.T) {
	var changes atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			changes.Add(1)
			w.WriteHeader(204)
			return
		}
		switch r.URL.Path {
		case "/user":
			fmt.Fprint(w, `{"id":1,"login":"owner"}`)
		case "/user/following":
			fmt.Fprint(w, `[{"id":2,"login":"alice"}]`)
		case "/user/followers":
			if r.URL.Query().Get("page") == "1" {
				w.Header().Set("Link", `<https://api.github.com/user/followers?page=2>; rel="next"`)
				fmt.Fprint(w, `[]`)
			} else {
				w.WriteHeader(500)
			}
		default:
			t.Errorf("unexpected request %s", r.URL)
		}
	})
	var out bytes.Buffer
	code := cli.Run(context.Background(), []string{"unfollow", "--all", "--yes", "--json"}, c, cli.IO{Out: &out}, "test")
	var report cli.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if code != 1 || changes.Load() != 0 || report.Summary.Targets != 0 || report.Status != "error" {
		t.Fatalf("code %d writes %d report %+v", code, changes.Load(), report)
	}
}

func TestCLIUsesAuthenticatedSnapshotForChanges(t *testing.T) {
	var changes atomic.Int32
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/user/following/alice" {
			changes.Add(1)
			w.WriteHeader(204)
			return
		}
		switch r.URL.Path {
		case "/user":
			fmt.Fprint(w, `{"id":1,"login":"owner"}`)
		case "/user/following":
			fmt.Fprint(w, `[{"id":2,"login":"alice"},{"id":3,"login":"bob"}]`)
		case "/user/followers":
			fmt.Fprint(w, `[{"id":3,"login":"bob"}]`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
	})
	var out bytes.Buffer
	code := cli.Run(context.Background(), []string{"unfollow", "--all", "--yes", "--json"}, c, cli.IO{Out: &out}, "test")
	var report cli.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if code != 0 || changes.Load() != 1 || report.Summary.Succeeded != 1 || report.Results[0].Login != "alice" {
		t.Fatalf("code %d report %+v", code, report)
	}
}
