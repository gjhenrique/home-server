package slopper

import "testing"

func TestNames(t *testing.T) {
	for _, x := range []string{"my-app", "a", "a23456789012345678901234567890"} {
		if !ValidApp(x) {
			t.Errorf("expected valid app %q", x)
		}
	}
	for _, x := range []string{"API", "api", "admin", "has_underscore", "-bad", "bad-"} {
		if ValidApp(x) {
			t.Errorf("expected invalid app %q", x)
		}
	}
	for _, x := range []string{"items", "user_notes", "a12345678901234567890123"} {
		if !ValidResource(x) {
			t.Errorf("expected valid resource %q", x)
		}
	}
	for _, x := range []string{"A", "1item", "a123456789012345678901234", "has-dash"} {
		if ValidResource(x) {
			t.Errorf("expected invalid resource %q", x)
		}
	}
}
