// pipelines defines the pipeline API.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package pipelines

// Pipeline defines a deployment and test pipeline.  Its scope is
// everything beyond the build step and unit/integration tests.
type Pipeline struct {
	Stack Stack `yaml:"stack"`

	// Bases contains base files to merge into this pipeline, using
	// strategic merging.  The file names must be relative to the
	// root given in the warprc.toml configuration.
	Bases []string `yaml:"bases,omitempty"`

	// Deploy describes the deployment steps.
	Deploy Deploy `yaml:"deploy"`

	Dev Dev `yaml:"dev"`

	// Run describes the run configurations.
	Run []Run `yaml:"run,omitempty" patchStrategy:"merge" patchMergeKey:"names"`

	Path string `yaml:"-"`
}

type Stack struct {
	// Name is the names of the stack.  If it is not given, a names
	// is acquired for Family with the name_manager.  Either Name or
	// Family must be given.
	Name string `yaml:"name,omitempty"`

	// Family for the name_manager.
	Family string `yaml:"family,omitempty"`

	Variant string `yaml:"variant,omitempty"`
}

type Deploy struct {
	Container *Container `yaml:"container,omitempty"`
	Kustomize *Kustomize `yaml:"kustomize,omitempty"`
}

type Container struct {
	// Path to the container manifest.  The container manifest associates
	// image names to actual addresses.  This mapping is integrated
	// into the kustomization.
	Manifest string `yaml:"manifest,omitempty"`

	ParsedManifest ContainerManifest `yaml:"-"`

	Label []string `yaml:"label,omitempty"`
	Push  string   `yaml:"push,omitempty"`
}

type ContainerManifest map[string]ContainerManifestEntry

type ContainerManifestEntry struct {
	Ref string `json:"ref"`
}

type Kustomize struct {
	// Path to the kustomization file, relative to the root given
	// in the warprc.toml configuration.
	Path string `yaml:"path" validate:"required"`

	// Disable prefixing the names of Kubernetes resources with
	// the names of the Stack.
	DisableNamePrefix bool `yaml:"disableNamePrefix"`
}

type Run struct {
	Name       string   `yaml:"name" validate:"required,alphanum"`
	Tags       []string `yaml:"tags" validate:"alphanum"`
	WorkingDir string   `yaml:"workingDir"`
	Command    []string `yaml:"command" validate:"min=1"`
	Env        []string `yaml:"env"`
}

type Dev struct {
	Ksync       []Ksync       `yaml:"ksync,omitempty" patchStrategy:"merge" patchMergeKey:"names"`
	BrowserSync []BrowserSync `yaml:"browserSync,omitempty" patchStrategy:"merge" patchMergeKey:"names"`
	PortForward []PortForward `yaml:"portForward,omitempty" patchStrategy:"merge" patchMergeKey:"names"`
}

type Ksync struct {
	Name             string `yaml:"name" validate:"required,alphanum"`
	Selector         string `yaml:"selector" validate:"required"`
	Local            string `yaml:"local" validate:"required"`
	Remote           string `yaml:"remote" validate:"required"`
	DisableReloading bool   `yaml:"disableReloading"`
	LocalReadOnly    bool   `yaml:"localReadOnly"`
	RemoteReadOnly   bool   `yaml:"remoteReadOnly"`
}

type BrowserSync struct {
	Name      string                 `yaml:"name" validate:"required,alphanum"`
	LocalPort int                    `yaml:"localPort" validate:"min=1"`
	K8sProxy  KsyncK8sProxy          `yaml:"k8sProxy"`
	Config    map[string]interface{} `yaml:"config"`
	Paths     []string               `yaml:"paths" validate:"min=1"`
}

type KsyncK8sProxy struct {
	Selector   string `yaml:"selector" validate:"required"`
	RemotePort int    `yaml:"remotePort" validate:"min=1"`
}

type PortForward struct {
	Name       string `yaml:"name" validate:"required,alphanum"`
	Selector   string `yaml:"selector" validate:"required"`
	LocalPort  int    `yaml:"localPort" validate:"min=1"`
	RemotePort int    `yaml:"remotePort" validate:"min=1"`
}
