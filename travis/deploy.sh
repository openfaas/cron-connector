#!/bin/bash

docker run --rm --privileged multiarch/qemu-user-static:register --reset
docker login --username $DOCKER_USER --password $DOCKER_PASSWORD
docker build -t zeerorg/cron-connector:$TRAVIS_TAG .
docker push zeerorg/cron-connector:$TRAVIS_TAG
docker build -t zeerorg/cron-connector:$TRAVIS_TAG-arm -f Dockerfile.armhf .
docker push zeerorg/cron-connector:$TRAVIS_TAG-arm