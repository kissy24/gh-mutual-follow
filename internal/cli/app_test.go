package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"gh-mutual-follow/internal/domain"
)

type fakeClient struct {
	reads                int
	writes               []string
	following, followers []domain.User
	readErr, writeErr    error
	failAt               int
	cancel               context.CancelFunc
}

func fixture() *fakeClient {
	return &fakeClient{
		following: []domain.User{{ID: 2, Login: "alice"}, {ID: 3, Login: "bob"}, {ID: 5, Login: "dave"}},
		followers: []domain.User{{ID: 3, Login: "bob"}, {ID: 4, Login: "charlie"}},
	}
}
func (f *fakeClient) User(context.Context) (domain.User, error) {
	f.reads++
	return domain.User{ID: 1, Login: "owner"}, f.readErr
}
func (f *fakeClient) Following(context.Context) ([]domain.User, error) {
	f.reads++
	return f.following, nil
}
func (f *fakeClient) Followers(context.Context) ([]domain.User, error) {
	f.reads++
	return f.followers, f.readErr
}
func (f *fakeClient) Change(ctx context.Context, login string, follow bool) error {
	f.writes = append(f.writes, login)
	if f.cancel != nil {
		f.cancel()
		return ctx.Err()
	}
	if len(f.writes) == f.failAt {
		return f.writeErr
	}
	return nil
}

func execute(t *testing.T, f *fakeClient, args ...string) (int, Report) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(context.Background(), args, f, IO{In: strings.NewReader(""), Out: &out, Err: &errOut}, "test")
	var result Report
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %s (%v), stderr %s", out.String(), err, errOut.String())
	}
	return code, result
}

func TestHelpDoesNotAccessGitHub(t *testing.T) {
	for _, args := range [][]string{nil, {"--help"}, {"--version"}, {"unfollow", "--help"}} {
		f := fixture()
		var out bytes.Buffer
		if code := Run(context.Background(), args, f, IO{Out: &out, Err: io.Discard}, "test"); code != 0 || f.reads != 0 || out.Len() == 0 {
			t.Fatalf("args %v: code %d, reads %d", args, code, f.reads)
		}
	}
}

func TestListJSON(t *testing.T) {
	f := fixture()
	code, r := execute(t, f, "list", "--json")
	if code != 0 || len(r.Users) != 3 || r.Users[0].Login != "alice" || r.Users[1].Login != "charlie" || len(f.writes) != 0 {
		t.Fatalf("code %d: %+v", code, r)
	}
	code, r = execute(t, fixture(), "list", "--type", "followers-only", "--json")
	if code != 0 || len(r.Users) != 1 || r.Users[0].Login != "charlie" {
		t.Fatalf("filtered: %+v", r)
	}
}

func TestRejectArgumentsBeforeNetwork(t *testing.T) {
	for _, args := range [][]string{
		{"unfollow"}, {"unfollow", "alice", "--all"}, {"unfollow", "--all", "--json"},
		{"list", "--all"}, {"list", "--type", "bad"}, {"unfollow", "--all", "--oops"},
		{"unfollow", "../alice", "--yes"}, {"follow-back", "--exclude", "alice"},
	} {
		args = append(args, "--json")
		f := fixture()
		code, r := execute(t, f, args...)
		if code != 2 || f.reads != 0 || r.Status != "error" {
			t.Fatalf("%v: code %d reads %d: %+v", args, code, f.reads, r)
		}
	}
}

func TestChangeContracts(t *testing.T) {
	tests := []struct {
		name                            string
		args                            []string
		code, writes, planned, excluded int
	}{
		{"dry run", []string{"unfollow", "--all", "--dry-run", "--yes", "--exclude", "ALICE"}, 0, 0, 1, 1},
		{"dedupe", []string{"unfollow", "ALICE", "alice", "--yes"}, 0, 1, 0, 0},
		{"follow back", []string{"follow-back", "charlie", "--yes"}, 0, 1, 0, 0},
		{"invalid target atomic", []string{"unfollow", "alice", "bob", "--yes"}, 2, 0, 0, 0},
		{"excluded invalid", []string{"unfollow", "bob", "--exclude", "bob", "--yes"}, 2, 0, 0, 0},
		{"empty", []string{"unfollow", "--all", "--exclude", "alice", "--exclude", "dave", "--yes"}, 0, 0, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fixture()
			code, r := execute(t, f, append(tt.args, "--json")...)
			if code != tt.code || len(f.writes) != tt.writes || r.Summary.Planned != tt.planned || r.Summary.Excluded != tt.excluded {
				t.Fatalf("code %d writes %v report %+v", code, f.writes, r)
			}
			s := r.Summary
			if s.Targets != s.Planned+s.Succeeded+s.Failed+s.Unknown+s.NotRun {
				t.Fatalf("unbalanced %+v", s)
			}
		})
	}
}

