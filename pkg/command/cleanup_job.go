package command

import (
	"log/slog"

	"github.com/reMarkable/k8s-hook/pkg/k8s"
	"github.com/reMarkable/k8s-hook/pkg/types"
)

func CleanupJob(input types.ContainerHookInput) int {
	k8s, err := k8s.NewK8sClient()
	if err != nil {
		slog.Error("Failed to talk to kubernetes", "err", err)
	}
	err = k8s.PruneSecrets()
	if err != nil {
		slog.Error("Failed to prune secrets", "err", err)
		return 1
	}
	err = k8s.DeletePod(input.State["jobPod"])
	if err != nil {
		slog.Error("Failed to clean up pod", "err", err)
		return 1
	}
	return 0
}
