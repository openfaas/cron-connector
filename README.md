# A Cron Connector for OpenFaaS

This is a cron event connector for OpenFaaS. This was built to provide a timer interface to trigger OpenFaaS functions. Also checkout [OpenFaaS docs on cron](https://docs.openfaas.com/reference/cron/) for other methods on how you can run functions triggered by cron.

This project was forked from [zeerorg/cron-connector](https://github.com/openfaas/cron-connector) to enable prompt updates and patches for end-users.

## How to Use

You need to have OpenFaaS deployed first, see [https://docs.openfaas.com](https://docs.openfaas.com) to get started

### Deploy using the Helm chart

The helm chart is available in the [faas-netes](https://github.com/openfaas/faas-netes/tree/master/chart/cron-connector) repo.

For faasd, you can edit your `docker-compose.yaml` file to see the deployment. See the chart above for the image name and configuration required.

### Add to faasd

Edit `/var/lib/faasd/docker-compose.yaml` and add:

```yaml
  cron-connector:
    image: "ghcr.io/openfaas/cron-connector:latest"
    environment:
      - gateway_url=http://gateway:8080
      - basic_auth=true
      - secret_mount_path=/run/secrets
    volumes:
      # we assume cwd == /var/lib/faasd
      - type: bind
        source: ./secrets/basic-auth-password
        target: /run/secrets/basic-auth-password
      - type: bind
        source: ./secrets/basic-auth-user
        target: /run/secrets/basic-auth-user
    cap_add:
      - CAP_NET_RAW
    depends_on:
      - gateway
```

Then restart faasd.
### Trigger a function from Cron

The function should have 2 annotations:

1. `topic` annotation should be `cron-function`.
2. `schedule` annotation should be the cron schedule on which to invoke function
3. `async` annotation defines async invocation (`true` or `false`) - default `false`

For example, we may have a function "nodeinfo" which we want to invoke every 5 minutes:

Deploy via the CLI:

```
faas-cli store deploy nodeinfo --annotation schedule="*/5 * * * *" --annotation topic=cron-function -annotation async="true"
```

Or via `stack.yml`:

```yaml
functions:
  nodeinfo:
    image: functions/nodeinfo
    annotations:
      topic: cron-function
      schedule: "*/5 * * * *"
      async: "true"
```

You can learn how to create and test the [Cron syntax here](https://crontab.guru/every-5-minutes).

