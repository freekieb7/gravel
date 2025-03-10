package test

import "testing"

func AssertTrue(t *testing.T, a, b any) bool {
	t.Helper()

	if a != b {
		t.Errorf(""+
			"Not equal: \n"+
			"Expected: %s\n"+
			"Actual: %s", a, b)
		return true
	}

	return true
}
