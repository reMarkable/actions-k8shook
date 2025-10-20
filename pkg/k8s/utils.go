package k8s

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/reMarkable/k8s-hook/pkg/types"
	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *K8sClient) GetNS() string {
	namespace := os.Getenv("ACTIONS_RUNNER_KUBERNETES_NAMESPACE")
	if namespace != "" {
		return namespace
	}
	return "default"
}

func (c *K8sClient) GetPodNodeName(mname string) (string, error) {
	pod, err := c.client.CoreV1().Pods(c.GetNS()).Get(c.ctx, mname, v1Meta.GetOptions{})
	if err != nil {
		return "", err
	}
	return pod.Spec.NodeName, nil
}

func (c *K8sClient) GetRunnerPodName() string {
	name := os.Getenv("ACTIONS_RUNNER_POD_NAME")
	if name == "" {
		name = "local-pod"
	}
	return name
}

func (c *K8sClient) GetVolumeClaimName() string {
	name := os.Getenv("ACTIONS_RUNNER_CLAIM_NAME")
	if name == "" {
		return c.GetRunnerPodName() + "-work"
	}
	return name
}

func (c *K8sClient) checkPermissions() {
	slog.Warn("TODO: Implement permission check for creating pods")
}

// copyExternals copies the externals directory into the workspace directory.
func copyExternals() {
	workspace := os.Getenv("RUNNER_WORKSPACE")
	if workspace != "" {
		slog.Info("Copying externals to workspace", "workspace", workspace)
		err := copyDir(filepath.Join(workspace, "../../externals"), filepath.Join(workspace, "../externals"))
		if err != nil {
			slog.Error("Failed to copy externals", "error", err)
		}
	}
}

// writeRunScript generates a shell script that sets up the environment and runs the specified entrypoint with its arguments.
// returns the path to the script in the container and the local path of the script file, or error if any.
func (c *K8sClient) writeRunScript(args types.InputArgs) (string, string, error) {
	prependPath := strings.Join(args.PrependPath, ":")
	cl := append([]string{args.Entrypoint}, args.EntrypointArgs...)
	scriptEnv, err := scriptEnvironment(args.EnvironmentVariables)
	if err != nil {
		return "", "", err
	}
	script := fmt.Sprintf(`
	#!/bin/sh -l
	set -e
	export PATH=%s:$PATH
	cd %s && exec %s %s
  `, prependPath, args.WorkingDirectory, scriptEnv, cl)
	f, err := os.CreateTemp(os.Getenv("RUNNER_TEMP"), "run-script-*.sh")
	if err != nil {
		return "", "", err
	}
	_, err = io.WriteString(f, script)
	if err != nil {
		return "", "", err
	}
	return "/__w/_temp/" + filepath.Base(f.Name()), f.Name(), nil
}

func scriptEnvironment(env map[string]string) (string, error) {
	var envstr strings.Builder
	envstr.WriteString("env")
	for k, v := range env {
		if strings.ContainsAny(k, `"'=$`) {
			return "", fmt.Errorf("invalid character [\"'=$] in environment variable key: %s", k)
		}
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `"`, `\"`)
		v = strings.ReplaceAll(v, `$`, `\$`)
		v = strings.ReplaceAll(v, "`", "\\`")
		envstr.WriteString(fmt.Sprintf(` "%s=%s"`, k, v))
	}
	return envstr.String(), nil
}

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}
		// Copy file
		return copyFile(path, targetPath, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := in.Close(); err != nil {
			slog.Warn("Failed to close source file", "error", err)
		}
	}()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); err != nil {
			slog.Warn("Failed to close destination file", "error", err)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}
