package command

import (
	"log/slog"
	"os"

	"github.com/reMarkable/k8s-hook/pkg/types"
)

func RunContainerStep(input types.ContainerHookInput) {
	slog.Error("RunContainerStep not implemented")
	os.Exit(1)
}
