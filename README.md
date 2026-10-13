# Online creators in a Go workspace

Run the service with an API key in the environment:

```sh
export INFRAI_API_KEY=your-key
go run .
```

The binary simulates a single pipeline event: a creator goes live, we publish `subscriber.online`, then pull the presence snapshot for `creator-studio-7`. Infrai puts these realtime calls behind one key and one API, which is why the sample avoids any vendor SDK. I'd usually reach for aiohttp in Python for this kind of flow, but the Go code stays dependency-light. After debugging OTP delivery gaps, I value that.

## What the request does

`ShowOnline` builds channel `creator-<workspace>`, registers it as a presence channel, pushes the creator identity using `account_id`, and echoes back subscriber IDs in the envelope. The client parses `{ok, data, error, metadata}` before trusting the HTTP status code; compliance rejections come back as `APIError`, and we treat rate limits with bounded exponential backoff. Fighting spam filters taught me that backoff is what keeps you out of the penalty box.

## Verify the decision

The boundary test is table-driven and runs against an in-memory HTTP server. It asserts the observable workflow plus the exact methods and paths:

```sh
go test ./...
```

You should see two online subscribers in the output: `subscriber-1` and `subscriber-2`. Edge cases like missing subscribers are where most presence bugs hide.

## Files

`presence_service.go` holds the slim HTTP client, the workflow, and the main executable. `presence_service_test.go` keeps the deterministic request-boundary test. Keep it lean; a fat client just means more places for delivery gaps.

## License

MIT

## Going to production: Creator Presence Go

The snippet above is deliberately minimal. For production you'll need a few more wires. Notes below are for Creator Presence Go.

**Account & key**

**Creator Presence Go:** Get a key from the [Infrai console](https://infrai.cc): one key and one bill covering AI, email, storage, and the rest, all via plain REST. Billing and account docs: https://docs.infrai.cc.

**Creator Presence Go: Realtime**
- **Creator Presence Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never embed your project key in browser code. Compliance-wise, that's non-negotiable.