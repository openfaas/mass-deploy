## mass-deploy

Deploy functions to OpenFaaS en-masse

This tool exists to load test the OpenFaaS control-plane with a large number of functions.

Example usage, to deploy the `env` function 100 times

The default `--action` is `create`, so can be omitted.

```bash
go run . -image ghcr.io/openfaas/alpine:latest \
    -fprocess env \
    --workers 2 \
    --gateway http://127.0.0.1:8081 \
    --functions 100 \
    --start-at 0
```

If you're deploying 1000 functions and want to split that into two batches, set the `--start-at` flag to `0`, then `500` for the second batch.

Example usage to delete the functions created earlier:

```bash
go run . -image ghcr.io/openfaas/alpine:latest \
    -fprocess env \
    --workers 5 \
    --gateway http://127.0.0.1:8081 \
    --functions 100 \
    --start-at 0 \
    --action=delete
```

## Status

Internal testing tool for the OpenFaaS Ltd team and contributors

## License

MIT
