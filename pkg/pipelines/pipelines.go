// SPDX-License-Identifier: MIT
// Copyright (c) 2019 Hadrien Chauvin

// Package pipelines defines the pipeline API.
package pipelines

import (
	"fmt"
	"strings"
)

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

	Lint Lint `yaml:"lint"`

	// Deploy describes the deployment steps.
	Deploy Deploy `yaml:"deploy"`

	Setups Setups `yaml:"setups,omitempty" patchStrategy:"merge" patchMergeKey:"name" validate:"dive"`

	// Command describes the command configurations.  Command configurations
	// are used to spawn processes that can access the stack, with
	// port forwarding and other mechanisms, between the setup and
	// the tear down of the stack.
	Commands []Command `yaml:"commands,omitempty" patchStrategy:"merge" patchMergeKey:"name" validate:"dive"`

	// Path is the absolute path to the pipeline definition.  It is
	// resolved by Read.
	Path string `yaml:"-"`
}

// Setups is a slice of named setups.  Within this slice, names
// are unique.
type Setups []Setup

// Get gets the setup given its name.
func (setups Setups) Get(name string) (*Setup, error) {
	for _, setup := range setups {
		if setup.Name == name {
			return &setup, nil
		}
	}
	return nil, fmt.Errorf(
		"cannot find setup named '%s'; available setups: %s",
		name,
		strings.Join(setups.Names(), " "))
}

// Get gets the setup given its name.  Panics if the setup cannot
// be found.
func (setups Setups) MustGet(name string) *Setup {
	s, err := setups.Get(name)
	if err != nil {
		panic(err.Error())
	}
	return s
}

