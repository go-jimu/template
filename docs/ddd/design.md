# Tactical Design

This file is the authority for tactical architecture. Business meaning and
context relationships are defined in [model.md](model.md).

## User write model

`User` is the only Aggregate Root in the User context. Commands operate on one
User and use the Domain-owned Repository collection:

- registration creates a valid Aggregate, saves it, then drains and dispatches
  its Domain Events;
- password change loads one Aggregate, invokes credential behavior, and saves
  it; and
- Repository persistence never dispatches Domain Events.

Infrastructure converts explicitly between the storage Data Object and the
Aggregate. Rehydration initializes the event collection. Exported Aggregate
fields are intentionally available to the Infrastructure converter; do not add
a `Restore`, `Rehydrate`, or reflection-based construction hook solely for
persistence.

The persistence adapter stores version `1` for a new User. Updates compare the
loaded version and increment it atomically. A zero-row update maps to the
Domain concurrency error.

## User read model

Product reads use an Application-owned QueryRepository and read DTOs. `Get`
and `List` do not load the write Aggregate. Pagination normalization belongs to
the query use case; xorm query construction remains in Infrastructure.

This CQRS split is intentionally lightweight: it separates read semantics from
the write Repository without introducing a separate database or projection
pipeline.

## Inbound protocol boundary

ConnectRPC handlers live in the User Transport adapter. They perform protocol
validation and mapping, delegate once to an Application command or query, map
the result, and translate errors to Connect codes.

Generated API types remain outside Domain and command/query inputs. Unexpected
internal errors retain their local diagnostic cause but expose only a safe
public Connect message.

The shared ConnectRPC interceptor creates request-scoped logging context,
recovers panics, and owns the single outer error log for an RPC request.

## Domain Events and Integration Messages

`UserCreated` is a User Domain Event. It is recorded by the Aggregate and
dispatched only after persistence succeeds.

The User Application Domain Event Handler is also the publication boundary for
this example: it translates `UserCreated` into the generated
`UserRegisteredV1` published contract and calls the provider-neutral
`message.Publisher`.

Kafka publishing, payload resolution, polling, offset handling, failure policy,
client lifecycle, and shutdown are runtime mechanics. They do not belong in
Domain or Application.

Notification receives `UserRegisteredV1` through
`transport/messagesubscriber`. The Subscriber validates the published payload,
maps it to `SendWelcomeNotification`, and delegates once. Subscriber failures
are returned to the Kafka runtime; the Subscriber does not print an inner error
log.

## Consistency and reliability

One command changes one User Aggregate. User persistence is the command's
consistency boundary. Domain Event dispatch and Integration Message publication
happen after that persistence boundary.

Event dispatch is best effort in the template: a dispatch failure is recorded
where it is deliberately suppressed, and registration still succeeds. The
Kafka consumer uses `DropAndCommitPolicy` so the example stays focused on
boundary wiring.

The template deliberately excludes outbox, retry, DLQ, durable notification
state, and idempotency storage. A production service must make an explicit
failure-tolerance decision before adding any of those mechanisms.

## Error propagation and logging

An error returned by a database, broker, network, SDK, or other external
dependency receives an origin stack through `oops.Wrap` where it enters code we
control. Internal layers may add the fields they own with
`oops.With(...).Wrap(err)` while preserving `errors.Is` identity.

Returning Domain, Application, Infrastructure, and Transport code does not
print the same error repeatedly. The outer execution boundary prints it once
with `sloghelper.Error(err)`:

- the ConnectRPC interceptor owns RPC failures;
- the Kafka error observer owns consumed-message failures;
- a Domain Event Handler owns its terminal failure because its handler contract
  has no error return; and
- code that intentionally suppresses an error logs it at the suppression point.

Application commands and queries still emit completion records. On failure
those inner records use info level and omit the error details.

## Runtime composition and lifecycle

Each bounded context exposes an Fx module that owns its constructors and
registrations. `cmd/main.go` loads and supplies aggregate configuration, selects
modules, configures process timeouts, and runs Fx; it does not validate or
construct individual components.

Each runtime component validates its own option and owns the lifecycle of the
resource it creates. ConnectRPC owns its listener and HTTP shutdown, EventBus
owns dispatcher close/drain, Kafka owns its consumer goroutine and client
close, and the database package owns the xorm engine.

## Verification seams

- Domain tests cover User invariants, password behavior, and recorded facts.
- Application tests cover orchestration, persistence-before-dispatch ordering,
  read normalization, error enrichment, and completion logs.
- Transport tests cover Connect contracts, error mapping, Integration Message
  validation, and one-delegation behavior.
- Infrastructure tests cover explicit conversion and optimistic-lock SQL
  shape.
- Runtime tests cover option validation, Fx graph completeness, request logging,
  and start/stop behavior without requiring real MySQL or Kafka.
- The architecture test parses imports and rejects cross-context or
  outward-pointing dependencies in Domain, Application, and Transport.
