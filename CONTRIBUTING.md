# Contributing

This guide explains how to set up a local development environment to test `actions-k8shook`.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [KIND (Kubernetes in Docker)](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Task](https://taskfile.dev/installation/) (optional, for using Taskfile commands)
- A GitHub Personal Access Token (PAT) with repo permissions

## Setup

### 1. Create a KIND Cluster

```bash
kind create cluster --name kind
```

### 2. Create the GitHub Runner Namespace

```bash
kubectl create namespace github-runner
```

### 3. Create a Secret with Your GitHub Token

Create a Kubernetes secret containing your GitHub Personal Access Token:

```bash
kubectl create secret generic gh-pat \
  --from-literal=pat=YOUR_GITHUB_TOKEN \
  --namespace github-runner
```

Replace `YOUR_GITHUB_TOKEN` with your actual GitHub PAT. You can generate one at <https://github.com/settings/tokens> with the following permissions:

- `repo` (Full control of private repositories)

### 4. Apply the Kubernetes Manifests

Apply all the manifests in the `integration/` directory:

```bash
kubectl apply -f integration/gr-sa.yaml
kubectl apply -f integration/gr-cluster-role.yaml
kubectl apply -f integration/gr-cluster-role-binding.yaml
```

This will create:

- Service account (`gr-sa`)
- ClusterRole with necessary pod permissions
- ClusterRoleBinding to bind the role to the service account

### 5. Configure Your Repository

Edit `integration/gr-deployment.yaml` and update the `REPO_URL` environment variable to point to your test repository:

```yaml
- name: REPO_URL
  value: https://github.com/YOUR_USERNAME/YOUR_REPO
```

Then apply the deployment:

```bash
kubectl apply -f integration/gr-deployment.yaml
```

This will create a deployment running the GitHub runner with the container hook

## Development Workflow

### Building and Testing Changes

When you make changes to the hook code, use the `rebuild-kind` task to rebuild the Docker image and update the running pod:

```bash
task rebuild-kind
```

This command will:

1. Build the Docker image with tag `runner:latest`
2. Load the image into the KIND cluster
3. Delete the existing GitHub runner pod (it will be recreated automatically by the deployment)

Alternatively, you can run the commands manually:

```bash
docker build . -t runner:latest
kind load docker-image runner:latest --name kind
kubectl get pods -n github-runner -l app=github-runner -o name | xargs -I {} kubectl -n github-runner delete {}
```

### Viewing Logs

Monitor the runner logs to see the hook in action:

```bash
kubectl logs -n github-runner -l app=github-runner -f
```

### Debugging

To enable debug output from the hook, uncomment the `DEBUG_HOOK` environment variable in the Dockerfile:

```dockerfile
ENV DEBUG_HOOK=1
```

Then rebuild and reload the image using `task rebuild-kind`.

## Running Tests

Run the test suite:

```bash
task test
```

Run tests with coverage:

```bash
task test:cover
```

## Linting

Lint the code before submitting:

```bash
task lint
```

## Testing Your Changes

1. Push a workflow to your test repository that uses container actions
2. The runner should pick up the job and execute it using the hook
3. Monitor the logs to verify the hook is working correctly

Example workflow file (`.github/workflows/test.yml`):

```yaml
name: Test Container Hook
on: [push]
jobs:
  test:
    runs-on: self-hosted
    container:
      image: alpine:latest
    steps:
      - run: echo "Hello from container!"
```

## Cleanup

To remove the test environment:

```bash
kubectl delete namespace github-runner
kind delete cluster --name kind
```
