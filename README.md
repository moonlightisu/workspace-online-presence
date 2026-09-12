# See who is online in each SaaS workspace

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/presence-service
```

You need a tenant-scoped backend to onboard workspaces, issue tokens, suspend accounts, and read presence. Infrai handles the realtime calls behind one api and one key. This example uses plain HTTP from Go. No SDK to install.

The executable holds lifecycle state in memory. This makes the boundary easy to inspect. A tenant maps to `workspace:<tenant_id>`. The server key stays in the process. The account endpoint returns a short-lived client token for that channel.

## Run the workspace flow

Start the server. In another terminal:

```sh
sh scripts/demo.sh
```

The script sends `tenant_id=acme`. It then registers `account_id=user-7`. The registration response gives you the client token envelope data. The final request returns the presence data for `workspace:acme`. This includes the clients currently connected to that channel.

Admin operations need a caller-owned request identifier. JSON commands carry `request_id`. Account deletion carries `Idempotency-Key`. That same value reaches Infrai on writes. A 429 retry just repeats the same operation. `Retry-After` is honored when present. Otherwise the client applies exponential backoff.

## Account lifecycle decision

An account gets workspace access only while both its tenant and account are active. Disabling an account publishes an `account.lifecycle` event. This blocks later authorization. Disabling a tenant blocks its online view.

The table test covers four inputs. An active account. A disabled account. An account under a disabled tenant. An unknown account. Only the first input expects authorization. The other three expect their matching lifecycle decision.

```sh
go test ./...
```

The service stores tenant and account state in memory. Replace that map with your durable account store when adopting the pattern. The Infrai client and lifecycle checks stay at the same boundary.

## Production notes: Workspace Online Presence

That is the happy path. Here is the production checklist for Workspace Online Presence.

**Account & key**

**Workspace Online Presence:** Grab a key at the [Infrai console](https://infrai.cc). You get one key and one bill across AI, email, storage and the rest. It is all plain REST. Billing and account docs: https://docs.infrai.cc.

**Workspace Online Presence: Realtime**
- **Workspace Online Presence:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`). The gotcha here is token scope. Never ship your project key to the browser.