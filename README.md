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
- Uber Fx module composition and graceful shutdown of ingress, event handling,
  consumers, and shared clients.
- `oops` error propagation and one outer error log through
  `sloghelper.Error`.
- Automated import-boundary checks plus behavior tests at each layer.

## Project layout

~~~text
cmd/                              process entry points and module selection
configs/defaults.yml              runtime configuration
docs/architecture.md              architecture navigation and change routing
docs/ddd/model.md                 business semantic authority
docs/ddd/design.md                tactical design authority
internal/business/
  user/
    domain/                       Aggregate, errors, Domain Events, Repository port
    application/
      command/                    write use cases
      query/                      read DTOs and QueryRepository
      eventhandler/               Domain Event reaction/publication boundary
    transport/connectrpc/         inbound generated-protocol adapter
    infrastructure/               xorm write/read adapters and conversion
    user.go                       bounded-context Fx module
  notification/
    application/command/          local welcome-notification use case
    transport/messagesubscriber/  inbound Integration Message adapter
    notification.go               bounded-context Fx module
internal/pkg/
  connectrpc/                     shared HTTP/2 runtime and interceptor
  database/                       shared xorm engine lifecycle
  eventbus/                       in-process Domain Event dispatcher lifecycle
  messagebus/                     go-jimu Kafka runtime and lifecycle
scripts/sql/                      ordered MySQL schema scripts
proto/                            source API and published message contracts
pkg/gen/                          generated code; never edit manually
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
| Add a write use case | `application/command` plus Domain behavior |
| Add a product read | `application/query` plus Infrastructure QueryRepository |
| Change the Connect API | `proto/user/v1`, regenerate, then Transport mapping |
| Publish a new cross-context fact | integration Proto plus User publication boundary |
| Consume an Integration Message | receiving `transport/messagesubscriber` |
| Change xorm or SQL mapping | `infrastructure` and `scripts/sql` |
| Change Kafka, HTTP, DB, config, or lifecycle mechanics | `internal/pkg` |
| Register a context component | the owning `<context>.go` Fx module |

Read [docs/architecture.md](docs/architecture.md) before a cross-layer change.
Do not infer a business rule from a directory or database column.

## Request and message flows

A query never loads the write Aggregate:

~~~text
Connect request -> Transport -> Query Handler -> QueryRepository -> read DTO
~~~

A command changes one Aggregate:

~~~text
Connect request
  -> Transport
  -> Command Handler
  -> Repository.Get (when required)
  -> Aggregate behavior
  -> Repository.Save
  -> drain Domain Events
  -> Event Dispatcher
~~~

Cross-context publication uses a separate contract:

~~~text
UserCreated (User Domain Event)
  -> UserCreatedHandler (Application)
  -> UserRegisteredV1 (Integration Message)
  -> go-jimu Kafka runtime
  -> Notification UserRegisteredSubscriber (Transport)
  -> SendWelcomeNotification (Application Command)
~~~

`UserCreated` and `UserRegisteredV1` are intentionally different. The Kafka
example uses `DropAndCommitPolicy` and does not implement outbox, idempotency,
retry, DLQ, or durable notification state.

## Getting started

Requirements:

- Go 1.25+
- MySQL 8+
- Kafka
- Make
- Buf and the pinned Proto plugins when regenerating contracts

Create the schema:

~~~bash
mysql -u root -p jimu < scripts/sql/001_create_user_account.sql
~~~

Ensure Kafka is reachable at `localhost:9092` and create the
`user.integration.v1.UserRegisteredV1` topic, or override the broker through
`KAFKA_BROKER`.

Start the server and invoke the example client:

~~~bash
make server
make client
~~~

The endpoint supports Connect, gRPC, and gRPC-Web. Liveness is available at
`GET /healthz`.

## API

| RPC | Side | Purpose |
| --- | --- | --- |
| Create | Command | Create one User and publish registration |
| Get | Query | Return one User read model |
| ChangePassword | Command | Apply credential behavior and optimistic locking |
| List | Query | Return a normalized page of User read models |

## Configuration

`configs/defaults.yml` is loaded explicitly. Environment placeholders keep
deployment-specific values out of code:

~~~yaml
mysql:
  host: ${MYSQL_HOST:localhost}
kafka:
  brokers:
    - ${KAFKA_BROKER:localhost:9092}
connect:
  addr: ${CONNECT_ADDR::8080}
~~~

Each runtime component validates its own Option/Config in its constructor.
`main` only loads and supplies aggregate configuration. Never log a password,
token, or full DSN.

## Error handling

- External dependency errors receive an origin stack through `oops.Wrap`.
- Internal layers add owned diagnostic fields through
  `oops.With(...).Wrap(err)`.
- Domain errors retain `errors.Is` identity.
- Only the outer execution boundary prints the error, always through
  `sloghelper.Error(err)`.

Application commands and queries still emit completion records, but returning
failure records omit error details when ConnectRPC middleware will log them.
Message Subscribers return failures to the Kafka observer. Domain Event
Handlers log terminal failures because their handler contract has no error
return.

## Protocol generation

Install pinned tools and regenerate contracts:

~~~bash
make tools
make generate
~~~

API contracts live under `proto/<context>/v1`. Published Integration Message
contracts live under `proto/<context>/integration/v1`. Generated code belongs
under `pkg/gen` and is never edited manually.

## Adding a feature

1. Name the actor/trigger, command or query, invariant owner, and observable
   outcome.
2. Update `model.md` only when accepted business meaning changes.
3. Update `design.md` only when Aggregate, transaction, Repository, CQRS,
   collaboration, or runtime ownership changes.
4. Put one-Aggregate rules on the Aggregate or a Value Object.
5. Put write orchestration in `application/command`.
6. Put product reads in `application/query` and its QueryRepository.
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
Real MySQL and Kafka suites remain environment-specific and are not required
for the template's local verification.

## License

MIT
