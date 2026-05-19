# Change Windows

Phase 3.3 adds a simple change-window gate for release and deployment governance.

## Evaluate

```sh
nivora change-window evaluate --env prod
```

Equivalent API:

```sh
curl -s http://localhost:8080/api/v1/change-windows/evaluate \
  -H 'content-type: application/json' \
  -d '{"environmentId":"prod"}'
```

## Local Examples

See:

- `examples/change-windows/prod-window.yaml`

## Limits

The model is intentionally small. It does not handle enterprise calendars, holidays, freeze exceptions, or ITSM systems yet.
