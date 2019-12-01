package k8s

import (
	"context"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/stacks/names"
)

func (k8s *K8s) Gc(ctx context.Context, cfg *config.Config, name names.Name) error {
	cmd, err := k8s.KubectlCommandContext(
		ctx,
		"delete",
		"--all-namespaces",
		"-l",
		Labels{
			StackLabel: name.DNSName(),
		}.String(),
		"all")
	if err != nil {
		return err
	}
	cfg.Logger().Pipe(logDomain+":gc", cmd)
	return cmd.Run()
}
