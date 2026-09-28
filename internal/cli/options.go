package cli

import (
	"fmt"
	"strings"

	"gh-mutual-follow/internal/domain"
)

type options struct {
	command, kind                         string
	names, excludes                       []string
	all, dryRun, yes, json, help, version bool
}

func parse(args []string) (options, error) {
	o := options{kind: "all"}
	for _, a := range args {
		if a == "--json" {
			o.json = true
		}
	}
	if len(args) == 0 {
		o.help = true
		return o, nil
	}
	if len(args) == 1 && args[0] == "--version" {
		o.version = true
		return o, nil
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		o.help = true
		return o, nil
	}
	o.command = args[0]
	if o.command != "list" && o.command != "follow-back" && o.command != "unfollow" {
		return o, fmt.Errorf("unknown command %q; use --help", o.command)
	}
	for i := 1; i < len(args); i++ {
		a := args[i]
		key, value, hasValue := strings.Cut(a, "=")
		switch key {
		case "--help", "-h":
			o.help = true
		case "--json":
			o.json = true
		case "--all":
			o.all = true
		case "--dry-run":
			o.dryRun = true
		case "--yes", "-y":
			o.yes = true
		case "--type", "--exclude":
			if !hasValue {
				i++
				if i == len(args) {
					return o, fmt.Errorf("%s requires a value", key)
				}
				value = args[i]
			}
			if key == "--type" {
				if o.command != "list" {
					return o, fmt.Errorf("--type is only supported by list")
				}
				o.kind = value
			} else {
				o.excludes = append(o.excludes, value)
			}
			continue
		default:
			if strings.HasPrefix(a, "-") {
				return o, fmt.Errorf("unknown flag %q", a)
			}
			o.names = append(o.names, a)
			continue
		}
		if hasValue {
			return o, fmt.Errorf("%s does not accept a value", key)
		}
	}
	if o.help {
		return o, nil
	}
	if o.command == "list" {
		if o.all || o.dryRun || o.yes || len(o.excludes) > 0 || len(o.names) > 0 {
			return o, fmt.Errorf("list accepts only --type and --json")
		}
		if o.kind != "all" && o.kind != "following-only" && o.kind != "followers-only" {
			return o, fmt.Errorf("--type must be all, following-only, or followers-only")
		}
	} else {
		if o.all == (len(o.names) > 0) {
			return o, fmt.Errorf("specify either USER... or --all")
		}
		if o.json && !o.dryRun && !o.yes {
			return o, fmt.Errorf("changes with --json require --yes (or --dry-run)")
		}
	}
	for _, name := range append(append([]string{}, o.names...), o.excludes...) {
		if !domain.ValidLogin(name) {
			return o, fmt.Errorf("invalid login %q", name)
		}
	}
	return o, nil
}

const help = `Usage: gh mutual-follow <command> [options]

Commands:
  list         List one-way follows (--type all|following-only|followers-only)
  follow-back  Follow users who follow you, but whom you do not follow
  unfollow     Unfollow users who do not follow you back

Changes require USER... or --all (mutually exclusive).
  --exclude USER  Exclude a user; repeat for multiple users
  --dry-run       Show the plan without changing anything
  --yes, -y       Skip the single confirmation
  --json         JSON output; changes require --yes or --dry-run
  --help, -h     Show help
  --version      Show version (root command only)

Examples:
  gh mutual-follow list
  gh mutual-follow follow-back --all
  gh mutual-follow unfollow --all --exclude alice --dry-run
  gh mutual-follow unfollow alice --yes --json

Exit codes: 0 success, 1 runtime failure, 2 invalid usage/target,
            3 confirmation declined, 130 interrupted.
`
