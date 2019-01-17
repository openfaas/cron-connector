# OpenFaas Cron Connector

This is a cron engine for OpenFaas. OpenFaas doesn't come with a timer tigger, hence this was built to provide timer interface with OpenFaas.

## How to Use

1. Clone this repository: `git clone https://github.com/zeerorg/cron-connector && cd cron-connector`
2. For OpenFaas deployed on Docker Swarm do: `docker stack deploy func -c ./docker-compose.yml`
3. For OpenFaas on RaspberryPi with Docker Swarm do: `docker stack deploy func -c ./docker-compose.armhf.yml`
4. For OpenFaas deployed on kubernetes do: `kubectl create -f ./kubernetes --namespace openfaas`
5. For OpenFaas on RaspberryPi kubernetes do: `kubectl create -f ./kubernetes-armhf --namespace openfaas`

## Adding function

The function should have 2 annotations:

1. `topic` annotation should be `cron-function`.
2. `schedule` annotation should be the cron schedule on which to invoke function.

Checkout a sample function yaml file at [sample-func/stack.yml](sample-func/stack.yml)
