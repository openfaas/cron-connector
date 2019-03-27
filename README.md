# A Cron Connector for OpenFaaS

This is a cron event connector for OpenFaaS. This was built to provide a timer interface to trigger OpenFaaS functions. Also checkout OpenFaas [docs on cron](https://docs.openfaas.com/reference/cron/) for other methods on how you can run cron connector.

## How to Use

You need to have OpenFaaS deployed first, see [https://docs.openfaas.com](https://docs.openfaas.com) to get started

Works with both armd64 and armhf (Raspberry Pi).

1. For Docker Swarm: 
```
curl -s https://raw.githubusercontent.com/zeerorg/cron-connector/master/yaml/docker-compose.yml | docker stack deploy func -c -
```

2. For Kubernetes:
```
curl -s https://raw.githubusercontent.com/zeerorg/cron-connector/master/yaml/kubernetes/connector-dep.yml | kubectl create --namespace openfaas -f -
```

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
