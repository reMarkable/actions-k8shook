package k8s

import (
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestK8sClient_waitForPodReady(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		want    v1.PodPhase
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := NewK8sClient()
			if err != nil {
				t.Fatalf("could not construct receiver type: %v", err)
			}
			gotErr := c.waitForPodReady(tt.name)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("waitForPodReady() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("waitForPodReady() succeeded unexpectedly")
			}
		})
	}
}
