IMG_NAME?=cron-connector

TAG?=dev
PLATFORMS?=linux/amd64,linux/arm/v7,linux/arm64
OWNER?=alexellis2
SERVER?=docker.io

VERSION := $(shell git describe --tags --dirty)
GIT_COMMIT := $(shell git rev-parse HEAD)

export DOCKER_CLI_EXPERIMENTAL=enabled
export DOCKER_BUILDKIT=1

.PHONY: publish-buildx-all
publish-buildx-all:
	@echo  $(SERVER)/$(OWNER)/$(IMG_NAME):$(TAG) && \
	docker buildx create --use --name=multiarch --node=multiarch && \
	docker buildx build \
		--platform $(PLATFORMS) \
		--push=true \
        --build-arg GIT_COMMIT=$(GIT_COMMIT) \
        --build-arg VERSION=$(VERSION) \
		--tag $(SERVER)/$(OWNER)/$(IMG_NAME):$(TAG) \
		.
