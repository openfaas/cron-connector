# Invoke your functions on a cron schedule

This is a cron event connector for OpenFaaS. This was built to provide a timer interface to trigger OpenFaaS functions. Also checkout [OpenFaaS docs on cron](https://docs.openfaas.com/reference/cron/) for other methods on how you can run functions triggered by cron.

This project was forked from [zeerorg/cron-connector](https://github.com/openfaas/cron-connector) to enable prompt updates and patches for end-users.

## How to Use

First, [deploy OpenFaaS with faasd or Kubernetes](https://docs.openfaas.com/deployment/)

### Deploy the connector for Kubernetes

For Kubernetes, see: [Scheduling function runs](https://docs.openfaas.com/reference/cron/)

### Deploy the connector for faasd

For faasd, see [Serverless For Everyone Else](https://gumroad.com/l/serverless-for-everyone-else).

### Trigger a function from Cron

The function should have 2 annotations:

1. `topic` annotation should be `cron-function`.
2. `schedule` annotation should be the cron schedule on which to invoke function

For example, we may have a function "nodeinfo" which we want to invoke every 5 minutes:

Deploy via the CLI:

```bash
faas-cli store deploy nodeinfo \
  --annotation schedule="*/5 * * * *" \
  --annotation topic=cron-function
```

Or via `stack.yml`:

```yaml
functions:
  nodeinfo:
    image: functions/nodeinfo
    annotations:
      topic: cron-function
      schedule: "*/5 * * * *"
```

You can learn how to create and test the [Cron syntax here](https://crontab.guru/every-5-minutes).

