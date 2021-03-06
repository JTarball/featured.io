# Image URL to use all building/pushing image targets
OPERATOR_IMAGE_TAG ?= controller:latest

SHELL := /bin/bash

include ./common.mk

all: run

run:  ##@dev Run main operator code
	@echo -e "${INFO} Running operator in dev mode"
	@go run ./cmd/main.go --dev --loglevel "DEBUG"

## ---

docker-build:  ##@build Build docker image for operator
	@echo -e "${INFO} Building docker for operator"
	docker build . -t ${OPERATOR_IMAGE_TAG}

kind:  ##@build create kind cluster
	kind create cluster

kind-clean: ##@build delete kind cluster
	@echo -e "${INFO} --- Cleaning kinD cluster ----"
	kind delete cluster
## ----

test-kind: test-kind-testing test-kind-helm test-kind-infrastructure-master test-kind-infrastructure-local  ##@test Run all integration tests with kinD (https://kind.sigs.k8s.io/)
	@echo -e "${INFO} Ran all kind integration tests"

test-kind-helm:  ##@test Run helm kinD tests
	@echo -e "${INFO} --- Run helm tests ----"
	@cd acceptance_test && go test -json ./helm/... -tags=kind | ./test-summary -verbose

test-kind-testing:  ##@test Run testing library kinD tests
	@echo -e "${INFO} --- Run testing library ----"
	@cd acceptance_test && go test -json ./testing/... -tags=kind | ./test-summary -verbose

test-kind-infrastructure-master:  ##@test Run infratructure kinD tests based off master
	@echo -e "${INFO} --- Run testing infrastructure master ----"
	@cd acceptance_test && go test -json ./featured/infrastructure/master/... -tags=kind | ./test-summary -verbose

test-kind-infrastructure-local:  ##@test Run infratructure kinD tests tests based off local code
	@echo -e "${INFO} --- Run testing infrastructure local ----"
	@cd acceptance_test && go test -json ./featured/infrastructure/local/... -tags=kind | ./test-summary -verbose


## ----

fmt:   ##@ci Run gofmt against code
	gofmt -l .

vet:   ##@ci Run go vet against code
	go vet ./...

## ----

clean: kind-clean
	@echo -e "${INFO} --- Cleaning ----"