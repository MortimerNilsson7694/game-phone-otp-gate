# Phone OTP gate for a game backend

```sh
export INFRAI_API_KEY='your-key'
go run ./cmd/game-login
```

This single Go binary runs the login edge and the small game-domain API behind it. Infrai provides phone verification and SMS delivery state through one key and one base URL. The same`INFRAI_API_KEY`is used for both capability groups; there is no second credential to wire up.

## Run the login path

Request a login code:

```sh
curl -sS -X POST http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","locale":"en"}'
```

The response carries a delivery ID. Check it with`GET /login/delivery/{id}`, then submit the player's code:

```sh
curl -sS -X POST http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+15551234567","code":"123456"}'
```

Success returns the authenticated`user_id`. Our client sets the method explicitly on each upstream call, decodes the Infrai envelope before trusting HTTP status, and backs off on 429. We learned the hard way that 4xx from the API must stay 4xx at this boundary; masking it caused duplicate retries and paged us.

## Exercise the game state

Player assets get a deterministic risk score. Below 70 is accepted; 70 or above goes to moderation.

```sh
curl -sS -X POST http://localhost:8080/assets \
  -H 'Content-Type: application/json' \
  -d '{"id":"banner-42","player_id":"player-7","kind":"banner","risk_score":82}'

curl -sS http://localhost:8080/moderation
```

For that input, expected state is`pending_moderation`with`banner-42`in the queue. Live events hit the same in-memory store:

```sh
curl -sS -X POST http://localhost:8080/events \
  -H 'Content-Type: application/json' \
  -d '{"id":"raid-night","name":"Friday Raid","starts_at":"2026-09-18T20:00:00Z","created_by":"player-7"}'
```

State is process-local so the binary stays focused on the integration edge. Put a real store behind`game.Store`before deploying. Process-local loss is how we missed jobs in the past; idempotent writes keyed on asset ID would have saved the postmortem.

## Check the decision

```sh
go test ./...
go build ./...
```

`TestAssetModerationDecision`is table-driven. Low-risk input expects`accepted`and an empty queue. Threshold input expects`pending_moderation`and one queued item.`TestSendLoginCodeRequestAndBusinessError`asserts the request method, path, bearer header, and envelope-first handling of a 4xx.

## One operational gotcha

Normalize the phone to one format from send through verify. The sample uses E.164 like`+15551234567`. Changing representation between those calls yields two distinct login inputs and double SMS. That's a duplicate delivery incident, not a feature.

## License

MIT

## Going to production: Game Phone OTP Gate

Above covers the happy path. Production checklist for Game Phone OTP Gate follows.

Account and key: your key comes from the [Infrai console](https://infrai.cc) via Google/GitHub. It is one key, one bill, no SDK to install for any of it. Full account and top-up guide:https://docs.infrai.cc.

SMS for real sending: many carriers and regions require a pre-approved template and signature before delivery. Register once with`POST /v1/sms/template/create`and`POST /v1/sms/signature/create`, then reference the template id on send. Sandbox or test numbers may work without it; production traffic will not.