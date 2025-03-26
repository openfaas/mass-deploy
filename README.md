# mass-deploy

Deploy functions to OpenFaaS en-masse

This tool is used to load test the OpenFaaS control-plane with a large number of functions.

It works with both OpenFaaS Standard/for Enterprises and with OpenFaaS Edge (faasd-pro).

## Initial setup

You should ideally set the pull policy for functions to "IfNotPresent", which will avoid downloading the same image from the registry for each function replica.

## Create 100 functions

This example uses the alpine image, which has the classic watchdog as its start-up process. Every request will fork a new process as given via `--fprocess` and then pipe stdio to it.

The `--workers` flag defaults to 1, but it can be increased to speed up the deployment. There will be a practical limit to how many workers should be used, so start low and increase it as required.

```bash
go run . create env \
    --image ghcr.io/openfaas/alpine:latest \
    --fprocess env \
    --workers 10 \
    --gateway http://127.0.0.1:8080 \
    --functions 100
```

The name `env` is an argument and will be used to template the name of each function, i.e. `env-1`, `env-2`, etc.

## Create an extra 100 functions

To create an extra 100 functions, set the `--start-at` flag to `100`.

```bash
go run . create env \
    --image ghcr.io/openfaas/alpine:latest \
    --fprocess env \
    --workers 10 \
    --gateway http://127.0.0.1:8080 \
    --functions 100 \
    --start-at 100
```

## Update all the functions

Imagine you had 10 functions, and wanted to update them all with a new label or annotation.

Here, we enable scale to zero after 3 minutes of idle time, and configure a cron expression to trigger the function every 10 minutes.

```bash
go run . create env \
    --image ghcr.io/openfaas/alpine:latest \
    --fprocess env \
    --gateway http://127.0.0.1:8080 \
    --functions 1 \
    --update-existing \
    --label com.openfaas.scale.zero=true \
    --label com.openfaas.scale.zero-duration=3m \
    --annotation topic=cron-function \
    --annotation schedule="*/10 * * * *" \
```

## Delete all the functions

Just like the creation, the deletion can be split into multiple batches using `--start-at`.

```bash
go run . delete env \
    --gateway http://127.0.0.1:8080 \
    --functions 100 \
    --start-at 0
```

## Viewing logs of the operator

Log tailing without stern:

```sh
kubectl logs -l app=gateway  -c operator -n openfaas   -f --prefix
```

Port forward OpenFaaS:

```bash
export OPENFAAS_URL=http://127.0.0.1:8080

kubectl port-forward -n openfaas svc/gateway 8081:8080 &

# If basic auth is enabled, you can now log into your gateway:
PASSWORD=$(kubectl get secret -n openfaas basic-auth -o jsonpath="{.data.basic-auth-password}" | base64 --decode; echo)
echo -n $PASSWORD | faas-cli login --username admin --password-stdin
```

## Viewing the logs for OpenFaaS Edge

```bash
sudo -E journalctl -u faasd-provider -f
```

## Status

Internal testing tool for the OpenFaaS Ltd team and contributors

## License

MIT
