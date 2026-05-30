## Websocket

The main method of connecting is using the websocket. The websocket is at
`_gomuks/websocket` and requires standard cookie authentication.

However, the RPC API is intentionally just JSON, so it can be sent over other
transports as well. In particular, when bundling the backend with the frontend,
it is recommended to use a direct in-process channel rather than the websocket.
If writing a frontend in a language other than Go, the [C FFI bindings](https://github.com/gomuks/gomuks/tree/main/pkg/ffi)
may be useful.

### Compression

If the `compress` query param is set to `1`, the websocket will use a
connection-wide deflate compression similar to how Discord's gateway works.
Only messages sent by the backend are compressed; client messages are not.

Enabling compression will also enable batched messages in a single frame, i.e.
multiple JSON objects may be concatenated together using a newline (`\n`) as the
separator if the backend detects that the connection isn't keeping up. There is
currently no option to turn off batching when compression is enabled.

Compression is recommended for lower bandwidth networks like mobile data.
It can save up to 70% of bandwidth. The primary downside is that standard
developer tools will no longer understand the websocket messages.

### Event buffering

The backend will buffer events that the client hasn't acknowledged yet to allow
faster resuming in case the websocket is interrupted. To use session resumption,
the client has to handle the `run_id` command and store the `run_id` field
inside the data. When reconnecting, the client should include the run ID as well
as the most recent negative request ID it received in the `run_id` and
`last_received_event` query params respectively.

The event buffer is entirely in-memory, which means resumption will fail if the
backend has been restarted. For non-resumed inits, the first `sync_complete`
event will have the `clear_state` flag set to true. For successful resumes, the
client will only get missed events rather than the full initial sync.

The `init_complete` event is always sent once the client is caught up regardless
of whether the resume succeeded or not.

### Keepalive pings and event acknowledgement

Clients MUST send periodic pings to keep the connection alive. The backend will
kill connections that don't send any data for over 60 seconds. Clients SHOULD
implement similar timeouts and reconnect if they don't receive any data from the
backend. The recommended ping interval is 15 seconds.

Unlike normal requests, pings are exclusive to the websocket layer and will
result in `"command": "pong"` instead of `"command": "response"`.

In addition to keeping the connection open, pings are used to acknowledge
received events so that the backend can remove them from its in-memory cache.
Clients should include the most recent negative request ID they received in the
`last_received_id` field inside the `data` of the ping.

#### Example

Request:

```json
{
  "command": "ping",
  "request_id": 123,
  "data": {
    "last_received_id": -456
  }
}
```

Response:

```json
{
  "command": "pong",
  "request_id": 123
}
```
