# GitHub Actions Kubernetes Container Hook

This is a reimplementation of GitHub's [Actions Kubernetes Container
Hook](https://github.com/actions/runner-container-hooks/tree/main/packages/k8s)
in Go. Main focus is on performance and better error reporting towards
workflow developers.

## USAGE

To use this hook, you'll need to make a release available in your runner image
somewhere and set the ENV variable `ACTIONS_RUNNER_CONTAINER_HOOKS_PATH` to
point to `hook.sh` - It's meant to be a drop-in replacement for the original
node implementation, with one exception: We use the `watch` API of the pods to
get real-time updates on the pod status instead of polling, so the runner
service account needs this permission in addition to the original ones.

## Supported ENV variables

- `DEBUG_HOOK` - Output additional debug information to the logs.
- `ENV_USE_KUBE_SCHEDULER` - Rely on affinity to tie the worker pod to the same
  node as the runner pod. By default, the hook set the nodeName field of the
  pod based on the runner pod's node.
- `ACTIONS_RUNNER_CLAIM_NAME` - override the default claim name used to find
  the runner pod. By default it will be `[runner-pod]-work` which works out of
  the box for ARC.
- `ENV_`
- `ENV_HOOK_DEFAULT_ENTRYPOINT` - Override the default entrypoint used when
  none is specified in the workflow file. By default, this is required for
  container actions.
- `ENV_HOOK_INSPECT_IMAGE` - **(Experimental)** When set to `1`, the hook will
  automatically inspect container images to extract the entrypoint from the
  image configuration. This eliminates the need to manually specify
  `ENV_HOOK_CONTAINER_STEP_ENTRYPOINT` for most container actions. The hook
  will fall back to `ENV_HOOK_CONTAINER_STEP_ENTRYPOINT` if image inspection
  fails or if the image has no entrypoint defined. This feature requires
  network access to the container registry.

## Limitations

So far this hook does not support:

- Services
- Hook-Extension side cars

This will likely be added in future releases.

## Non-Goals

This hook will not support running containers on other nodes than the
runner pod.
