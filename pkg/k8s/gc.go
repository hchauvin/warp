package k8s

import (
	"context"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/stacks/names"
)

func Gc(ctx context.Context, cfg *config.Config, name names.Name) error {
	kubectlPath, err := cfg.Tools[config.Kubectl].Resolve()
	if err != nil {
		return err
	}

	cmd := proc.GracefulCommandContext(
		ctx,
		kubectlPath,
		"delete",
		"--all-namespaces",
		"-l",
		Labels{
			StackLabel: name.DNSName(),
		}.String(),
		"all")
	cfg.Logger().Pipe(logDomain+":gc", cmd)
	return cmd.Run()
}
