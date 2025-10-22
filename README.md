# GitHub Actions Kubernetes Container Hook

This is a reimplementation of GitHub's [Actions Kubernetes Container
Hook](https://github.com/actions/runner-container-hooks/tree/main/packages/k8s)
in Go. Main focus is on performance and better error reporting towards
workflow developers.

## USAGE

To use this hook, you'll need to make a release available in your runner image
somewhere and set the ENV variable `ACTIONS_RUNNER_CONTAINER_HOOKS_PATH` to
point to hook.sh - It's meant to be a drop-in replacement for the original
node implementation, with one exception: We use the `watch` API of the pods to
get real-time updates on the pod status instead of polling, so the runner
service account needs this permission in addition to the original ones.

## Limitations

So far this hook does not support:

- Services
- Container Actions

This will likely be added in future releases.

## Non-Goals

This hook will not support running containers on other nodes than the
runner pod.
