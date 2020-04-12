# featured.io

![CI](https://github.com/JTarball/featured.io/workflows/CI/badge.svg?branch=master)


A distributed feature flag tool for Kubernetes


## Status

*active development

### Task list
- [ ] Implement a Kubernetes operator
  - [x] Generate basic CRD / API
    - [x] Import auto-generated code script
  - [x] Implement a basic controller
    - [x] Implement a control interface for configmaps
      - [x] Add some basic tests
  - [ ] Implement basic docker build for operator
    - [x] Write dockerfile for operator
    - [ ] Automate docker build/push process
- [ ] Implement requirements for kubernetes operator
- [ ] Helm chart to deploy operator
- [ ] Produce documentation on godoc website

