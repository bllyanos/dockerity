package domain

import "testing"

func TestDeploymentTransitions(t *testing.T) {
	ok := [][2]DeploymentStatus{
		{DeploymentPending, DeploymentRunning}, {DeploymentRunning, DeploymentSucceeded},
		{DeploymentRunning, DeploymentFailed}, {DeploymentPending, DeploymentFailed}, {DeploymentPending, DeploymentCanceled},
	}
	bad := [][2]DeploymentStatus{
		{DeploymentPending, DeploymentSucceeded}, {DeploymentRunning, DeploymentCanceled},
		{DeploymentSucceeded, DeploymentFailed}, {DeploymentFailed, DeploymentRunning},
	}
	for _, c := range ok {
		if !c[0].CanTransition(c[1]) {
			t.Errorf("%s -> %s rejected", c[0], c[1])
		}
	}
	for _, c := range bad {
		if c[0].CanTransition(c[1]) {
			t.Errorf("%s -> %s allowed", c[0], c[1])
		}
	}
	if DeploymentRunning.Terminal() || !DeploymentCanceled.Terminal() {
		t.Fatal("terminal")
	}
}
