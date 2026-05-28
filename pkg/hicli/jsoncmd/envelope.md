## Envelope

All RPC messages (requests, responses and events) use the same envelope format:

* `command` (string): the command/event name.
* `request_id` (integer): incrementing (or decrementing) message ID. Responses
  will always contain the same ID. Requests without an ID will never be replied
  to. For events, the backend will start at -1 and go backwards.
* `data` (any): payload for the command/event, type depends on the command.

### Responses

For every request with `request_id` set, the backend replies exactly once with
the same `request_id` and one of:

* `command: "response"` with `data` containing the command-specific return value.
* `command: "error"` with `data` being the error message string.

### Cancellation

Requests can be canceled by sending a `cancel` request with the target
`request_id` inside `data`, plus an optional `reason` string. Cancellation is
best-effort: some operations may not stop immediately and there is no guarantee
of rollbacks.
