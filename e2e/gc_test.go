package e2e

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/k8s"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/stacks"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/hchauvin/warp/pkg/warp"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"testing"
)

func TestGc(t *testing.T) {
	godotenv.Load("../.env")

	gcCfg := &warp.GcCfg{
		WorkingDir: "../examples",
		ConfigPath: ".warprc.toml",
	}

	fullPath := filepath.Join(gcCfg.WorkingDir, gcCfg.ConfigPath)
	cfg, err := config.Read(fullPath)
	assert.NoError(t, err)

	pipeline, err := pipelines.Read(cfg, "gc")
	assert.NoError(t, err)

	name, holdErrc, releaseName, err := stacks.Hold(cfg, pipeline)
	assert.NoError(t, err)
	defer releaseName()
	go func() {
		for err := range holdErrc {
			assert.Fail(t, "detached error", err)
		}
	}()

	detachedErrc := make(chan error)
	err = stacks.Exec(context.Background(), cfg, pipeline, &stacks.ExecConfig{
		Name: *name,
		Tail: true,
		Run:  []string{"test"},
	}, detachedErrc)
	assert.NoError(t, err)
	go func() {
		for err := range detachedErrc {
			assert.Fail(t, "detached error", err)
		}
	}()

	k8sClient, err := k8s.New(cfg)
	assert.NoError(t, err)
	defer k8sClient.Ports.CancelForwarding()

	echoServiceName := fmt.Sprintf("%s-echo", name.DNSName())
	_, err = k8sClient.Clientset.CoreV1().
		Services("default").
		Get(echoServiceName, metav1.GetOptions{})
	assert.NoError(t, err)

	err = k8sClient.Gc(
		context.Background(),
		cfg,
		names.Name{Family: name.Family, ShortName: name.ShortName},
		&k8s.GcOptions{})
	assert.NoError(t, err)

	_, err = k8sClient.Clientset.CoreV1().
		Services("default").
		Get(echoServiceName, metav1.GetOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
