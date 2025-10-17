package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/reMarkable/k8s-hook/pkg/command"
	"github.com/reMarkable/k8s-hook/pkg/types"
)

func main() {
	if os.Getenv("DEBUG_HOOK") == "1" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}
	if checkPipedInput() {
		hookInput := getInput()
		switch hookInput.Command {
		case "prepare_job":
			command.PrepareJob(hookInput)
		case "cleanup_job":
			command.CleanupJob(hookInput)
		case "run_container_step":
			command.RunContainerStep(hookInput)
		case "run_script_step":
			command.RunScriptStep(hookInput)
		default:
			slog.Error("Unknown command", "command", hookInput.Command)
			os.Exit(1)
		}
	} else {
		fmt.Println("No piped input detected. This hook is intended to be run by github actions runner.")
	}
}

func getInput() types.ContainerHookInput {
	hookInput := types.ContainerHookInput{}
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&hookInput); err != nil {
		fmt.Fprintf(os.Stderr, "Unexpected JSON structure: %v\n", err)
		os.Exit(1)
	}
	res, _ := json.Marshal(hookInput)
	fmt.Printf("%s\n", res)
	return hookInput
}

func checkPipedInput() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	if fi.Mode()&os.ModeCharDevice == 0 {
		return true
	}
	return false
}
