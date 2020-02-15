package terraform

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/proc"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"io/ioutil"
	"os"
	"path/filepath"
	"github.com/otiai10/copy"
	"strconv"
	"strings"
)

func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
) error {
	return applyDestroy(ctx, cfg, pipeline, name, "apply")
}

func Destroy(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
) error {
	return applyDestroy(ctx, cfg, pipeline, name, "destroy")
}

func applyDestroy(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
	command string,
) error {
	terraformPath, err := cfg.ToolPath(config.TerraformCLI)
	if err != nil {
		return err
	}

	finalTerraformConfigPath := filepath.Join(cfg.Path(cfg.OutputRoot), "terraform", name.String())
	if err := os.RemoveAll(finalTerraformConfigPath); err != nil {
		return err
	}

	terraformConfigPath := cfg.Path(pipeline.Deploy.Terraform.Path)
	if err := copy.Copy(terraformConfigPath, finalTerraformConfigPath); err != nil {
		return fmt.Errorf("cannot copy terraform config from '%s' to '%s': %v",
			terraformConfigPath, finalTerraformConfigPath)
	}

	tfBackendFile, err := ioutil.TempFile(finalTerraformConfigPath, "backend_*.tf")
	if err != nil {
		return err
	}
	defer tfBackendFile.Close()
	back, err := backendConfig(cfg)
	if err != nil {
		return err
	}
	if _, err := tfBackendFile.WriteString(back); err != nil {
		return fmt.Errorf("cannot write terraform backend config to %s: %v", tfBackendFile.Name(), err)
	}
	tfBackendFile.Close()

	args := []string{command, "-auto-approve"}
	trans := env.NewTransformer(env.StackTemplateFuncs(cfg, name))
	for k, tpl := range pipeline.Deploy.Terraform.Var {
		v, err := trans.Get(ctx, tpl)
		if err != nil {
			return fmt.Errorf("cannot expand template for variable '%s': %v", k, err)
		}
		args = append(args, "-var", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, finalTerraformConfigPath)

	cmd := proc.GracefulCommandContext(ctx, terraformPath, args...)
	cfg.Logger().Pipe(config.TerraformCLI.LogDomain(), cmd)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not run terraform on '%s': %v", finalTerraformConfigPath, err)
	}
	return nil
}

func backendConfig(
	cfg *config.Config,
) (string, error) {
	var b strings.Builder
	b.WriteString("terraform {\n  backend \"")
	b.WriteString(cfg.Terraform.Backend)
	b.WriteString("\" {\n")
	for k, v := range cfg.Terraform.Var {
		b.WriteString("    ")
		b.WriteString(k)
		b.WriteString(" = ")
		b.WriteRune('"')
		b.WriteString(strconv.Quote(v))
		b.WriteString("\"\n")
	}
	b.WriteString("  }\n}\n")
	return b.String(), nil
}
