# 📬 sublyra-api — technical documentation

`sublyra-api` is a study project for reliable, asynchronous subscription workflows. It exposes an HTTP API for newsletter opt-in and opt-out, persists subscription state in MongoDB, and records the corresponding integration event using the **Transactional Outbox** pattern.

> Current scope: the transactional write side is implemented. An outbox relay, RabbitMQ publisher/consumer, and Resend email adapter are planned but are not implemented yet.

To read this documentation in Portuguese, see [`README.pt-BR.md`](README.pt-BR.md).

## Architecture

The code follows a Ports and Adapters-inspired layout and uses Uber Fx for dependency injection and lifecycle management.

```text
cmd/api/                         application entry point
internal/
├── config/                      environment configuration
├── core/
│   ├── domain/                  subscription and outbox models
│   ├── ports/                   application error contract
│   └── services/                subscription use cases
├── inbound/http/rest/           Fiber controllers and middleware
├── outbound/mongodb/            MongoDB repositories and schemas
├── infra/                       HTTP server, logger, MongoDB client
└── platform/                    JWT and validation helpers
```

The dependency flow is:

```text
HTTP request → Fiber controller → subscription service → MongoDB repository
                                                       ├─ subscriptions
                                                       └─ outbox
```

Main technologies: Go 1.26.3, Fiber v3, MongoDB Go Driver v2, Uber Fx, and JWT.

## Transactional Outbox

The problem explored here is the dual-write problem. A subscription request needs to update business state and eventually trigger an external side effect, such as sending an email. Writing to MongoDB and calling an email provider directly are two independent operations: either one can succeed while the other fails.

The current implementation writes both documents inside one MongoDB transaction:

1. The service creates or updates a document in `subscriptions.subscriptions`.
2. In the same transaction, it appends an event to `subscriptions.outbox`.
3. MongoDB commits both operations or rolls both back.
4. A future relay will read pending outbox events and publish them to RabbitMQ.
5. A future consumer will use Resend to deliver the confirmation/cancellation email.

The transactional methods are `InsertWithOutbox`, `RenewConfirmationWithOutbox`, and `RenewUnsubscribedWithOutbox` in the MongoDB subscription repository.

### Important MongoDB requirement

MongoDB multi-document transactions require a replica set or a sharded cluster. A standalone MongoDB instance is not sufficient. Local development must point `MONGO_URI` to a replica-set-enabled deployment.

### Collections

`subscriptions.subscriptions` stores the aggregate:

