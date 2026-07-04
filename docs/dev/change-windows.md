# Change Windows

Phase 3.3 adds a simple change-window gate for release and deployment governance.

## Manage And Evaluate

```sh
nivora change-window create --file examples/change-windows/prod-window.yaml
nivora change-window create \
  --name prod-business-hours \
  --env prod \
  --timezone UTC \
  --start 09:00 \
  --end 17:00 \
  --day monday \
  --day tuesday \
  --allowed=true
nivora change-window list
nivora change-window get <change-window-id>
nivora change-window evaluate --env prod
nivora change-window evaluate --env prod --at 2026-05-18T02:00:00Z
```

Equivalent API:

```sh
curl -s http://localhost:8080/api/v1/change-windows
curl -s http://localhost:8080/api/v1/change-windows \
  -H 'content-type: application/json' \
  -d '{"name":"prod-business-hours","environmentId":"prod","timezone":"UTC","startTime":"09:00","endTime":"17:00","daysOfWeek":["monday"],"allowed":true}'
curl -s http://localhost:8080/api/v1/change-windows/<id>
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
