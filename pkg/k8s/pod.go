// Package k8s  is a kubernetes client for kubernetes runner hook
package k8s

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reMarkable/k8s-hook/pkg/types"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1Meta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
)

type K8sClient struct {
	client kubernetes.Interface
	config *rest.Config
	ctx    context.Context
}

var (
	ErrPodTimeout   = errors.New("timeout waiting for pod to be ready")
	ErrNotSupported = errors.New("feature not supported in kubernetes hook")
)

type PodType int

const (
	PodTypeJob PodType = iota
	PodTypeContainerStep
)

const JobVolumeName = "work"

func NewK8sClient() (*K8sClient, error) {
	var clientset *kubernetes.Clientset
	var config *rest.Config
	var err error
	// Allow running outside the cluster for testing purposes
	// creates the in-cluster config
	config, err = rest.InClusterConfig()
	// Fall back to local kubernetes auth
	if err != nil {
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &K8sClient{client: clientset, ctx: context.Background(), config: config}, nil
}

func (c *K8sClient) CreatePod(args types.InputArgs, podType PodType) (string, error) {
	podSpec := c.preparePodSpec(args.Container, podType)
	if podType == PodTypeJob {
		copyExternals()
	}
	if args.Container.CreateOptions != "" {
		return "", fmt.Errorf("%w: CreateOptions provided: %s", ErrNotSupported, args.Container.CreateOptions)
	}

	pod, err := c.client.CoreV1().Pods(c.GetNS()).Create(c.ctx, podSpec, v1Meta.CreateOptions{})
	if err != nil {
		var statusErr *k8sErrors.StatusError
		if errors.As(err, &statusErr) {
			c.checkPermissions()
		}
		return "", err
	}

	if err = c.waitForPodReady(pod.Name); err != nil {
		return "", err
	}

	return pod.Name, nil
}

func (c *K8sClient) ExecStepInPod(name string, args types.InputArgs) error {
	containerPath, runnerPath, err := c.writeRunScript(args)
	defer func() {
		err = os.Remove(runnerPath)
		if err != nil {
			slog.Warn("Failed to remove temporary run script", "err", err)
		}
	}()
	if err != nil {
		slog.Error("Failed to write run script", "err", err)
		return err
	}

	req := c.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(c.GetNS()).
		SubResource("exec")
	req.VersionedParams(&v1.PodExecOptions{
		Container: "job",
		Command:   []string{"sh", "-e", containerPath},
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	slog.Debug("trying to exec", "req", req.URL().String(), "name", name, "command", containerPath)
	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		slog.Error("Failed to setup remote executor", "err", err)
		return err
	}

	opt := remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	}
	cancelCtx, cancel := context.WithCancel(c.ctx)
	defer cancel()
	if err := exec.StreamWithContext(cancelCtx, opt); err != nil {
		slog.Error("Failed to stream context", "err", err)
		return err
	}

	return nil
}

func (c *K8sClient) PrunePods() error {
	podList, err := c.client.CoreV1().Pods(c.GetNS()).List(c.ctx, v1Meta.ListOptions{
		LabelSelector: "runner-pod=" + c.GetRunnerPodName(),
	})
	if err != nil {
		return err
	}

	for _, pod := range podList.Items {
		slog.Info("Pruning pod", "pod", pod.Name)
		err = c.DeletePod(pod.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *K8sClient) DeletePod(name string) error {
	err := c.client.CoreV1().Pods(c.GetNS()).Delete(c.ctx, name, v1Meta.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (c *K8sClient) preparePodSpec(cont types.ContainerDefinition, podType PodType) *v1.Pod {
	jobContainer := v1.Container{
		Name:    "job",
		Image:   cont.Image,
		Command: []string{"tail"},
		Args:    []string{"-f", "/dev/null"},
		Env: []v1.EnvVar{
			{
				Name: "GITHUB_ACTIONS", Value: "true",
			},
			{
				Name: "CI", Value: "true",
			},
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      JobVolumeName,
				MountPath: "/__w",
			},
			{
				Name:      JobVolumeName,
				MountPath: "/github/home",
				SubPath:   "_temp/_github_home",
			},
			{
				Name:      JobVolumeName,
				MountPath: "/github/workflow",
				SubPath:   "_temp/_github_workflow",
			},
		},
	}
	if os.Getenv("ENV_DISABLE_IMAGE_PULL") != "true" {
		jobContainer.ImagePullPolicy = v1.PullIfNotPresent
	}

	for k, v := range cont.EnvironmentVariables {
		jobContainer.Env = append(jobContainer.Env, v1.EnvVar{Name: k, Value: v})
	}

	if cont.WorkingDirectory != "" {
		jobContainer.WorkingDir = cont.WorkingDirectory
	}
	var name string
	if podType == PodTypeContainerStep {
		workspace := os.Getenv("GITHUB_WORKSPACE")
		// remove anything before _work to get the subpath
		i := strings.LastIndex(workspace, "_work/")
		workspaceRelativePath := workspace[i+len("_work/"):]

		name = c.GetRunnerPodName() + "-step-" + podPostfix()
		jobContainer.VolumeMounts = append([]v1.VolumeMount{
			{
				Name:      JobVolumeName,
				MountPath: "/github/workspace",
				SubPath:   workspaceRelativePath,
			},
			{
				Name:      JobVolumeName,
				MountPath: "/github/file_commands",
				SubPath:   "_temp/_runner_file_commands",
			},
		}, jobContainer.VolumeMounts...)
	} else {
		name = c.GetRunnerPodName() + "-workflow"
		jobContainer.VolumeMounts = append([]v1.VolumeMount{
			{
				Name:      JobVolumeName,
				MountPath: "/__e",
				SubPath:   "externals",
			},
		}, jobContainer.VolumeMounts...)
	}
	podSpec := &v1.Pod{
		ObjectMeta: v1Meta.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"runner-pod": c.GetRunnerPodName(),
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers:    []v1.Container{jobContainer},
			Volumes: []v1.Volume{
				{
					Name: JobVolumeName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							ClaimName: c.GetVolumeClaimName(),
						},
					},
				},
			},
		},
	}
	if os.Getenv("ENV_USE_KUBE_SCHEDULER") == "true" {
		podSpec.Spec.Affinity = &v1.Affinity{
			NodeAffinity: &v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: v1.NodeSelectorOpIn,
									Values:   []string{c.GetRunnerPodName()},
								},
							},
						},
					},
				},
			},
		}
	} else {
		podSpec.Spec.NodeName, _ = c.GetPodNodeName(c.GetRunnerPodName())
	}
	if cont.Registry != nil {
		secretName, err := c.createImagePullSecret(cont)
		if err != nil {
			slog.Warn("Failed to create pull secret", "err", err)
		} else {
			podSpec.Spec.ImagePullSecrets = []v1.LocalObjectReference{
				{
					Name: secretName,
				},
			}
		}
	}
	return podSpec
}

func (c *K8sClient) waitForPodReady(name string) error {
	var err error
	timeout := getPrepareJobTimeout()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	factory := informers.NewSharedInformerFactoryWithOptions(
		c.client,
		time.Second*10,
		informers.WithNamespace(c.GetNS()),
		informers.WithTweakListOptions(func(opt *v1Meta.ListOptions) {
			opt.FieldSelector = fields.OneTermEqualSelector("metadata.name", name).String()
		}),
	)

	informer := factory.Core().V1().Pods().Informer()

	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: podEventHandler(cancel, &err),
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	factory.Start(ctx.Done())
	<-ctx.Done() // Wait until pod is running, failed, or timeout
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timeout waiting for %d seconds for pod to be ready: %w", timeout, ErrPodTimeout)
	}

	return err
}
