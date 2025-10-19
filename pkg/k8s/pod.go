// Package k8s  is a kubernetes client for kubernetes runner hook
package k8s

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/reMarkable/k8s-hook/pkg/types"
	v1 "k8s.io/api/core/v1"
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
	client *kubernetes.Clientset
	ctx    context.Context
	config *rest.Config
}

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

func (c *K8sClient) CreatePod(args types.InputArgs) (string, error) {
	podName := c.getRunnerPodName() + "-workflow"
	jobContainer := v1.Container{
		Name:            "job",
		Image:           args.Container.Image,
		ImagePullPolicy: v1.PullIfNotPresent,
		Command:         []string{"tail"},
		Args:            []string{"-f", "/dev/null"},
		Env: []v1.EnvVar{
			{
				Name: "GITHUB_ACTIONS", Value: "true",
			},
			{
				Name: "CI", Value: "true",
			},
		},
	}
	for k, v := range args.Container.EnvironmentVariables {
		jobContainer.Env = append(jobContainer.Env, v1.EnvVar{Name: k, Value: v})
	}
	if args.Container.WorkingDirectory != "" {
		jobContainer.WorkingDir = args.Container.WorkingDirectory
	}
	podSpec := &v1.Pod{
		ObjectMeta: v1Meta.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"runner-pod": c.getRunnerPodName(),
			},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers:    []v1.Container{jobContainer},
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
									Values:   []string{c.getRunnerPodName()},
								},
							},
						},
					},
				},
			},
		}
	} else {
		podSpec.Spec.NodeName, _ = c.GetPodNodeName(c.getRunnerPodName())
	}

	pod, err := c.client.CoreV1().Pods(c.GetNS()).Create(c.ctx, podSpec, v1Meta.CreateOptions{})
	if err != nil {
		return "", err
	}
	if err = c.waitForPodReady(pod.Name); err != nil {
		return "", err
	}
	return pod.Name, nil
}

func (c *K8sClient) ExecStepInPod(name string, command string, args []string) (string, error) {
	req := c.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(c.GetNS()).
		SubResource("exec")
	cl := []string{command}
	req.VersionedParams(&v1.PodExecOptions{
		Container: "job",
		Command:   append(cl, args...),
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	slog.Debug("trying to exec", "req", req.URL().String(), "name", name, "command", command, "args", args)
	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		slog.Error("Failed to setup remote executor", "err", err)
		return "", err
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
		return "", err
	}

	return "", nil
}

func (c *K8sClient) waitForPodReady(name string) error {
	var err error
	t := os.Getenv("ACTIONS_RUNNER_PREPARE_JOB_TIMEOUT_SECONDS")

	timeout, convErr := strconv.Atoi(t)
	if convErr != nil {
		slog.Info("Using default timeout of 600 seconds for preparing job pod")
		timeout = 600
	}
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

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj any) {
			pod := newObj.(*v1.Pod)
			slog.Debug("Pod status changed", "pod", pod.Name, "status", pod.Status.Phase)
			for _, c := range pod.Status.ContainerStatuses {
				slog.Debug("Container state", "name", c.Name, "state", c.State)
				if c.State.Waiting != nil && c.State.Waiting.Reason == "ImagePullBackOff" {
					slog.Error("Runner failed to pull image", "pod", pod.Name, "reason", c.State.Waiting.Reason, "message", c.State.Waiting.Message)
					err = fmt.Errorf("failed to pull image: %s", c.State.Waiting.Message)
					cancel()
				}
				if c.State.Waiting != nil && c.State.Waiting.Reason == "CrashLoopBackOff" {
					slog.Error("Runner image crashing on startup", "pod", pod.Name, "reason", c.State.Waiting.Reason, "message", c.State.Waiting.Message)
					err = fmt.Errorf("image crashing on startup: %s", c.State.Waiting.Message)
					cancel()
				}
			}
			if pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodFailed {
				if pod.Status.Phase == v1.PodFailed {
					err = fmt.Errorf("pod failed")
				}
				cancel()
			}
		},
	})
	factory.Start(ctx.Done())
	<-ctx.Done() // Wait until pod is running, failed, or timeout
	if ctx.Err() == context.DeadlineExceeded && err == nil {
		return fmt.Errorf("timeout waiting for %d seconds for pod to be ready", timeout)
	}
	return err
}

func (c *K8sClient) PrunePods() error {
	podList, err := c.client.CoreV1().Pods(c.GetNS()).List(c.ctx, v1Meta.ListOptions{
		LabelSelector: "runner-pod=" + c.getRunnerPodName(),
	})
	if err != nil {
		re

	return nil
}

func (c *K8sClient) DeletePod(name string) error {
	err := c.client.CoreV1().Pods(c.GetNS()).Delete(c.ctx, name, v1Meta.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
