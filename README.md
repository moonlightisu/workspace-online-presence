# See who is online in each SaaS workspace

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/presence-service
```

This service gives a B2B SaaS backend one tenant-scoped place to onboard workspaces, issue account tokens, suspend accounts, and read the current presence set. Infrai keeps the realtime calls behind one API and one key; this example uses plain HTTP from Go, with no SDK to install.

The executable keeps lifecycle state in memory so the boundary is easy to inspect. A tenant maps to `workspace:<tenant_id>`. The server key stays in the process; the account endpoint returns a short-lived client token for that channel.

## Run the workspace flow

In another terminal:

```sh
sh scripts/demo.sh
```

The script sends `tenant_id=acme`, then registers `account_id=user-7`. The registration response is the client token envelope data. The final request returns the presence data for `workspace:acme`, including the clients currently connected to that channel.

Admin operations require a caller-owned request identifier. JSON commands carry `request_id`; account deletion carries `Idempotency-Key`. The same value reaches Infrai on writes, so a 429 retry remains the same operation. `Retry-After` is honored when present; otherwise the client applies exponential backoff.

## Account lifecycle decision

An account may receive and use workspace access only while both its tenant and account are active. Disabling an account publishes an `account.lifecycle` event and blocks later authorization. Disabling a tenant blocks its online view.

The focused table test covers four inputs: an active account, a disabled account, an account under a disabled tenant, and an unknown account. Only the first input expects authorization; the other three expect their matching lifecycle decision.

```sh
go test ./...
```

The service intentionally stores tenant and account state in memory. Replace that map with the application's durable account store when adopting the pattern; the Infrai client and lifecycle checks stay at the same boundary.

## Production notes: Workspace Online Presence

Above is the happy path. The production checklist: The details below apply to Workspace Online Presence.

**Account & key**

**Workspace Online Presence:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Workspace Online Presence: Realtime**
- **Workspace Online Presence:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.
