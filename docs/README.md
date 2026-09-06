# 📬 sublyra-api — technical documentation

`sublyra-api` is a study project for reliable, asynchronous subscription workflows. It exposes an HTTP API for newsletter opt-in and opt-out, persists subscription state in MongoDB, and records the corresponding integration event using the **Transactional Outbox** pattern.
`sublyra-api` is a study project for reliable, asynchronous subscription workflows. It exposes an HTTP API for newsletter opt-in and opt-out, persists subscription state in MongoDB, records integration events using the **Transactional Outbox** pattern, and processes email delivery asynchronously via **RabbitMQ** and **Resend**.

> Current scope: the transactional write side is implemented. An outbox relay, RabbitMQ publisher/consumer, and Resend email adapter are planned but are not implemented yet.

To read this documentation in Portuguese, see [`README.pt-BR.md`](README.pt-BR.md).

## Architecture

The code follows a Ports and Adapters-inspired layout and uses Uber Fx for dependency injection and lifecycle management.
The code follows a Ports and Adapters (Hexagonal Architecture) layout and uses Uber Fx for dependency injection and application lifecycle management.

```text
cmd/api/                         application entry point
internal/
├── config/                      environment configuration
├── core/
│   ├── domain/                  subscription and outbox models
│   ├── ports/                   application error contract
│   ├── domain/                  subscription, outbox models, and domain events
│   ├── ports/                   application error contracts
│   └── services/                subscription use cases
├── inbound/http/rest/           Fiber controllers and middleware
├── outbound/mongodb/            MongoDB repositories and schemas
├── infra/                       HTTP server, logger, MongoDB client
├── inbound/http/rest/           Fiber controllers, DTOs, and middleware
├── infra/                       HTTP server, logger, MongoDB, RabbitMQ, and Resend mailer
├── outbound/
│   ├── mongodb/                 MongoDB repositories and schemas
│   └── rabbitmq/                RabbitMQ topology, publisher, and async workers (relay, email)
└── platform/                    JWT and validation helpers
```

The dependency flow is:
The data flow is asynchronous and resilient:

```text
HTTP request → Fiber controller → subscription service → MongoDB repository
                                                       ├─ subscriptions
                                                       └─ outbox
HTTP Request → Fiber Controller → Subscription Service → MongoDB Repository
                                                         ├─ subscriptions
                                                         └─ outbox (pending)
                                                                 │
                                                                 ▼
                                                         Outbox Relay Worker
                                                                 │
                                                                 ▼
                                                         RabbitMQ Exchange (sublyra.events)
                                                                 │
                                                                 ▼
                                                         Email Consumer Worker
                                                                 │
                                                                 ▼
                                                         Resend API (Mailer)
```

Main technologies: Go 1.26.3, Fiber v3, MongoDB Go Driver v2, Uber Fx, and JWT.
Main technologies: Go 1.26.3, Fiber v3, MongoDB Go Driver v2, RabbitMQ (amqp091-go), Resend Go SDK, Uber Fx, and JWT.

## Transactional Outbox
## Transactional Outbox & Async Pipeline

The problem explored here is the dual-write problem. A subscription request needs to update business state and eventually trigger an external side effect, such as sending an email. Writing to MongoDB and calling an email provider directly are two independent operations: either one can succeed while the other fails.
The core pattern explored here is solving the dual-write problem. A subscription request must update business state and eventually trigger an external side effect (sending an email). Writing to MongoDB and calling an external API directly are independent operations: either one can succeed while the other fails.

The current implementation writes both documents inside one MongoDB transaction:
The implementation writes both documents inside a single MongoDB transaction:

