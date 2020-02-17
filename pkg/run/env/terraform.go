package env

import (
	"context"
	"fmt"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/terraform"
	"sync"
	"text/template"
)

func TerraformTemplateFuncs(cfg *config.Config, tf *terraform.Terraform) TemplateFuncs {
	return &terraformTemplateFuncs{
		tf: tf,
	}
}

type terraformTemplateFuncs struct {
	tf *terraform.Terraform
	outputs map[string]terraform.Output
	outputsMut sync.Mutex
}

func (funcs *terraformTemplateFuncs) TxtFuncMap(ctx context.Context) template.FuncMap {
	return map[string]interface{}{
		"terraformOutputScalar": func(name string) (string, error) {
			funcs.outputsMut.Lock()
			defer funcs.outputsMut.Unlock()
			if funcs.outputs == nil {
				outputs, err := funcs.tf.Outputs(ctx)
				if err != nil {
					return "", err
				}
				funcs.outputs = outputs
			}
			value, ok := funcs.outputs[name]
			if !ok {
				return "", fmt.Errorf("output '%s' not found", name)
			}
			return fmt.Sprintf("%s", value.Value), nil
		},
	}
}
