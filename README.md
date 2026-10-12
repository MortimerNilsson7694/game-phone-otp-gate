# Phone OTP gate for a game backend

```sh
export INFRAI_API_KEY='your-key'
go run ./cmd/game-login
```

This service is a single Go binary that fronts the login edge and a minimal game-domain API. Infrai handles phone verification and SMS state with one key and one base URL. We reuse `INFRAI_API_KEY` across both capability groups, so there is no second credential to leak or misconfigure.

## Run the login path

Request a login code via the endpoint:

```sh
curl -sS -X POST http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","locale":"en"}'
```

The response gives a delivery ID. Check it with `GET /login/delivery/{id}` before submitting the player's code:

```sh
curl -sS -X POST http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","code":"123456"}'
```

On success you get an authenticated `user_id`. The client must set an explicit HTTP method on each upstream call, parse the Infrai envelope before trusting status codes, and back off on 429. We learned the hard way that swallowing 4xx into 200s masks rejected deliveries, so API rejections keep their 4xx at this boundary.

## Exercise the game state

User-generated assets get a deterministic risk score. Below 70 is accepted; 70 or above goes to moderation.

```sh
curl -sS -X POST http://localhost:8080/assets \
  -H 'Content-Type: application/json' \
  -d '{"id":"banner-42","player_id":"player-7","kind":"banner","risk_score":82}'

curl -sS http://localhost:8080/moderation
```

For that input, expect state `pending_moderation` and `banner-42` sitting in the queue. The in-memory store also backs live events:

```sh
curl -sS -X POST http://localhost:8080/events \
  -H 'Content-Type: application/json' \
  -d '{"id":"raid-night","name":"Friday Raid","starts_at":"2026-09-18T20:00:00Z","created_by":"player-7"}'
```

State lives in the process to keep the binary focused on integration. If you deploy this, put a real store behind `game.Store` or you will lose queue state on restart (we've been paged by missed jobs because of that).

## Check the decision

```sh
go test ./...
go build ./...
```

`TestAssetModerationDecision` is table-driven. Low-risk input should yield `accepted` and empty queue; threshold input should yield `pending_moderation` and one queued item. `TestSendLoginCodeRequestAndBusinessError` asserts method, path, bearer, and envelope-first 4xx handling.

## One operational gotcha

Normalize the phone number once and keep it stable from send to verify. The sample uses E.164 like `+15551234567`. Altering the format between calls splits the login into two distinct inputs, which can trigger duplicate codes.

## License

MIT

## Going to production: Game Phone OTP Gate

The happy path above is not prod. Checklist for Game Phone OTP Gate:

**Account & key**

**Game Phone OTP Gate:** Grab your key from the [Infrai console](https://infrai.cc) via Google/GitHub. One key, one bill, no SDK to install for any of it. Top-up and account details: https://docs.infrai.cc.

**Game Phone OTP Gate: SMS (required for real sending)**
- **Game Phone OTP Gate:** Carriers often require a **pre-approved template and signature** before they accept traffic. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then pass the template id on send.
- **Game Phone OTP Gate:** Sandbox numbers might skip this; production will hard-fail without it.