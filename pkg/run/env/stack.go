package env

import (
	"context"
	"github.com/hchauvin/warp/pkg/config"
	"github.com/hchauvin/warp/pkg/stacks/names"
	"text/template"
)

func StackTemplateFuncs(cfg *config.Config, name names.Name) TemplateFuncs {
	return &stackTemplateFuncs{
		name,
	}
}

type stackTemplateFuncs struct {
	name names.Name
}

func (funcs *stackTemplateFuncs) TxtFuncMap(ctx context.Context) template.FuncMap {
	return map[string]interface{}{
		"namePrefix": func() string {
			return funcs.name.DNSName() + "-"
		},
	}
}
