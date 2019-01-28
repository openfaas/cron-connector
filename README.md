# A Cron Connector for OpenFaaS

This is a cron event connector for OpenFaaS. This was built to provide a timer interface to trigger OpenFaaS functions.

## How to Use

You need to have OpenFaaS deployed first, see [https://docs.openfaas.com](https://docs.openfaas.com) to get started

1. Clone this repository:

```
git clone https://github.com/zeerorg/cron-connector
cd cron-connector
```

2. For Docker Swarm: `docker stack deploy func -c ./yaml/docker-compose.yml`
3. For Docker Swarm on Raspberry Pi: `docker stack deploy func -c ./yaml/docker-compose.armhf.yml`
4. For Kubernetes: `kubectl create -f ./yaml/kubernetes --namespace openfaas`
5. For Kubernetes on Raspberry Pi: `kubectl create -f ./yaml/kubernetes-armhf --namespace openfaas`

## Adding function

The function should have 2 annotations:

1. `topic` annotation should be `cron-function`.
2. `schedule` annotation should be the cron schedule on which to invoke function

For example, we may have a function "nodeinfo" which we want to invoke every 5 minutes:

```yaml
functions:
  nodeinfo:
    image: functions/nodeinfo
    annotations:
      topic: cron-function
      schedule: "*/5 * * * *"
```

You can learn how to create and test the [Cron syntax here](https://crontab.guru/every-5-minutes).

See the full example here: [sample/stack.yml](sample/stack.yml)
