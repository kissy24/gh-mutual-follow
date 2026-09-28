// Package domain calculates one-way follow relationships without side effects.
package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type User struct {
	ID       int64  `json:"-"`
	Login    string `json:"login"`
	URL      string `json:"url"`
	Relation string `json:"relation"`
}

var loginPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,38}(\[bot\])?$`)

// ValidLogin also accepts GitHub's bot login suffix, without allowing path or terminal injection.
func ValidLogin(login string) bool { return loginPattern.MatchString(login) }

func Sort(users []User) {
	sort.Slice(users, func(i, j int) bool { return strings.ToLower(users[i].Login) < strings.ToLower(users[j].Login) })
}

func Difference(self int64, following, followers []User) []User {
	f, r := make(map[int64]User), make(map[int64]User)
	for _, u := range following {
		if u.ID != self {
			f[u.ID] = u
		}
	}
	for _, u := range followers {
		if u.ID != self {
			r[u.ID] = u
		}
	}
	result := []User{}
	for id, u := range f {
		if _, ok := r[id]; !ok {
			u.Relation = "following-only"
			result = append(result, u)
		}
	}
	for id, u := range r {
		if _, ok := f[id]; !ok {
			u.Relation = "followers-only"
			result = append(result, u)
		}
	}
	Sort(result)
	return result
}

func Select(users []User, relation string, names []string, all bool, excludes []string) ([]User, []User, error) {
	candidates := make(map[string]User)
	for _, u := range users {
		if u.Relation == relation {
			candidates[strings.ToLower(u.Login)] = u
		}
	}
	chosen := make(map[string]User)
	if all {
		chosen = candidates
	} else {
		for _, name := range names {
			key := strings.ToLower(name)
			u, ok := candidates[key]
			if !ok {
				return nil, nil, fmt.Errorf("%s is not a %s target; no changes made", name, relation)
			}
			chosen[key] = u
		}
	}
	omit := make(map[string]bool)
	for _, name := range excludes {
		omit[strings.ToLower(name)] = true
	}
	selected, excluded := []User{}, []User{}
	for key, u := range chosen {
		if omit[key] {
			excluded = append(excluded, u)
		} else {
			selected = append(selected, u)
		}
	}
	Sort(selected)
	Sort(excluded)
	return selected, excluded, nil
}
