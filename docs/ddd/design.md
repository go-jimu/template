# Tactical Design

This file is the authority for tactical architecture. Business meaning and
context relationships are defined in [model.md](model.md).

## User write model

`User` is the only Aggregate Root in the User context. The User `Application`
methods that write operate on one User and use the Domain-owned Repository
collection:

- `CreateUser` creates a valid Aggregate, saves it, then drains its Domain
  Events and dispatches them with one `DispatchAll`;
- `ChangePassword` loads one Aggregate, invokes credential behavior, and saves
  it; and
- Repository persistence never dispatches Domain Events.

Infrastructure converts explicitly between the storage Data Object and the
Aggregate. Rehydration initializes the event collection. Exported Aggregate
fields are intentionally available to `infrastructure/convert.go`, which copies
slices such as the password hash so the two never share memory; do not add
a `Restore`, `Rehydrate`, or reflection-based construction hook solely for
persistence.

The persistence adapter stores version `1` for a new User. Updates compare the
loaded version and increment it atomically. A zero-row update maps to the
Domain concurrency error.

## User read model

Product reads live in `application/query`, separate from the write-side
`Application`: the `query.Users` query object serves `Get` and `List` straight
from the QueryRepository. Reads have no invariant, transaction, or Domain
Event, so they skip the Aggregate and the write-side orchestration, and the two
sides keep separate dependencies. Page-size normalization belongs to
`Users.List`; xorm query construction remains in Infrastructure.

`Users.List` pages by keyset, newest User first. User IDs are UUIDv7, so ID
order is creation order and the last ID of a page is the cursor: the next page
reads `id < cursor ORDER BY id DESC`, served by `idx_deleted_at_id`. The
repository fetches one extra row to know whether a next page exists, and there
is no total count. Transport exposes the cursor as an opaque base64url
`page_token`.

This CQRS split is intentionally lightweight: it separates read semantics from
the write Repository without introducing a separate database or projection
pipeline.

## Inbound protocol boundary

ConnectRPC handlers live in the User Transport adapter. They perform protocol
validation and mapping, delegate once to the Application (writes) or
`query.Users` (reads), map the result,
and translate errors to Connect codes.

Generated API types remain outside Domain and Application inputs. Unexpected
internal errors retain their local diagnostic cause but expose only a safe
public Connect message.

The shared ConnectRPC interceptor creates request-scoped logging context,
recovers panics, and owns the single outer error log for an RPC request.

## Domain Events and Integration Messages

`UserCreated` is a User Domain Event. It is recorded by the Aggregate and
dispatched only after persistence succeeds. It is keyed by User ID, so the
dispatcher handles one User's events in order while its workers handle
different Users concurrently.

The User Application Domain Event Handler is also the publication boundary for
this example: it translates `UserCreated` into the generated
`UserRegisteredV1` published contract and calls the provider-neutral
`message.Publisher`.

Kafka publishing, payload resolution, polling, offset handling, failure policy,
client lifecycle, and shutdown are runtime mechanics. They do not belong in
Domain or Application.

Notification receives `UserRegisteredV1` through
`transport/messagesubscriber`. The Subscriber validates the published payload,
translates it into Notification's own language, a `Recipient` with an address
and display name plus the message ID as `Reference`, and delegates once to
`Application.SendWelcomeNotification`. Notification does not conform to User's
model; only the subscriber knows `UserRegisteredV1`. Subscriber failures
are returned to the Kafka runtime; the Subscriber does not print an inner error
log.

## Consistency and reliability

One write use case changes one User Aggregate. User persistence is its
consistency boundary. Domain Event dispatch and Integration Message publication
happen after that persistence boundary.

Event dispatch is best effort in the template: a dispatch failure is recorded
where it is deliberately suppressed, and registration still succeeds. The
Kafka consumer's `failure-policy` is `drop` by default, which commits a failed
message, or `dlq`, which first publishes it to `<topic>.dlq`.

The template deliberately excludes outbox, retry, durable notification state,
and idempotency storage. A production service must make an explicit
failure-tolerance decision before adding any of those mechanisms.

## Error propagation and logging

An error returned by a database, broker, network, SDK, or other external
dependency is wrapped once where it enters code we control, with
`oops.With(...).Wrap(err)` carrying the fields that layer owns; the wrap
records the origin stack and preserves `errors.Is` identity. Outer layers
return it as is unless they own a new field.

Returning Domain, Application, Infrastructure, and Transport code does not
print the same error repeatedly. The outer execution boundary prints it once
with `sloghelper.Error(err)`:

- the ConnectRPC interceptor owns RPC failures;
- the Kafka error observer owns consumed-message failures;
- a Domain Event Handler owns its terminal failure because its handler contract
  has no error return; and
- code that intentionally suppresses an error logs it at the suppression point.

Completion records belong to the runtime, not to Application: the ConnectRPC
interceptor records each request, and the message runtime wraps every
subscribed handler to record each handled message.

## Runtime composition and lifecycle

Each bounded context exposes an Fx module that owns its constructors and
registrations. `cmd/server/main.go` loads and supplies aggregate configuration,
selects modules, configures process timeouts, and runs Fx; it does not validate
or construct individual components.

Each runtime component validates its own option. Constructors register the
hooks of the resource they open: the database package pings and closes the
xorm engine, and messagebus pings and closes the Kafka client.

`pkg.Lifecycle` orders the runtimes that hand work to each other. It is listed
after every bounded-context module and registers the dispatcher drain, the
Kafka consumer, and the ConnectRPC server, in that order. Fx stops hooks in
reverse, so shutdown runs:

1. the server turns `/readyz` unavailable and drains in-flight requests;
2. the consumer stops fetching and finishes its current message;
3. the dispatcher handles every accepted Domain Event; and
4. the Kafka client and the database close.

Each step can still use everything stopped after it.

## Verification seams

- Domain tests cover User invariants, password behavior, and recorded facts.
- Application tests cover orchestration, persistence-before-dispatch ordering,
  and the suppressed dispatch failure log; query tests cover page-size
  normalization.
- Transport tests cover Connect contracts, error mapping, Integration Message
  validation, and one-delegation behavior.
- Infrastructure tests cover explicit conversion and optimistic-lock SQL
  shape. Integration tests (`make integration`) apply `migrations/` to MySQL in
  Docker and cover the repository, keyset paging, and the list query plan.
- Runtime tests cover option validation, Fx graph completeness, request logging,
  and start/stop behavior without requiring real MySQL or Kafka.
- The architecture test parses imports and rejects cross-context or
  outward-pointing dependencies in Domain, Application, Infrastructure, and
  Transport, and keeps the private RPC contract in Transport.