// Names gets a slice of all the setup names.
func (setups Setups) Names() []string {
	names := make([]string, len(setups))
	for i, setup := range setups {
		names[i] = setup.Name
	}
	return names
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

// Lint describes the linting steps
type Lint struct {
	// DisableHelmKubeScore disables applying kube-score on the
	// Kubernetes resources created by Helm.
	DisableHelmKubeScore bool `yaml:"disableHelmKubeScore"`

	// DisableKustomizeKubeScore disables applying kube-score on
	// the Kubernetes resources created by Kustomize.
	DisableKustomizeKubeScore bool `yaml:"disableKustomizeKubeScore"`
}

// Deploy describes the deployment steps.
type Deploy struct {
	// Container describes the deployment steps relative to
	// containerization.
	Container *Container `yaml:"container,omitempty"`

	// Helm describes the Helm chart to use to deploy to
	// a Kubernetes cluster.  If it is omitted,
	// the stack is not deployed to Kubernetes with helm.
	//
	// The Helm step always happens before the Kustomize step.
	Helm *Helm `yaml:"helm,omitempty"`

	// Kustomize describes the Kustomize config to use to
	// deploy to a Kubernetes cluster.  If it is omitted,
	// the stack is not deployed to Kubernetes with kustomize.
	Kustomize *Kustomize `yaml:"kustomize,omitempty"`

	// Terraform describes the Terraform config to use to
	// apply a Terraform config.  If it is omitted,
	// the stack is not deployed with Terraform.
	Terraform *Terraform `yaml:"terraform,omitempty"`
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

// ContainerManifestEntry is an entry in the container manager.
type ContainerManifestEntry struct {
	// Ref is a reference to an actual image in a container registry.
	Ref string `json:"ref"`
}

// Helm describe the Help config to use to deploy to a Kubernetes cluster.
type Helm struct {
	// Path to the helm chart, relative to the root given in the
	// warprc.toml configuration.
	Path string `yaml:"path" validate:"required"`

	// Args are additional arguments to pass to the Helm CLI.
	Args []string `yaml:"args"`

	// Selector is a label to use with `kubectl apply` for resource pruning.
	// If not specified, it defaults to `warp.stack=<stack name>`, where
	// `<stack name>` is the name of the stack.
	LabelSelector string `yaml:"labelSelector"`
}

// Kustomize describes the Kustomize config to use to
// deploy to a Kubernetes cluster.
type Kustomize struct {
	// Path to the kustomization file, relative to the root given
	// in the warprc.toml configuration.
	Path string `yaml:"path" validate:"required"`

	// DisableNamePrefix Disables prefixing the names of Kubernetes
	// resources with the names of the Stack.
	DisableNamePrefix bool `yaml:"disableNamePrefix"`

	// PatchesStrategicMerge contains additional patches to apply
	// with a strategic merge.  The patches are subject to template
	// expansion.
	PatchesStrategicMerge []string `yaml:"patchesStrategicMerge,omitempty" patchStrategy:"append"`

	SecretGenerator []KustomizeSecretGenerator `yaml:"secretGenerator"`
}

type KustomizeSecretGenerator struct {
	Name string `yaml:"name"`
	Literals []string `yaml:"literals"`
}

// Terraform describes the Terraform config to use to
// deploy with Terraform.
type Terraform struct {
	// Path to the directory containing the Terraform config.
	Path string

	// Var is a map of terraform variables to values.  The values
	// are expanded using gotemplates.
	Var map[string]string
}

// Setup defines the resources to set up and tear down around the
// execution of test or batch commands.
type Setup struct {
	// Name is the name of the environment.
	Name string `yaml:"name" validate:"required,name"`

	// Bases contains base files to merge into this pipeline, using
	// strategic merging.  The file names must be relative to the
	// root given in the warprc.toml configuration.
	//
	// Loops in the inheritance chain are forbidden and explicitly
	// controlled for.
	Bases []string `yaml:"bases,omitempty"`

	// Before is a list of command hooks that are executed concurrently
	// before the command itself is actually executed.  If any of the
	// hook fails, the other hooks and the command itself are skipped.
	// The execution is eventually reported as a failure.
	Before []CommandHook `yaml:"before,omitempty" validate:"dive"`

	// Env is a list of environment variables, specified as "name=value"
	// strings.  The values can be templated.  The template functions
	// allow, e.g., to request service addresses, configuration values,
	// that come from the deployment stage.
	Env []string `yaml:"env"`

	// Dev describes the dev tools.  They are enabled when running
	// a pipeline in dev mode.  They can be used interactively,
	// during local development or debugging.
	Dev Dev `yaml:"dev"`
}

// BaseCommand gives instructions to set up the environment and invoke
// a command.
type BaseCommand struct {
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

// Command describes a command configuration.  Command configurations
// are used to spawn processes that can access the stack, with
// port forwarding and other mechanisms, between the setup and
// the tear down of the stack.
type Command struct {
	BaseCommand `yaml:",inline"`

	// Name is the name of the run configuration.
	Name string `yaml:"name" validate:"required,name"`

	// Description describes the run configuration.
	Description string `yaml:"description,omitempty"`

	// Tags are arbitrary tags associated with the run configuration.
	// They can be used to organize run configurations.
	Tags []string `yaml:"tags" validate:"name"`

	Setup string `yaml:"setup"`
}

// CommandHook is a command used as a set-up/tear-down hook.
type CommandHook struct {
	// Name is an optional name to help identify the hook.
	Name string `yaml:"name,omitempty" validate:"name"`

	// DependsOn lists the name of other command hooks that must be executed
	// successfully before this command hook is executed.  Loops
	// are of course forbidden.
	DependsOn []string `yaml:"dependsOn"`

	// WaitFor indicates that the hook must wait for the resources
	// to be in a certain state.
	WaitFor *WaitForHook `yaml:"waitFor,omitempty"`

	// Run indicates that the hook must execute a command.
	Run *BaseCommand `yaml:"run,omitempty"`

	// HttpGet indicates that the hook must perform an HTTP Get.
	HTTPGet *HTTPGet `yaml:"httpGet,omitempty"`

	// Timeout in seconds after which the hook is considered to have failed.
	// 0 indicates no timeout.
	TimeoutSeconds int `yaml:"timeoutSeconds,omitempty"`
}

// WaitForHook is a hook that waits for external resources to be available,
// running or ready.
type WaitForHook struct {
	// Resources lists the resource kinds to wait for.
	Resources []WaitForResourceKind `yaml:"resources"`
}

// WaitForResourceKind defines what kind of waiting should be performed.
type WaitForResourceKind string

const (
	// Endpoints waits for all the services in the stack to have at least one
	// endpoint ready.
	Endpoints = WaitForResourceKind("endpoints")

	// Pods waits for all the pods in the stack to be ready.
	Pods = WaitForResourceKind("pods")

	// OnePodPerService waits for at least one pod up and running per service.
	OnePodPerService = WaitForResourceKind("onePodPerService")
)

// WaitForResourceKinds contains all the valid resource kinds to wait for.
var WaitForResourceKinds = []WaitForResourceKind{
	Endpoints,
	Pods,
	OnePodPerService,
}

// HTTPGet is a hook that waits for a URL to returns a 2xx status.
type HTTPGet struct {
	// URL to send the GET request to.  The URL is subject to
	// template substitution.
	URL string `yaml:"url" validate:"required"`

	// HTTPHeaders is a list of custom HTTP headers to set in
	// the request.  HTTP allows repeated headers.
	HTTPHeaders []HTTPHeader `yaml:"httpHeaders"`
}

// HTTPHeader specifies the name and value of an HTTP header.
type HTTPHeader struct {
	// Name of the HTTP header
	Name string `yaml:"name" validate:"required"`

	// Value of the HTTP header.
	Value string `yaml:"value"`
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

	// DeploymentName is the name of the deployment used to create the pods.
	DeploymentName string `yaml:"deploymentName" validate:"required"`

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
