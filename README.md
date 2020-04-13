# featured.io

![CI](https://github.com/JTarball/featured.io/workflows/CI/badge.svg?branch=master)


A distributed feature flag tool for Kubernetes

> Requires kubernetes > v1.17

> Requires helm 3

## Status

### Active Development

### Task list
- [ ] Implement a Kubernetes operator
  - [x] Generate basic CRD / API
    - [x] Import auto-generated code script
  - [x] Implement a basic controller
    - [x] Implement a control interface for configmaps
      - [x] Add some basic tests
  - [ ] Implement basic docker build for operator
    - [x] Write dockerfile for operator
    - [x] Automate docker build/push process
- [ ] Implement requirements for kubernetes operator
- [ ] Helm chart to deploy operator
  - [ ] Integration / Acceptance Tests
    - [x] Add defense against Helm 2 
    - [x] Add helm template test
    - [x] Add helm linting test
    - [ ] Add helm install test
    - [ ] Add helm chart upgrade / rollback test
    - [x] Add support for running simple k8s tests on `kind`
  - [ ] Automate versioning
    - [ ] Add helm template test to check version is set with the correct hash
- [ ] Implement proper CIs (Cloud Integration / Continuous Integration)
  - [x] Improve `cost` (don't build pipeline for changes to .md etc)
- [ ] Produce documentation on godoc website

