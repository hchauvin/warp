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
	dir string
	terraformPath string
}

func New(cfg *config.Config, dir string) (*Terraform, error) {
	terraformPath, err := cfg.ToolPath(config.TerraformCLI)
	if err != nil {
		return nil, err
	}
	return &Terraform{cfg, dir, terraformPath}, nil
}

func (tf *Terraform) Init(ctx context.Context) error {
	args := []string{"init"}

	cmd := proc.GracefulCommandContext(ctx, tf.terraformPath, args...)
	cmd.Dir = tf.dir
	tf.cfg.Logger().Pipe(config.TerraformCLI.LogDomain(), cmd)
	return cmd.Run()
}

func (tf *Terraform) Apply(ctx context.Context, autoApprove bool, vars []string) error {
	return tf.applyOrDestroy(ctx, "apply", autoApprove, vars)
}

func (tf *Terraform) Destroy(ctx context.Context, autoApprove bool, vars []string) error {
	return tf.applyOrDestroy(ctx, "destroy", autoApprove, vars)
}

func (tf *Terraform) applyOrDestroy(ctx context.Context, command string, autoApprove bool, vars []string) error {
	// TODO: explore -refresh=false, sometimes, for performance reasons.
	args := []string{command, "-input=false", "-refresh=true"}
	if autoApprove {
		args = append(args, "-auto-approve")
	}
	args = appendVarArgs(args, vars)
	// args = append(args, tf.dir)

	cmd := proc.GracefulCommandContext(ctx, tf.terraformPath, args...)
	cmd.Dir = tf.dir
	tf.cfg.Logger().Pipe(config.TerraformCLI.LogDomain() + ":" + command, cmd)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not run terraform on '%s': %v", tf.dir, err)
	}
	return nil
}

type Output struct {
	Sensitive bool            `json:"sensitive"`
	Type      json.RawMessage `json:"type"`
	Value     json.RawMessage `json:"value"`
}

func (tf *Terraform) Outputs(ctx context.Context) (map[string]Output, error) {
	terraformPath, err := tf.cfg.ToolPath(config.TerraformCLI)
	if err != nil {
		return nil, err
	}

	args := []string{"output", "-json"}

	cmd := proc.GracefulCommandContext(ctx, terraformPath, args...)
	cmd.Dir = tf.dir
	tf.cfg.Logger().PipeStderr(config.TerraformCLI.LogDomain() + ":output", cmd)

	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("could not run terraform in '%s': %v", tf.dir, err)
	}

	var m map[string]Output
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func appendVarArgs(args []string, vars []string) (nextArgs []string) {
	for _, v := range vars {
		args = append(args, "-var", v)
	}
	return args
}