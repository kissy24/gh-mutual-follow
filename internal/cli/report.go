package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"gh-mutual-follow/internal/domain"
)

type Summary struct {
	Targets   int `json:"targets"`
	Planned   int `json:"planned"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Unknown   int `json:"unknown"`
	NotRun    int `json:"not_run"`
	Excluded  int `json:"excluded"`
}
type Result struct {
	Login  string        `json:"login"`
	Status string        `json:"status"`
	Error  *domain.Error `json:"error"`
}
type Report struct {
	typeFilter    string
	SchemaVersion int           `json:"schema_version"`
	Command       string        `json:"command"`
	Account       *string       `json:"account"`
	Host          string        `json:"host"`
	DryRun        bool          `json:"dry_run"`
	Status        string        `json:"status"`
	Error         *domain.Error `json:"error"`
	Users         []domain.User `json:"users"`
	Results       []Result      `json:"results"`
	Summary       Summary       `json:"summary"`
}

func (r *Report) summarize() {
	r.Summary = Summary{}
	for _, v := range r.Results {
		if v.Status != "excluded" {
			r.Summary.Targets++
		}
		switch v.Status {
		case "planned":
			r.Summary.Planned++
		case "succeeded":
			r.Summary.Succeeded++
		case "failed":
			r.Summary.Failed++
		case "unknown":
			r.Summary.Unknown++
		case "not_run":
			r.Summary.NotRun++
		case "excluded":
			r.Summary.Excluded++
		}
	}
	sort.Slice(r.Results, func(i, j int) bool { return strings.ToLower(r.Results[i].Login) < strings.ToLower(r.Results[j].Login) })
}

func (r Report) text(w io.Writer) {
	if r.Account == nil {
		return
	}
	if r.Account != nil {
		fmt.Fprintf(w, "Account: %s (%s)\n", *r.Account, r.Host)
	}
	if r.Command == "list" {
		if r.Error != nil {
			return
		}
		for _, relation := range []string{"following-only", "followers-only"} {
			if r.typeFilter != "all" && r.typeFilter != relation {
				continue
			}
			count := 0
			for _, u := range r.Users {
				if u.Relation == relation {
					count++
				}
			}
			fmt.Fprintf(w, "\n%s: %d\n", relation, count)
			for _, u := range r.Users {
				if u.Relation == relation {
					fmt.Fprintf(w, "%s\t%s\n", u.Login, u.URL)
				}
			}
		}
	} else {
		fmt.Fprintf(w, "Action: %s\n", r.Command)
		for _, v := range r.Results {
			label := strings.ToUpper(v.Status)
			if v.Status == "planned" {
				label = "Would unfollow"
				if r.Command == "follow-back" {
					label = "Would follow"
				}
			}
			fmt.Fprintf(w, "%s\t%s", label, v.Login)
			if v.Error != nil {
				fmt.Fprintf(w, "\t%s", v.Error.Message)
			}
			fmt.Fprintln(w)
		}
		s := r.Summary
		fmt.Fprintf(w, "Targets: %d / Planned: %d / Succeeded: %d / Failed: %d / Unknown: %d / Not run: %d / Excluded: %d\n", s.Targets, s.Planned, s.Succeeded, s.Failed, s.Unknown, s.NotRun, s.Excluded)
		if s.Targets == 0 && r.Error == nil {
			fmt.Fprintln(w, "No changes needed.")
		}
	}
}

// trackedWriter prevents continuing with mutations after a failed confirmation display.
type trackedWriter struct {
	io.Writer
	err error
}

func (w *trackedWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	w.err = err
	return n, err
}

func emit(r Report, jsonOutput bool, streams IO, code int) int {
	r.summarize()
	w := &trackedWriter{Writer: streams.Out}
	if jsonOutput {
		if err := json.NewEncoder(w).Encode(r); err != nil {
			return 1
		}
	} else {
		r.text(w)
		if r.Error != nil {
			fmt.Fprintln(streams.Err, r.Error.Message)
		}
	}
	if w.err != nil {
		return 1
	}
	return code
}