```json
{
  "_id": "ObjectId",
  "email": "person@example.com",
  "status": "pending | subscribed | unsubscribed",
  "confirmation_token": "optional JWT",
  "unsubscribe_token": "optional JWT",
  "subscribed_at": "optional datetime",
  "unsubscribed_at": "optional datetime",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

The `email` field has a unique index.

`subscriptions.outbox` stores integration events:

```json
{
  "_id": "ObjectId",
  "aggregate_id": "subscription ObjectId",
  "email": "person@example.com",
  "event": "outbox_subscription_confirmation_requested",
  "attempts": 0,
  "payload": {
    "email": "person@example.com",
    "status": "pending",
    "confirmation_token": "JWT"
  },
  "status": "pending",
  "published_at": "optional datetime",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

Supported event names are `outbox_subscription_confirmation_requested` and `outbox_subscription_cancellation_requested`. The status model defines `pending`, `published`, and `failed`; processing and transitions are future work.

## Subscription lifecycle

```text
new email ──request──> pending ──confirm token──> subscribed
                          ▲                            │
                          │                            │ request cancellation
                          │                            ▼
                     renew request <── unsubscribed <── confirm token
```

Confirmation and unsubscription tokens are JWTs with a 15-minute lifetime. The token stored in the subscription document must match the supplied token before the state transition is accepted.

## HTTP API

Base path: `/api/v1`

| Method | Path | Body/query | Success |
| --- | --- | --- | --- |
| `POST` | `/subscription` | `{"email":"person@example.com"}` | `202 Accepted` |
| `POST` | `/subscription/confirm?token=...` | JWT query parameter | `202 Accepted` |
| `POST` | `/unsubscription` | `{"email":"person@example.com"}` | `202 Accepted` |
| `POST` | `/unsubscription/confirm?token=...` | JWT query parameter | `202 Accepted` |

```bash
curl -i -X POST http://localhost:8080/api/v1/subscription \
  -H "Content-Type: application/json" \
  -d '{"email":"person@example.com"}'
```

Successful responses use stable application codes such as `SUBSCRIPTION_PENDING`, `SUBSCRIPTION_CONFIRMED`, `UNSUBSCRIPTION_PENDING`, and `UNSUBSCRIPTION_CONFIRMED`.

The server currently applies request logging, panic recovery, request IDs, and an in-memory rate limit of three requests per 30 seconds. CORS configuration and the token guard are not active yet. Ready-to-run requests are available in [`.http/subscriptions.http`](../.http/subscriptions.http).

## Configuration

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `SERVER_HOST` | no | `localhost` | HTTP bind address |
| `SERVER_PORT` | no | `8080` | HTTP port |
| `MONGO_URI` | yes | — | MongoDB replica-set connection URI |
| `JWT_SECRET_KEY` | yes | — | Signs confirmation/cancellation JWTs |
| `RESEND_SECRET_KEY` | currently yes | — | Reserved for the planned Resend adapter |

Do not commit real credentials. Use local values in `.env` and keep only placeholders in `.env.example`.

## Development commands

```bash
make run    # go run ./cmd/api
make test   # go test ./...
make fmt    # go fmt ./...
make tidy   # go mod tidy
make lint   # golangci-lint run
make build  # build bin/api
```

## Tests

Run the unit test suite with:

```bash
go test ./... -count=1 -cover
```

The current suite covers domain state transitions and cooldown rules, JWT validation, outbox document construction, and the main subscription service scenarios using lightweight test doubles. MongoDB transaction behavior requires the replica-set environment and is not part of the unit suite.

Docker Compose builds the API and provisions MongoDB 8.0 as a single-node replica set. The MongoDB healthcheck initializes `rs0` and the API waits until the node becomes primary. Inside the Compose network, `MONGO_URI` is automatically overridden to use the `mongo` service:

```bash
docker compose -f docker/docker-compose.yml up --build
```

The API is exposed on port `8080`, MongoDB on port `27017`, and database data is persisted in the named volume `mongo_data`. To stop the services while retaining data, run `docker compose -f docker/docker-compose.yml down`. Add `--volumes` only when you intentionally want to delete the local database.

## Planned RabbitMQ and Resend flow

```text
MongoDB outbox → relay → RabbitMQ exchange → email consumer → Resend API
```

Reliability concerns to preserve while implementing it:

- Publish with a stable event ID and use publisher confirms.
- Make consumers idempotent; delivery must be treated as at least once.
- Claim events atomically so relay instances do not concurrently publish the same work.
- Increment `attempts`, retry with backoff, and dead-letter exhausted messages.
- Mark an event `published` only after broker confirmation and store `published_at`.
- Keep provider-specific Resend code behind an outbound port.
- Add an outbox polling index, for example on `status` and `created_at`.
- Propagate `event_id`, `aggregate_id`, and request ID for observability.

## Current limitations

- No outbox polling/CDC relay exists yet.
- RabbitMQ connection, topology, and publisher are configured; the outbox relay and email consumer are not yet wired into the flow.
- Automated MongoDB integration tests are not implemented yet.
- The Compose environment uses a single MongoDB replica-set member and therefore does not provide production high availability.
- Outbox cleanup, retention, and concurrent claiming are not implemented.
- Observability is limited to HTTP and application logs.
