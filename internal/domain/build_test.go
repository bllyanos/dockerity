package domain

import "testing"

func TestBuildTransitions(t *testing.T) {
	allowed := map[[2]BuildStatus]bool{
		{BuildQueued, BuildRunning}:    true,
		{BuildQueued, BuildCanceled}:   true,
		{BuildQueued, BuildFailed}:     true,
		{BuildRunning, BuildSucceeded}: true,
		{BuildRunning, BuildFailed}:    true,
		{BuildRunning, BuildCanceled}:  true,
	}
	all := []BuildStatus{BuildQueued, BuildRunning, BuildSucceeded, BuildFailed, BuildCanceled}
	for _, from := range all {
		for _, to := range all {
			if got := from.CanTransition(to); got != allowed[[2]BuildStatus{from, to}] {
				t.Errorf("%s -> %s: got %v", from, to, got)
			}
		}
	}
	for _, s := range all {
		if s.Terminal() != (s == BuildSucceeded || s == BuildFailed || s == BuildCanceled) {
			t.Errorf("%s terminal", s)
		}
	}
	src := BuildSourceStates(BuildCanceled)
	src[0] = "mutated"
	if BuildSourceStates(BuildCanceled)[0] != BuildQueued {
		t.Fatal("BuildSourceStates must return a copy")
	}
}
