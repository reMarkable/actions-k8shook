# GitHub Actions Kubernetes Container Hook

This is a reimplementation of GitHub's [Actions Kubernetes Container
Hook](https://github.com/actions/runner-container-hooks/tree/main/packages/k8s)
in Go. Main focus is on performance and better error reporting towards
workflow developers.

## Non-Goals

This hook will not support running containers on other nodes than the
runner pod.
