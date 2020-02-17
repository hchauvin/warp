package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/pipelines"
	"github.com/hchauvin/warp/pkg/run/env"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"github.com/hchauvin/warp/pkg/terraform"
	"github.com/otiai10/copy"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Exec(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
) (rootModulePath string, err error) {
	rootModulePath, err = CreateRootModule(ctx, cfg, pipeline, name)
	if err != nil {
		return rootModulePath, err
	}
	tf, err := terraform.New(cfg, rootModulePath)
	if err != nil {
		return rootModulePath, err
	}
	if err := tf.Init(ctx); err != nil {
		return rootModulePath, err
	}
	if err := tf.Apply(ctx, true, nil); err != nil {
		return "", err
	}
	return rootModulePath, nil
}

func Destroy(
	ctx context.Context,
	cfg *config.Config,
	pipeline *pipelines.Pipeline,
	name names.Name,
) error {
	rootModulePath, err := CreateRootModule(ctx, cfg, pipeline, name)
	if err != nil {
		return err
	}
	tf, err := terraform.New(cfg, rootModulePath)
	if err != nil {
		return err
	}
	return tf.Destroy(ctx, true, nil)
}

func CreateRootModule(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name) (rootModulePath string, err error) {
	finalRootModulePath := filepath.Join(cfg.Path(cfg.OutputRoot), "terraform", name.String())
	if err := cleanUpRootModule(finalRootModulePath, false); err != nil {
		return "", fmt.Errorf("cannot clean up terraform root module '%s': %v", finalRootModulePath, err)
	}

	terraformConfigPath := cfg.Path(pipeline.Deploy.Terraform.Path)
	if err := copy.Copy(terraformConfigPath, finalRootModulePath); err != nil {
		return "", fmt.Errorf("cannot copy terraform config from '%s' to '%s': %v",
			terraformConfigPath, finalRootModulePath)
	}

	tfBackendFile, err := ioutil.TempFile(finalRootModulePath, "backend_*.tf")
	if err != nil {
		return "", err
	}
	defer tfBackendFile.Close()
	back, err := backendConfig(cfg)
	if err != nil {
		return "", err
	}
	if _, err := tfBackendFile.WriteString(back); err != nil {
		return "", fmt.Errorf("cannot write terraform backend config to %s: %v", tfBackendFile.Name(), err)
	}
	tfBackendFile.Close()

	tfvarsFile, err := ioutil.TempFile(finalRootModulePath, "*.auto.tfvars.json")
	if err != nil {
		return "", err
	}
	defer tfvarsFile.Close()
	tfvars, err := expandVars(ctx, cfg, pipeline, name)
	if err != nil {
		return "", err
	}
	tfvarsJson, err := json.Marshal(tfvars)
	if err != nil {
		return "", err
	}
	if _, err := tfvarsFile.Write(tfvarsJson); err != nil {
		return "", fmt.Errorf("cannot write terraform tfvars '%s': %v", tfvarsFile.Name(), err)
	}
	tfvarsFile.Close()

	return finalRootModulePath, nil
}

func cleanUpRootModule(path string, removeDotTerraform bool) error {
	if removeDotTerraform {
		return os.RemoveAll(path)
	}

	if stat, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	} else if (!stat.IsDir()) {
		return os.Remove(path)
	}

	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.Name() == ".terraform" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(path, fi.Name())); err != nil {
			return err
		}
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
		b.WriteString(strconv.Quote(v))
		b.WriteRune('\n')
	}
	b.WriteString("  }\n}\n")
	return b.String(), nil
}

func expandVars(ctx context.Context, cfg *config.Config, pipeline *pipelines.Pipeline, name names.Name) (vars map[string]interface{}, err error) {
	vars = make(map[string]interface{}, len(pipeline.Deploy.Terraform.Var))
	trans := env.NewTransformer(env.StackTemplateFuncs(cfg, name))
	for k, tpl := range pipeline.Deploy.Terraform.Var {
		v, err := trans.Get(ctx, tpl)
		if err != nil {
			return nil, fmt.Errorf("cannot expand template for variable '%s': %v", k, err)
		}
		vars[k] = v
	}
	return vars, nil
}
