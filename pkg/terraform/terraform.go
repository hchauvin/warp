package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/proc"
)

type Terraform struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Terraform {
	return &Terraform{cfg}
}

func (tf *Terraform) Outputs(ctx context.Context) (map[string]string, error) {
	terraformPath, err := tf.cfg.ToolPath(config.TerraformCLI)
	if err != nil {
		return nil, err
	}

	args := []string{"output", "-json"}

	cmd := proc.GracefulCommandContext(ctx, terraformPath, args...)
	tf.cfg.Logger().Pipe(config.TerraformCLI.LogDomain(), cmd)

	b, err := cmd.Output()
	if err != nil {
		return make(map[string]string), fmt.Errorf("could not run terraform: %v", err)
	}

	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	outputs := make(map[string]string, len(m))
	for k, v := range m {
		s, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		outputs[k] = string(s)
	}
	return outputs, nil
}