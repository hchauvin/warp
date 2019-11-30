# warp: Yet another deployment wrapper 🚀

[![CircleCI](https://circleci.com/gh/hchauvin/warp.svg?style=svg)](https://circleci.com/gh/hchauvin/warp) [![GoDoc](https://godoc.org/github.com/hchauvin/warp?status.svg)](https://godoc.org/github.com/hchauvin/warp) [![Coverage Status](https://coveralls.io/repos/github/hchauvin/warp/badge.svg?branch=master&t=2W0xju)](https://coveralls.io/github/hchauvin/warp?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/hchauvin/warp)](https://goreportcard.com/report/github.com/hchauvin/warp) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`warp` is yet another command-line interface for deploying Kubernetes stacks.  The approach is very similar to
[Skaffold](https://github.com/GoogleContainerTools/skaffold), but `warp` is specifically
tailored to end-to-end testing.  It is a tiny wrapper (warper?) to develop and perform end-to-end tests,
at warp seed. 🚀

`warp` is useful when you want to manage more than one deployment of a stack.  This arises when
you have multiple software engineers working on the same project, or an undetermined number of
parallel jobs running in Continuous Integration.  `warp` uses
[name_manager](https://github.com/hchauvin/name_manager) to minimize the number of stacks that
need to be deployed.

Stacks are deployed and testing according to _pipelines._  These pipelines give all the
steps to deploy the stack, all the tools to launch during local development (file syncing,
port forwarding, webapp live reload, ...), all the end-to-end tests to run.

The end-to-end tests and local development tools are automatically passed environment
variables to connect to the stack.  Performing end-to-end tests on a test stack
is greatly simplified.  For instance, this would be the pipeline to perform Cypress browser
tests on a web app:

```yaml
# stack identifies the stacks managed by this pipeline.
# Stacks either have a fixed name (typical for production stacks),
# or a name built from a fixed component (the "family" name) and a variable 
# component (the "short" name).  The short name is automatically generated.
stack:
  family: my-web-app
deploy:
  # kustomize instructs the pipeline to deploy the stack using Kustomize.
  kustomize:
    # path is the path to the Kustomize configuration.
    # "kustomize/base/kustomization.yml" would be a file referencing all
    # the Kubernetes resources to deploy and the kustomizations to apply.
    path: kustomize/base
# commands lists all the commands that can be executed against the stack.
commands:
- name: test
  description: Non-interactive testing using Cypress.
  env:
    # If the frontend is implemented as a Kubernetes service named
    # frontend and its pods serve the frontend on port 8080, the
    # template would expand to, e.g., "127.0.0.1:57689", where
    # 57689 is a random port.  In the background, warp forwards
    # the 8080 port of a pod to this local random port.
    # The port forwarding ends when the test completes.
    - 'CYPRESS_baseUrl=http://{{ serviceAddress "frontend" 8080 }}'
  workingDir: frontend
  command: ['yarn', 'cypress', 'run']
- name: open
  descripton: Interactive testing using Cypress.
  env:
    - 'CYPRESS_baseUrl=http://{{ serviceAddress "frontend" 8080 }}'
    workingDir: frontend
    command: ['yarn', 'cypress', 'open']
```

To run all the tests, if the above pipeline is in `pipeline.yml`,
use `warp pipeline.yml --run test --tail` (`--tail` follows the stdout/stderr
of all the containers in the stack).  To develop or debug tests,
run `warp pipeline.yml --run open --tail`.

It doesn't get simpler than that. 🏖

Because pipelines can inherit the steps of other, _base_ pipelines, it is possible to
have an overlay organization that would mimic how project are organized with Kustomize.
For instance, you could have a `base` folder container a base `pipeline.yml` and `kustomization.yml`,
a `dev` folder containing configurations for a "dev" stack with file synchronization and the ability
to manage multiple stacks, and a `prod` folder containing the configuration for the production stack.

`warp` can be either consumed as a Go package or through a standalone Command-Line Interface (CLI).
It is continuously tested on Linux, Mac OSX, and Windows.

## License

`warp` is licensed under [The MIT License](./LICENSE).