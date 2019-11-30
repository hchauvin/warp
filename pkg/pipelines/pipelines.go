// pipelines defines the pipeline API.
//
// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin
package pipelines

// Pipeline defines a deployment and test pipeline.  Its scope is
// everything beyond the build step and unit/integration tests.
type Pipeline struct {
	// Stack gives the identity of the stacks created by the pipeline.
	Stack Stack `yaml:"stack"`

	// Bases contains base files to merge into this pipeline, using
	// strategic merging.  The file names must be relative to the
	// root given in the warprc.toml configuration.
	//
	// Loops in the inheritance chain are forbidden and explicitly
	// controlled for.
	Bases []string `yaml:"bases,omitempty"`

	// Deploy describes the deployment steps.
	Deploy Deploy `yaml:"deploy"`

	// Dev describes the dev tools.  They are enabled when running
	// a pipeline in dev mode.  They can be used interactively,
	// during local development or debugging.
	Dev Dev `yaml:"dev"`

	// Command describes the command configurations.  Command configurations
	// are used to spawn processes that can access the stack, with
	// port forwarding and other mechanisms, between the setup and
	// the tear down of the stack.
	Commands []Command `yaml:"commands,omitempty" patchStrategy:"merge" patchMergeKey:"names" validate:"dive"`

	// Path is the absolute path to the pipeline definition.  It is
	// resolved by Read.
	Path string `yaml:"-"`
}

// Stack gives the identity of the stacks created by the pipeline.
type Stack struct {
	// Name is the names of the stack.  If it is not given, a names
	// is acquired for Family with the name_manager.  Either Name or
	// Family must be given.
	Name string `yaml:"name,omitempty"`

	// Family for the name_manager.
	Family string `yaml:"family,omitempty"`

	// Variant is the variant of the stack.  Stacks of the same family
	// can indeed be produced by multiple pipelines to share resources.
	// Variant is a way to differentiate between them.
	Variant string `yaml:"variant,omitempty"`
}

// Deploy describes the deployment steps.
type Deploy struct {
	// Container describes the deployment steps relative to
	// containerization.
	Container *Container `yaml:"container,omitempty"`

	// Kustomize describes the Kustomize config to use to
	// deploy to a Kubernetes cluster.  If it is omitted,
	// the stack is not deployed to Kubernetes.
	Kustomize *Kustomize `yaml:"kustomize,omitempty"`
}

// Container describes the deployment steps relative to
// containerization.
type Container struct {
	// Manifest is the path to the container manifest, relative to
	// the workspace root (see config.Config.WorkspaceDir).
	//
	// The container manifest associates image names, as they are
	// referenced in configuration files (such as Kubernetes resources)
	// to actual addresses.
	Manifest string `yaml:"manifest,omitempty"`

	// ParsedManifest is populated by Read and contains the parsed
	// version of the Manifest.
	ParsedManifest ContainerManifest `yaml:"-"`

	// Label gives a list of additional labels to apply to all the
	// images in the manifest.  New images are created with these labels,
	// and pushed to the container registry.
	Label []string `yaml:"label,omitempty"`

	// Push gives an alternative container registry to use for the
	// stack.  The images in the manifest are tagged for this registry,
	// then pushed to this registry.  This is useful, e.g., when
	// wants to put images to a production registry.
	Push string `yaml:"push,omitempty"`
}

// ContainerManifest associates "shortcut" image names to actual container
// image reference.  A container manifest is typically an output of the
// build process.
type ContainerManifest map[string]ContainerManifestEntry

type ContainerManifestEntry struct {
	Ref string `json:"ref"`
}

// Kustomize describes the Kustomize config to use to
// deploy to a Kubernetes cluster.
type Kustomize struct {
	// Path to the kustomization file, relative to the root given
	// in the warprc.toml configuration.
	Path string `yaml:"path" validate:"required"`

	// Disable prefixing the names of Kubernetes resources with
	// the names of the Stack.
	DisableNamePrefix bool `yaml:"disableNamePrefix"`
}

// Command describes a command configuration.  Command configurations
// are used to spawn processes that can access the stack, with
// port forwarding and other mechanisms, between the setup and
// the tear down of the stack.
type Command struct {
	// Name is the name of the run configuration.
	Name string `yaml:"name" validate:"required,name"`

	// Description describes the run configuration.
	Description string `yaml:"description,omitempty"`

	// Tags are arbitrary tags associated with the run configuration.
	// They can be used to organize run configurations.
	Tags []string `yaml:"tags" validate:"name"`

	// WorkingDir is the working directory where to execute the command,
	// relative to the workspace root (see config.Config.WorkspaceDir).
	// If omitted, the working directory is not changed.
	WorkingDir string `yaml:"workingDir,omitempty"`

	// Command is the command to execute.
	Command []string `yaml:"command" validate:"min=1"`

	// Env is a list of environment variables, specified as "name=value"
	// strings.  The values can be templated.  The template functions
	// allow, e.g., to request service addresses, configuration values,
	// that come from the deployment stage.
	Env []string `yaml:"env"`
}

