package command

import (
	"log/slog"
	"os"

	"github.com/reMarkable/k8s-hook/pkg/k8s"
	"github.com/reMarkable/k8s-hook/pkg/types"
)

func CleanupJob(input types.ContainerHookInput) {
	k8s, err := k8s.NewK8sClient()
	if err != nil {
		slog.Error("Failed to talk to kubernetes", "err", err)
	}
	deleteErr := k8s.DeletePod(input.State["JobPod"])
	if err != nil {
		slog.Error("Failed to clean up pod", "err", deleteErr)
		os.Exit(1)
	}
}