1. The service creates or updates a document in `subscriptions.subscriptions`.
2. In the same transaction, it appends an event to `subscriptions.outbox`.
3. MongoDB commits both operations or rolls both back.
4. A future relay will read pending outbox events and publish them to RabbitMQ.
5. A future consumer will use Resend to deliver the confirmation/cancellation email.
2. In the same transaction, it appends an integration event to `subscriptions.outbox` with status `pending`.
3. MongoDB commits both operations atomically or rolls both back.
4. The **Outbox Relay Worker** continuously polls pending outbox events, updates their status to `processing`, and publishes them to the RabbitMQ exchange (`sublyra.events`). Upon successful publish, the outbox event status becomes `published`.
5. The **Email Consumer Worker** consumes events from the RabbitMQ queue (`sublyra.email`), renders HTML templates (`confirmation.html` or `cancellation.html`), and delivers emails via the **Resend API**.
6. Upon successful email dispatch, the outbox event status is updated to `delivered`. If dispatch fails or max attempts are reached, the message is routed to retry/DLQ queues and marked `failed`.

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
  "event": "outbox_subscription_confirmation_requested | outbox_subscription_cancellation_requested",
  "attempts": 0,
  "payload": {
    "email": "person@example.com",
    "status": "pending",
    "confirmation_token": "JWT"
  },
  "status": "pending",
  "status": "pending | processing | published | delivered | failed",
  "last_error": "optional error message",
  "published_at": "optional datetime",
  "created_at": "datetime",
  "updated_at": "datetime"
}
```

Supported event names are `outbox_subscription_confirmation_requested` and `outbox_subscription_cancellation_requested`. The status model defines `pending`, `published`, and `failed`; processing and transitions are future work.
Supported event names:
- `outbox_subscription_confirmation_requested`
- `outbox_subscription_cancellation_requested`

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
The server currently applies request logging, panic recovery, request IDs, and an in-memory rate limit of 3 requests per 30 seconds. Ready-to-run HTTP requests are available in [`.http/subscriptions.http`](../.http/subscriptions.http).

## RabbitMQ Topology & Messaging Architecture

The application automatically declares its messaging topology on startup:

- **Exchanges**:
  - `sublyra.events` (`direct`, durable): Main event exchange.
  - `sublyra.retry` (`direct`, durable): Holding exchange for delayed retries (30s TTL).
  - `sublyra.dlx` (`direct`, durable): Dead-letter exchange for unrecoverable or exhausted messages.
- **Queues**:
  - `sublyra.email`: Active email consumer queue (configured with `x-dead-letter-exchange: sublyra.retry`).
  - `sublyra.email.retry`: Temporary holding queue with `x-message-ttl: 30000` (30s) that dead-letters back to `sublyra.events`.
  - `sublyra.email.dlq`: Dead-letter queue for failed/unprocessable messages.
- **Routing Keys**:
  - `subscription.confirmation.requested`
  - `subscription.cancellation.requested`
  - `subscription.email.retry`
  - `subscription.dead`

## Configuration

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `SERVER_HOST` | no | `localhost` | HTTP bind address |
| `SERVER_PORT` | no | `8080` | HTTP port |
| `APP_SCHEME` | no | `http` | URL scheme used for constructing confirmation/cancellation link URLs |
| `APP_HOST` | no | `localhost:8080` | Application host domain/address used for constructing link URLs |
| `MONGO_URI` | yes | — | MongoDB replica-set connection URI |
| `JWT_SECRET_KEY` | yes | — | Secret key for signing confirmation/cancellation JWTs |
| `RABBITMQ_URI` | yes | — | RabbitMQ AMQP connection URI |
| `RESEND_SECRET_KEY` | yes | — | Resend API key for sending email notifications |
| `RESEND_FROM_EMAIL` | no | `onboarding@resend.dev` | Sender email address for outgoing emails |
| `OUTBOX_POLL_INTERVAL` | no | `2s` | Outbox relay polling interval |
| `OUTBOX_BATCH_SIZE` | no | `50` | Maximum outbox batch size per polling iteration |
| `OUTBOX_MAX_ATTEMPTS` | no | `5` | Maximum retry attempts before marking outbox event as `failed` |
| `RABBITMQ_PREFETCH` | no | `5` | Consumer prefetch count for RabbitMQ worker channel |

Do not commit real credentials. Use local values in `.env` and keep only placeholders in `.env.example`.

## Development commands

```bash
make run    # go run ./cmd/api
make test   # go test ./...
make test   # go test ./... -count=1
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
The test suite covers domain logic and state transitions, cooldown rules, JWT generation and validation, outbox document construction, repository schema mappers, RabbitMQ topology declaration, and publisher routing.

Docker Compose builds the API and provisions MongoDB 8.0 as a single-node replica set. The MongoDB healthcheck initializes `rs0` and the API waits until the node becomes primary. Inside the Compose network, `MONGO_URI` is automatically overridden to use the `mongo` service:
## Docker Infrastructure

Docker Compose provisions the entire environment: MongoDB 8.0 single-node replica set, RabbitMQ 4 with Management UI, and the API container.

```bash
docker compose -f docker/docker-compose.yml up --build
```

The API is exposed on port `8080`, MongoDB on port `27017`, and database data is persisted in the named volume `mongo_data`. To stop the services while retaining data, run `docker compose -f docker/docker-compose.yml down`. Add `--volumes` only when you intentionally want to delete the local database.
Exposed ports:
- **API**: `http://localhost:8080`
- **MongoDB**: `localhost:27017`
- **RabbitMQ AMQP**: `localhost:5672`
- **RabbitMQ Management Dashboard**: `http://localhost:15672` (credentials: `guest` / `guest`)

## Planned RabbitMQ and Resend flow
Database and message broker state are persisted in named volumes `mongo_data` and `rabbitmq_data`. To stop services while retaining data, run `docker compose -f docker/docker-compose.yml down`. Add `--volumes` to delete local database and queue data.

```text
MongoDB outbox → relay → RabbitMQ exchange → email consumer → Resend API
```
## Current Limitations & Future Enhancements

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
- **Outbox Retention & Cleanup**: Automated deletion or archiving of historical `delivered` or `failed` outbox events is not yet implemented.
- **Distributed Lock / Concurrent Relay**: Polling relay assumes a single running relay worker instance. Multi-replica worker claiming with atomic MongoDB locks is a planned enhancement.
- **Automated Integration Tests**: Full end-to-end integration tests using testcontainers for MongoDB and RabbitMQ are planned.
