# Event Model

Nivora events use a CloudEvents-style envelope with `specversion`, `id`, `type`, `source`, `subject`, `time`, `datacontenttype`, and `data`.

Phase 0 defines event topics in AsyncAPI and includes an in-memory event bus for local boundaries. Durable event transport is future work.

