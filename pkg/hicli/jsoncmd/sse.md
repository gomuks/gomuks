## Server-sent events

As an alternative to websockets, clients can use server-sent events for
receiving data and the `exec` endpoint for sending.

The SSE endpoint is `_gomuks/sse`. It takes the same query parameters for
session resumption and catchup syncs as the websocket. However, unlike
websockets, compression is enabled automatically based on the `Accept-Encoding`
header rather than a query param. The backend supports `zstd` and `deflate`.

By default, the endpoint returns standard server-sent events with the
`text/event-stream` mime type. Optionally, clients can set the `Accept` header
to `application/jsonl` to receive raw JSON lines instead. When using the raw
encoding, the server will occasionally send a plain `null` to ensure the socket
stays alive.

Event acknowledgement is done by sending a POST request to `_gomuks/sse/ping`
with `run_id`, `listener_id` and `last_received_event` as query parameters.
However, there's no connection killing, so the interval for acks can be longer
than with websockets. 1 minute is recommended.
