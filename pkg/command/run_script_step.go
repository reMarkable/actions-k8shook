package command

import (
	"log/slog"
	"os"

	"github.com/reMarkable/k8s-hook/pkg/k8s"
	"github.com/reMarkable/k8s-hook/pkg/types"
)

func RunScriptStep(input types.ContainerHookInput) {
	k8s, err := k8s.NewK8sClient()
	if err != nil {
		slog.Error("Failed to talk to kubernetes", "err", err)
		os.Exit(1)
	}
	k8s.ExecStepInPod(input.State["JobPod"], input.Args.Container.Entrypoint, input.Args.Container.EntrypointArgs)
}
