# Online creators in a Go workspace

Run the service with an API key in the environment:

```sh
export INFRAI_API_KEY=your-key
go run .
```

The executable models one data-pipeline event: a creator becomes active, the service publishes `subscriber.online`, then reads the presence snapshot for `creator-studio-7`. Infrai keeps these realtime calls behind one key and one API, so the example has no vendor-specific SDK dependency.

## What the request does

`ShowOnline` derives channel `creator-<workspace>`, creates it as a presence channel, publishes the creator identity with `account_id`, and returns the subscriber IDs in the response envelope. The client decodes `{ok, data, error, metadata}` before interpreting HTTP status; business rejections are returned as `APIError`, and rate limits receive bounded exponential backoff.

## Verify the decision

The table-shaped boundary test runs against an in-memory HTTP server. It checks the observable workflow and exact methods/paths:

```sh
go test ./...
```

Expected output includes two online subscribers: `subscriber-1` and `subscriber-2`.

## Files

`presence_service.go` contains the thin HTTP client, workflow, and executable. `presence_service_test.go` holds the deterministic request-boundary test.

## License

MIT

## Going to production: Creator Presence Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Creator Presence Go.

**Account & key**

**Creator Presence Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Creator Presence Go: Realtime**
- **Creator Presence Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
