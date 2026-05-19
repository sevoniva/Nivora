# Change Window

A Change Window defines whether delivery is allowed for an environment at a given time.

## Model

The Phase 3.3 model is intentionally small:

- environment id
- timezone
- start and end time
- days of week
- allowed flag
- metadata

If no change window is configured for an environment, the local foundation currently treats the environment as allowed and records that reason.

## Current Limitations

Phase 3.3 does not implement complex calendars, holiday schedules, freeze exceptions, or ITSM change records. Those require future RFCs.
