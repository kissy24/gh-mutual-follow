package domain

import (
	"reflect"
	"testing"
)

func TestDifferenceUsesIDsAndExcludesSelf(t *testing.T) {
	following := []User{{ID: 1, Login: "owner"}, {ID: 2, Login: "Alice"}, {ID: 3, Login: "old-name"}, {ID: 2, Login: "Alice"}}
	followers := []User{{ID: 3, Login: "new-name"}, {ID: 4, Login: "bob"}}
	got := Difference(1, following, followers)
	want := []User{{ID: 2, Login: "Alice", Relation: "following-only"}, {ID: 4, Login: "bob", Relation: "followers-only"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSelectValidatesAllBeforeExcluding(t *testing.T) {
	users := []User{{ID: 2, Login: "Alice", Relation: "following-only"}, {ID: 3, Login: "bob", Relation: "followers-only"}}
	selected, excluded, err := Select(users, "following-only", []string{"ALICE", "alice"}, false, nil)
	if err != nil || len(selected) != 1 || len(excluded) != 0 {
		t.Fatalf("selection: %v %v %v", selected, excluded, err)
	}
	selected, excluded, err = Select(users, "following-only", nil, true, []string{"ALICE", "absent"})
	if err != nil || len(selected) != 0 || len(excluded) != 1 {
		t.Fatalf("exclusion: %v %v %v", selected, excluded, err)
	}
	for _, name := range []string{"bob", "absent"} {
		if _, _, err := Select(users, "following-only", []string{"alice", name}, false, []string{name}); err == nil {
			t.Fatalf("accepted invalid target %q", name)
		}
	}
}

func TestValidLoginRejectsControlCharactersAndPaths(t *testing.T) {
	for _, name := range []string{"", "../user", "a/b", "a\n", "a\x1b", "--all", "@bob", "a,b"} {
		if ValidLogin(name) {
			t.Errorf("accepted %q", name)
		}
	}
	for _, name := range []string{"alice", "ALICE", "a-b", "dependabot[bot]"} {
		if !ValidLogin(name) {
			t.Errorf("rejected %q", name)
		}
	}
}
