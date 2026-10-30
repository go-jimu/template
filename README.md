# Go-Jimu DDD Template

An executable reference for building a small Domain-Driven Design service with
Go, [go-jimu components](https://github.com/go-jimu/components), Uber Fx,
ConnectRPC, xorm, MySQL, and Kafka.

The example is intentionally explicit. It shows where business rules,
orchestration, inbound mapping, persistence, messages, errors, and lifecycle
belong so a new bounded context can copy a coherent shape instead of copying
framework glue.

## Read the project quickly

For an AI coding agent, start with [AGENTS.md](AGENTS.md). The architecture
sources then have one responsibility each:

1. [docs/ddd/model.md](docs/ddd/model.md) — business terms, authority,
   invariants, scenarios, and context relationships.
2. [docs/ddd/design.md](docs/ddd/design.md) — Aggregate, Repository, CQRS,
   event/message, error, and runtime decisions.
3. [docs/architecture.md](docs/architecture.md) — dependency direction and a
   table routing each kind of change to its owner.

For the current state, the model owns business meaning and the design owns
tactical choices. Code and tests show how those decisions are implemented.

## What the example demonstrates

- A User Aggregate with validated registration and credential behavior.
- A write Repository shaped as `Get`/`Save` with optimistic locking.
- CQRS-style read DTOs through an Application-owned QueryRepository.
- A dedicated ConnectRPC Transport layer that only performs inbound mapping.
- Domain Events for same-context facts and Integration Messages for
  cross-context published language.
- A go-jimu Kafka runtime for Integration Message publication and consumption.
- Uber Fx module composition with an explicit shutdown order: server,
  consumer, Domain Event drain, then shared clients.
- Keyset pagination over UUIDv7 IDs with an opaque page token.
- `oops` error propagation and one outer error log through
  `sloghelper.Error`.
- Automated import-boundary checks plus behavior tests at each layer.

## Project layout

~~~text
cmd/server/                       server entry point and module selection
cmd/client/                       example client
configs/server/defaults.yml       runtime configuration
docs/architecture.md              architecture navigation and change routing
docs/ddd/model.md                 business semantic authority
docs/ddd/design.md                tactical design authority
internal/business/
  user/
    domain/                       Aggregate, errors, Domain Events, Repository port
    application/
      application.go              Application; use cases are its methods
      assembler.go                Aggregate-to-DTO assembly
      query/                      read DTOs and QueryRepository port
      eventhandler/               Domain Event reaction/publication boundary
    transport/connectrpc/         inbound generated-protocol adapter
    infrastructure/               xorm write/read adapters and conversion
    user.go                       bounded-context Fx module
  notification/
    application/                  local welcome-notification use case
    transport/messagesubscriber/  inbound Integration Message adapter
    notification.go               bounded-context Fx module
internal/pkg/
  connectrpc/                     shared HTTP/2 runtime and interceptor
  database/                       shared xorm engine
  eventbus/                       in-process Domain Event dispatcher
  messagebus/                     go-jimu Kafka runtime
  lifecycle.go                    start and shutdown order of the runtimes
migrations/                       ordered MySQL migrations
proto/                            source API and published message contracts
gen/                              generated code; never edit manually
~~~

Dependencies point inward:

~~~text
Transport -> Application -> Domain
Infrastructure ---------> Domain/Application ports
Runtime ----------------> modules and technical adapters
~~~

## Where to make a change

| Goal | Primary location |
| --- | --- |
| Change a User invariant | `internal/business/user/domain` |
| Add a write use case | an `Application` method plus Domain behavior |
| Add a product read | an `Application` method, `application/query`, and the QueryRepository adapter |
| Change the Connect API | `proto/user/private/v1`, regenerate, then Transport mapping |
| Publish a new cross-context fact | integration Proto plus User publication boundary |
| Consume an Integration Message | receiving `transport/messagesubscriber` |
| Change xorm or SQL mapping | `infrastructure` and a new `migrations/NNN_*.sql` |
| Change Kafka, HTTP, DB, config, or lifecycle mechanics | `internal/pkg` |
| Register a context component | the owning `<context>.go` Fx module |

Read [docs/architecture.md](docs/architecture.md) before a cross-layer change.
Do not infer a business rule from a directory or database column.

## Request and message flows

A query never loads the write Aggregate:

~~~text
Connect request -> Transport -> Application.GetUser -> QueryRepository -> read DTO
~~~

A write use case changes one Aggregate:

~~~text
Connect request
  -> Transport
  -> Application.CreateUser / ChangePassword
  -> Repository.Get (when required)
  -> Aggregate behavior
  -> Repository.Save
  -> drain Domain Events
  -> Event Dispatcher (DispatchAll)
~~~

Cross-context publication uses a separate contract:

~~~text
UserCreated (User Domain Event)
  -> UserCreatedHandler (Application)
  -> UserRegisteredV1 (Integration Message)
  -> go-jimu Kafka runtime
  -> Notification UserRegisteredSubscriber (Transport)
  -> Application.SendWelcomeNotification
~~~

`UserCreated` and `UserRegisteredV1` are intentionally different. The Kafka
`failure-policy` is `drop` or `dlq`; the example does not implement outbox,
idempotency, retry, or durable notification state.

## Getting started

Requirements:

- Go 1.26+
- MySQL 8+
- Kafka
- Make
- Buf and the pinned Proto plugins when regenerating contracts

Create the schema:

~~~bash
mysql -u root -p jimu < migrations/001_create_user_account.sql
~~~

Ensure Kafka is reachable at `localhost:9092` and create the
`user.integration.v1.UserRegisteredV1` topic, or override the broker through
`KAFKA_BROKER`.

Start the server and invoke the example client:

~~~bash
make server
make client
~~~

The endpoint supports Connect, gRPC, and gRPC-Web. `GET /healthz` reports
liveness; `GET /readyz` reports readiness and turns unavailable once shutdown
begins.

## API

| RPC | Side | Purpose |
| --- | --- | --- |
| Create | Command | Create one User and publish registration |
| Get | Query | Return one User read model |
| ChangePassword | Command | Apply credential behavior and optimistic locking |
| List | Query | Return one keyset page of User read models, newest first; pass `next_page_token` back as `page_token` |

## Configuration

`configs/server/defaults.yml` is loaded explicitly; run the server from the
repository root. Environment placeholders keep
deployment-specific values out of code:

~~~yaml
mysql:
  host: ${MYSQL_HOST:localhost}
kafka:
  brokers:
    - ${KAFKA_BROKER:localhost:9092}
  failure-policy: ${KAFKA_FAILURE_POLICY:drop}   # or dlq
connect:
  addr: ${CONNECT_ADDR::8080}
~~~

Each runtime component validates its own Option/Config in its constructor.
`main` only loads and supplies aggregate configuration. Never log a password,
token, or full DSN.

## Error handling

- An external dependency error is wrapped once where it enters our code, with
  `oops.With(...).Wrap(err)` carrying that layer's fields and the origin stack.
- Outer layers return it as is unless they own a new field.
- Domain errors retain `errors.Is` identity.
- Only the outer execution boundary prints the error, always through
  `sloghelper.Error(err)`.

Completion records come from the runtime: the ConnectRPC interceptor logs each
request and the message runtime logs each handled message. Message Subscribers
return failures to the Kafka observer. Domain Event
Handlers log terminal failures because their handler contract has no error
return.

## Protocol generation

Install pinned tools and regenerate contracts:

~~~bash
make tools
make generate
~~~

API contracts live under `proto/<context>/private/v1`. Published Integration
Message contracts live under `proto/<context>/integration/v1`. Generated code
belongs under `gen/` and is never edited manually.

## Adding a feature

1. Name the actor/trigger, use case, invariant owner, and observable outcome.
2. Update `model.md` only when accepted business meaning changes.
3. Update `design.md` only when Aggregate, transaction, Repository, CQRS,
   collaboration, or runtime ownership changes.
4. Put one-Aggregate rules on the Aggregate or a Value Object.
5. Put write orchestration in an `Application` method.
6. Put product reads in an `Application` method backed by `application/query`
   and its QueryRepository.
7. Map protocol input/output in Transport; never import generated API code into
   Domain.
8. Use a Domain Event for a same-context fact and an Integration Message for
   stable cross-context published language.
9. Keep database, broker, SDK, config, and lifecycle mechanics in
   Infrastructure or `internal/pkg`.
10. Register constructors and handlers in the owning context's Fx module.
11. Add the lowest-boundary test that catches the intended regression.
12. Run `make verify`.

## Verification

~~~bash
make verify
~~~

The target runs Buf lint/generation, all Go tests with the race detector and
coverage, and `go vet`. The architecture test parses business-package imports
and rejects outward or cross-context dependencies.

Repository SQL tests use a test-only SQL driver for optimistic-lock shape.
`make integration` applies `migrations/` to MySQL in Docker through
testcontainers and covers the repositories, keyset paging, and the list query
plan; CI runs it on every pull request. Kafka suites remain
environment-specific.

## License

MIT