func TestStopsOnFirstFailure(t *testing.T) {
	f := fixture()
	f.following = append(f.following, domain.User{ID: 6, Login: "eve"})
	f.failAt = 2
	f.writeErr = errors.New("rejected")
	code, r := execute(t, f, "unfollow", "--all", "--yes", "--json")
	if code != 1 || len(f.writes) != 2 || r.Summary.Succeeded != 1 || r.Summary.Failed != 1 || r.Summary.NotRun != 1 {
		t.Fatalf("code %d: %+v", code, r)
	}
}

func TestConfirmation(t *testing.T) {
	for _, tt := range []struct {
		input        string
		terminal     bool
		code, writes int
	}{
		{"y\n", true, 0, 2}, {"YES\n", true, 0, 2}, {"\n", true, 3, 0}, {"no\n", true, 3, 0}, {"", true, 3, 0}, {"y", true, 3, 0}, {"y\n", false, 2, 0},
	} {
		f := fixture()
		var out, errOut bytes.Buffer
		code := Run(context.Background(), []string{"unfollow", "--all"}, f, IO{In: strings.NewReader(tt.input), Out: &out, Err: &errOut, InputTTY: tt.terminal}, "test")
		if code != tt.code || len(f.writes) != tt.writes {
			t.Fatalf("%+v: code %d writes %v", tt, code, f.writes)
		}
		if tt.terminal && !strings.Contains(errOut.String(), "alice") {
			t.Fatal("confirmation omitted targets")
		}
	}
}

func TestCancellationStopsWrites(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := fixture()
	f.cancel = cancel
	var out bytes.Buffer
	code := Run(ctx, []string{"unfollow", "--all", "--yes", "--json"}, f, IO{Out: &out, Err: io.Discard}, "test")
	var r Report
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if code != 130 || len(f.writes) != 1 || r.Summary.Unknown != 1 || r.Summary.NotRun != 1 {
		t.Fatalf("code %d %+v", code, r)
	}
}

func TestFilteredTextAndErrorsDoNotInventEmptyLists(t *testing.T) {
	var out bytes.Buffer
	code := Run(context.Background(), []string{"list", "--type", "followers-only"}, fixture(), IO{Out: &out}, "test")
	if code != 0 || strings.Contains(out.String(), "following-only:") || !strings.Contains(out.String(), "followers-only: 1") {
		t.Fatalf("filtered text: %s", out.String())
	}
	f := fixture()
	f.readErr = errors.New("offline")
	out.Reset()
	code = Run(context.Background(), []string{"list"}, f, IO{Out: &out}, "test")
	if code != 1 || strings.Contains(out.String(), "following-only: 0") {
		t.Fatalf("failure looked like empty success: %s", out.String())
	}
}

func TestErrorsUnknownOutcomesAndEmptyJSON(t *testing.T) {
	f := fixture()
	f.readErr = &domain.Error{Code: "authentication_failed", Message: "Sign in first."}
	code, r := execute(t, f, "list", "--json")
	if code != 1 || r.Account != nil || r.Users == nil || r.Error.Code != "authentication_failed" {
		t.Fatalf("auth failure: %+v", r)
	}
	f = fixture()
	f.failAt = 1
	f.writeErr = &domain.Error{Code: "outcome_unknown", Message: "Check list."}
	code, r = execute(t, f, "unfollow", "--all", "--yes", "--json")
	if code != 1 || r.Summary.Unknown != 1 || r.Summary.NotRun != 1 || len(f.writes) != 1 {
		t.Fatalf("unknown outcome: %+v", r)
	}
	f = fixture()
	f.following = nil
	f.followers = nil
	code, r = execute(t, f, "list", "--json")
	if code != 0 || r.Users == nil || len(r.Users) != 0 {
		t.Fatalf("empty users: %+v", r)
	}
	var out bytes.Buffer
	code = Run(context.Background(), []string{"unfollow", "--all"}, f, IO{Out: &out}, "test")
	if code != 0 || !strings.Contains(out.String(), "No changes needed") {
		t.Fatalf("empty non-interactive: %d %s", code, out.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestBrokenConfirmationDoesNotMutate(t *testing.T) {
	f := fixture()
	code := Run(context.Background(), []string{"unfollow", "--all"}, f, IO{In: strings.NewReader("yes\n"), Out: io.Discard, Err: brokenWriter{}, InputTTY: true}, "test")
	if code != 1 || len(f.writes) != 0 {
		t.Fatalf("code %d writes %v", code, f.writes)
	}
}

func TestCancellationWhileWaitingForConfirmation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	errOut := &cancelWriter{cancel: cancel}
	f := fixture()
	code := Run(ctx, []string{"unfollow", "--all"}, f, IO{In: reader, Err: errOut, InputTTY: true}, "test")
	if code != 130 || len(f.writes) != 0 {
		t.Fatalf("code %d writes %v", code, f.writes)
	}
}

type cancelWriter struct{ cancel context.CancelFunc }

func (w *cancelWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "[y/N]") {
		w.cancel()
	}
	return len(p), nil
}
