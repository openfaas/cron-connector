#!/usr/bin/env bash

# Login into docker
docker login --username $DOCKER_USER --password $DOCKER_PASSWORD

# Build for amd64 and push
buildctl build --frontend dockerfile.v0 \
            --local dockerfile=. \
            --local context=. \
            --exporter image \
            --exporter-opt name=docker.io/zeerorg/cron-connector:test-build-amd64 \
            --exporter-opt push=true \
            --frontend-opt platform=linux/amd64 \
            --frontend-opt filename=./Dockerfile


# Build for armhf and push
buildctl build --frontend dockerfile.v0 \
            --local dockerfile=. \
            --local context=. \
            --exporter image \
            --exporter-opt name=docker.io/zeerorg/cron-connector:test-build-armhf \
            --exporter-opt push=true \
            --frontend-opt platform=linux/armhf \
            --frontend-opt filename=./Dockerfile.armhf


export DOCKER_CLI_EXPERIMENTAL=enabled

# Create manifest list and push that
docker manifest create zeerorg/cron-connector:test-build \
            zeerorg/cron-connector:test-build-amd64 \
            zeerorg/cron-connector:test-build-armhf

docker manifest annotate zeerorg/cron-connector:test-build zeerorg/cron-connector:test-build-armhf --arch arm
docker manifest annotate zeerorg/cron-connector:test-build zeerorg/cron-connector:test-build-amd64 --arch amd64

docker manifest push zeerorg/cron-connector:test-build
