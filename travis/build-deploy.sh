#!/usr/bin/env bash

# Set image tag
TAG=$TRAVIS_BRANCH
if [ "$TRAVIS_PULL_REQUEST" -ne "false" ]
then
  TAG=$TRAVIS_PULL_REQUEST_BRANCH
fi

# Login into docker
docker login --username $DOCKER_USER --password $DOCKER_PASSWORD

architectures="arm arm64 amd64"
images=""
platforms=""

for arch in $architectures
do
# Build for all architectures and push manifest
  platforms="linux/$arch,$platforms"
done

platforms=${platforms::-1}


# Push multi-arch image
buildctl build --frontend dockerfile.v0 \
      --local dockerfile=. \
      --local context=. \
      --exporter image \
      --exporter-opt name=docker.io/zeerorg/cron-connector:$TAG \
      --exporter-opt push=true \
      --frontend-opt platform=$platforms \
      --frontend-opt filename=./Dockerfile.cross

# Push image for every arch with arch prefix in tag
for arch in $architectures
do
# Build for all architectures and push manifest
  buildctl build --frontend dockerfile.v0 \
      --local dockerfile=. \
      --local context=. \
      --exporter image \
      --exporter-opt name=docker.io/zeerorg/cron-connector:$TAG-$arch \
      --exporter-opt push=true \
      --frontend-opt platform=linux/$arch \
      --frontend-opt filename=./Dockerfile.cross &
done

wait

docker pull zeerorg/cron-connector:$TAG-arm
docker tag zeerorg/cron-connector:$TAG-arm zeerorg/cron-connector:$TAG-armhf
docker push zeerorg/cron-connector:$TAG-armhf
