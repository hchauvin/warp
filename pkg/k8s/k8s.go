// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package k8s implements Kubernetes-specific code.
package k8s

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/proc"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"path/filepath"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// K8s wraps a Kubernetes client.
type K8s struct {
	cfg        *config.Config
	Clientset  *kubernetes.Clientset
	DynClient  dynamic.Interface
	restconfig *rest.Config
	Ports      *Ports
}

// New creates a new Kubernetes client wrapper.
func New(cfg *config.Config) (*K8s, error) {
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
	restconfig, err := clientLoader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot get Kubernetes client config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	dynClient, err := dynamic.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	client := &K8s{
		cfg:        cfg,
		Clientset:  clientset,
		DynClient:  dynClient,
		restconfig: restconfig,
	}
	client.Ports = newPorts(client)
	return client, nil
}

// Apply executes "kubectl apply".  The resources matching the label
// selector are pruned.
func (k8s *K8s) Apply(ctx context.Context, resourcesPath string, labelSelector string) error {
	cmd, err := k8s.KubectlCommandContext(ctx, "apply",
		"-f", resourcesPath,
		"--force",
		"--prune", "-l",
		labelSelector,
		"--grace-period=0",
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

// KubectlCommandContext returns a command object to call the kubectl command.
func (k8s *K8s) KubectlCommandContext(ctx context.Context, args ...string) (*exec.Cmd, error) {
	kubectlPath, err := k8s.cfg.ToolPath(config.Kubectl)
	if err != nil {
		return nil, err
	}

	return k8s.KubectlLikeCommandContext(ctx, kubectlPath, args...)
}

// KubectlLikeCommandContext returns a command object to call the kubectl command,
// or a command that behave similarly concerning the "--context" option
// and the "KUBECONFIG" environment variable.
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