// Dev describes the dev tools.  They are enabled when running
// a pipeline in dev mode.  They can be used interactively,
// during local development or debugging.
type Dev struct {
	// Ksync lists ksync configurations.
	Ksync []Ksync `yaml:"ksync,omitempty" patchStrategy:"merge" patchMergeKey:"names" validate:"dive"`

	// BrowserSync lists browser-sync configurations.
	BrowserSync []BrowserSync `yaml:"browserSync,omitempty" patchStrategy:"merge" patchMergeKey:"names" validate:"dive"`

	// PortForward lists port forwarding configurations.
	PortForward []PortForward `yaml:"portForward,omitempty" patchStrategy:"merge" patchMergeKey:"names" validate:"dive"`
}

// Ksync is a dev tool to synchronize local files (such as source files) and the
// files of Kubernetes pods.  ksync is roughly the equivalent, in the context of
// a multi-node system, of mounting a local volume.  It can be used to update
// pods when the source file changes, without having to build the containers
// again.
//
// Ksync is a tiny wrapper on top of the popular https://github.com/ksync/ksync.
type Ksync struct {
	// Name is the name of the ksync configuration.
	Name string `yaml:"name" validate:"required,name"`

	// Selector is a Kubernetes selector to select the pods to synchronize.
	Selector string `yaml:"selector" validate:"required"`

	// Local is the path to the local file or folder to synchronize.
	Local string `yaml:"local" validate:"required"`

	// Remote is the path to the remote file or folder to synchronize.
	// The file or folder lives on the pod's file system.
	Remote string `yaml:"remote" validate:"required"`

	// DisableReloading disables reloading the pod when the files change.
	// Reloading is enabled by default.
	DisableReloading bool `yaml:"disableReloading"`

	// LocalReadOnly sets the local file system as read only.  This means
	// that only changes coming from the local file system will be mirrored
	// to the pod file system.  Changes in the pod file system won't make
	// it back to the local file system.
	LocalReadOnly bool `yaml:"localReadOnly"`

	// RemoteReadOnly sets the pod file system as read only.  This means
	// that only changes coming from the od file system will be mirrored
	// to the local file system.  Changes in the local file system won't make
	// it back to the pod file system.
	RemoteReadOnly bool `yaml:"remoteReadOnly"`
}

// BrowserSync is a dev tool to perform live reload in a browser: when the code
// for a page changes, the page is automatically refreshed.  This process saves
// times and provides for an improved developer experience.  BrowserSync
// only makes sense when it comes with, e.g., Ksync.
//
// BrowserSync is a tiny wrapper on top of the popular https://www.browsersync.io/.
type BrowserSync struct {
	// Name is the name of the browser-sync configuration.
	Name string `yaml:"name" validate:"required,name"`

	// LocalPort is the local port the website can be accessed to, using a browser.
	LocalPort int `yaml:"localPort" validate:"min=1"`

	// K8sProxy gives the Kubernetes pod to listen to.  browser-sync acts as a proxy
	// between this Kubernetes port and "LocalPort".  This proxy is responsible
	// for injecting custom JavaScript code to enable live reloading.
	K8sProxy BrowserSyncK8sProxy `yaml:"k8sProxy"`

	// Config gives an additional map of configuration options to pass
	// to browser-sync.  See https://www.browsersync.io/docs/options.
	//
	// One important option is reloadDelay.  This option must typically be manually
	// tuned to account for the round trip with a remote server (the local file
	// changes must be sent over the network).
	Config map[string]interface{} `yaml:"config"`

	// Paths is a list of local file paths/path specs that are watched.
	Paths []string `yaml:"paths" validate:"min=1"`
}

// BrowserSyncK8sProxy configures browser-sync as a proxy to a Kubernetes pod.
type BrowserSyncK8sProxy struct {
	// Selector is a Kubernetes selector.  The first pod that matches the
	// selector is used for the proxy.
	Selector string `yaml:"selector" validate:"required"`

	// RemotePort is the port on the pod to proxy to.
	RemotePort int `yaml:"remotePort" validate:"min=1"`
}

// PortForward is a dev tool to forward the port on a pod to the local machine.
type PortForward struct {
	// Name is the name of the PortForward configuration.
	Name string `yaml:"name" validate:"required,name"`

	// Selector is a Kubernetes selector.  The first pod that matches the
	// selector is used for the port forwarding.  This is in accordance
	// with the behavior of `kubectl forward-port`.
	Selector string `yaml:"selector" validate:"required"`

	// LocalPort is the local port to forward to.
	LocalPort int `yaml:"localPort" validate:"min=1"`

	// RemotePort is the remote port to forward from.
	RemotePort int `yaml:"remotePort" validate:"min=1"`
}
