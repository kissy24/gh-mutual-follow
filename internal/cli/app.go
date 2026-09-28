// Package cli implements the command contract independently of terminal and API adapters.
package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"gh-mutual-follow/internal/domain"
)

type Client interface {
	User(context.Context) (domain.User, error)
	Following(context.Context) ([]domain.User, error)
	Followers(context.Context) ([]domain.User, error)
	Change(context.Context, string, bool) error
}
type IO struct {
	In                 io.Reader
	Out, Err           io.Writer
	InputTTY, ErrorTTY bool
}

func safeError(err error) *domain.Error {
	var e *domain.Error
	if errors.As(err, &e) {
		return e
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return &domain.Error{Code: "outcome_unknown", Message: "Request interrupted; check list before retrying."}
	}
	return &domain.Error{Code: "api_error", Message: "GitHub operation failed; no further changes were attempted."}
}

func Run(ctx context.Context, args []string, client Client, streams IO, version string) int {
	if streams.In == nil {
		streams.In = strings.NewReader("")
	}
	if streams.Out == nil {
		streams.Out = io.Discard
	}
	if streams.Err == nil {
		streams.Err = io.Discard
	}
	o, err := parse(args)
	r := Report{SchemaVersion: 1, Command: o.command, Host: "github.com", DryRun: o.dryRun, Status: "ok", Users: []domain.User{}, Results: []Result{}}
	r.typeFilter = o.kind
	finish := func(code int, e *domain.Error) int {
		r.Error = e
		if code != 0 {
			r.Status = "error"
		}
		if code == 3 || code == 130 {
			r.Status = "cancelled"
		}
		return emit(r, o.json, streams, code)
	}
	if err != nil {
		return finish(2, &domain.Error{Code: "invalid_arguments", Message: err.Error()})
	}
	if o.help || o.version {
		var e error
		if o.version {
			_, e = fmt.Fprintln(streams.Out, "gh-mutual-follow "+version)
		} else {
			_, e = fmt.Fprint(streams.Out, help)
		}
		if e != nil {
			return 1
		}
		return 0
	}
	fail := func(err error) int {
		if ctx.Err() != nil {
			return finish(130, &domain.Error{Code: "cancelled", Message: "Interrupted; completed changes were not rolled back."})
		}
		return finish(1, safeError(err))
	}
	if ctx.Err() != nil {
		return fail(ctx.Err())
	}
	user, err := client.User(ctx)
	if err != nil {
		return fail(err)
	}
	r.Account = &user.Login
	following, err := client.Following(ctx)
	if err != nil {
		return fail(err)
	}
	followers, err := client.Followers(ctx)
	if err != nil {
		return fail(err)
	}
	if ctx.Err() != nil {
		return fail(ctx.Err())
	}
	users := domain.Difference(user.ID, following, followers)
	if o.command == "list" {
		for _, u := range users {
			if o.kind == "all" || u.Relation == o.kind {
				r.Users = append(r.Users, u)
			}
		}
		return finish(0, nil)
	}
	relation := "following-only"
	if o.command == "follow-back" {
		relation = "followers-only"
	}
	selected, excluded, err := domain.Select(users, relation, o.names, o.all, o.excludes)
	if err != nil {
		return finish(2, &domain.Error{Code: "invalid_target", Message: err.Error()})
	}
	for _, u := range selected {
		r.Results = append(r.Results, Result{Login: u.Login, Status: "not_run"})
	}
	for _, u := range excluded {
		r.Results = append(r.Results, Result{Login: u.Login, Status: "excluded"})
	}
	if o.dryRun {
		for i := range selected {
			r.Results[i].Status = "planned"
		}
		return finish(0, nil)
	}
	if len(selected) == 0 {
		return finish(0, nil)
	}
	if !o.yes {
		if !streams.InputTTY {
			return finish(2, &domain.Error{Code: "confirmation_required", Message: "Non-interactive changes require --yes; use --dry-run to preview."})
		}
		prompt := &trackedWriter{Writer: streams.Err}
		fmt.Fprintf(prompt, "Account: %s (github.com)\nAction: %s\nTargets: %d / Excluded: %d\n", user.Login, o.command, len(selected), len(excluded))
		for _, u := range selected {
			fmt.Fprintf(prompt, "  %s\n", u.Login)
		}
		fmt.Fprintf(prompt, "Apply %s to these %d users? [y/N]: ", o.command, len(selected))
		if prompt.err != nil {
			return finish(1, &domain.Error{Code: "output_error", Message: "Could not display confirmation; no changes made."})
		}
		accepted, err := confirm(ctx, streams.In)
		if err != nil {
			return fail(err)
		}
		if !accepted {
			return finish(3, &domain.Error{Code: "cancelled", Message: "Cancelled; no changes made."})
		}
	}
	for i, u := range selected {
		if ctx.Err() != nil {
			return fail(ctx.Err())
		}
		if streams.ErrorTTY && !o.json {
			fmt.Fprintf(streams.Err, "[%d/%d] %s %s\n", i+1, len(selected), o.command, u.Login)
		}
		err := client.Change(ctx, u.Login, o.command == "follow-back")
		if err != nil {
			e := safeError(err)
			r.Results[i].Error = e
			r.Results[i].Status = "failed"
			if e.Code == "outcome_unknown" {
				r.Results[i].Status = "unknown"
			}
			return fail(err)
		}
		r.Results[i].Status = "succeeded"
	}
	if ctx.Err() != nil {
		return fail(ctx.Err())
	}
	return finish(0, nil)
}

func confirm(ctx context.Context, in io.Reader) (bool, error) {
	type answer struct {
		line string
		err  error
	}
	ch := make(chan answer, 1)
	go func() {
		line, err := bufio.NewReader(io.LimitReader(in, 4096)).ReadString('\n')
		ch <- answer{line, err}
	}()
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case a := <-ch:
		if a.err != nil {
			return false, nil
		}
		s := strings.TrimSpace(a.line)
		return strings.EqualFold(s, "y") || strings.EqualFold(s, "yes"), nil
	}
}
