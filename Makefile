# Image URL to use all building/pushing image targets
OPERATOR_IMAGE_TAG ?= controller:latest

include ./common.mk

all: run

run:  ##@dev Run main operator code
	@echo -e "${INFO} Running operator in dev mode"
	@go run ./cmd/main.go --dev --loglevel "DEBUG"

docker-build:  ##@build Build docker image for operator
	@echo -e "${INFO} Building docker for operator"
	docker build . -t ${OPERATOR_IMAGE_TAG}
