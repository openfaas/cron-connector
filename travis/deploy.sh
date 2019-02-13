#!/bin/bash

#docker login --username $DOCKER_USER --password $DOCKER_PASSWORD
docker build -t zeerorg/cron-connector .
#docker push zeerorg/cron-connector:$TRAVIS_TAG
sudo docker run --privileged linuxkit/binfmt:v0.6
docker build -t zeerorg/cron-connector:arm -f Dockerfile.armhf .
#docker push zeerorg/cron-connector:$TRAVIS_TAG-arm