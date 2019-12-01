package k8s

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/proc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"path/filepath"
)

func NewClient(cfg *config.Config) (*kubernetes.Clientset, error) {
	var kubeconfig string
	var defaultContext string
	if cfg.Kubernetes != nil {
		kubeconfig = cfg.Kubernetes.KubeconfigEnvVar
		defaultContext = cfg.Kubernetes.DefaultContext
	}
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	if kubeconfig == "" {
		if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			return nil, fmt.Errorf("cannot determine path to kubeconfig")
		}
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.Precedence = filepath.SplitList(kubeconfig)
	overrides := &clientcmd.ConfigOverrides{}
	overrides.CurrentContext = defaultContext

	clientLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		overrides)
	config, err := clientLoader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot get Kubernetes client config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

type K8s struct {
	cfg       *config.Config
	clientset *kubernetes.Clientset
	Ports     *Ports
}

func New(cfg *config.Config) (*K8s, error) {
	clientset, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	client := &K8s{
		cfg:       cfg,
		clientset: clientset,
	}
	client.Ports = newPorts(client)
	return client, nil
}

func (k8s *K8s) Apply(ctx context.Context, resourcesPath string, labelSelector string) error {
	cmd, err := k8s.KubectlCommandContext(ctx, "apply",
		"-f", resourcesPath,
		"--force",
		"--prune", "-l",
		labelSelector,
	)
	if err != nil {
		return err
	}
	k8s.cfg.Logger().Pipe(config.Kubectl.LogDomain(), cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not apply kubectl patch '%s': %v", resourcesPath, err)
	}
	return nil
}

func (k8s *K8s) DeleteAll(ctx context.Context, labelSelector string) error {
	cmd, err := k8s.KubectlCommandContext(ctx, "delete",
		"all",
		"-l", labelSelector,
	)
	if err != nil {
		return err
	}
	k8s.cfg.Logger().Pipe(logDomain+":kubectl", cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not delete k8s resources: %v", err)
	}
	return nil
}

func (k8s *K8s) KubectlCommandContext(ctx context.Context, args ...string) (*exec.Cmd, error) {
	kubectlPath, err := k8s.cfg.Tools[config.Kubectl].Resolve()
	if err != nil {
		return nil, err
	}

	return k8s.KubectlLikeCommandContext(ctx, kubectlPath, args...)
}

func (k8s *K8s) KubectlLikeCommandContext(ctx context.Context, command string, args ...string) (*exec.Cmd, error) {
	kcfg := k8s.cfg.Kubernetes
	if kcfg == nil {
		cmd := proc.GracefulCommandContext(ctx, command, args...)
		return cmd, nil
	}

	if kcfg.DefaultContext != "" {
		args = append([]string{"--context", kcfg.DefaultContext}, args...)
	}
	cmd := proc.GracefulCommandContext(ctx, command, args...)

	if kcfg.KubeconfigEnvVar != "" {
		if len(cmd.Env) == 0 {
			cmd.Env = os.Environ()
		}
		cmd.Env = append(cmd.Env, "KUBECONFIG="+kcfg.KubeconfigEnvVar)
	}

	return cmd, nil
}
