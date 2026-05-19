# Change Windows

Phase 3.3 adds a simple change-window gate for release and deployment governance.

## Evaluate

```sh
nivora change-window evaluate --env prod
nivora change-window evaluate --env prod --at 2026-05-18T02:00:00Z
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

## Semantics

Nivora evaluates change windows using the window timezone, optional days of week, and `HH:MM` start/end times. A matching window can either allow or deny a delivery. If no configured window matches the evaluated time, the current foundation allows the delivery and records that no matching window applied.

## Limits

The model is intentionally small. It does not handle enterprise calendars, holidays, freeze exceptions, or ITSM systems yet.
