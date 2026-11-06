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

Generated API types remain outside Domain and Application inputs. connect-go
sends a client only the `*connect.Error` values Transport builds: an outcome
the client can act on carries its error's message, and any other error becomes
`internal` with a fixed message. The original error stays attached as the
local cause for `errors.Is` and the interceptor's log, and is never sent.

Two RPC contracts serve two audiences. Public and private name the contract:
its audience, compatibility promise, and fields.

- `user.private.v1.UserService` serves this deployment: it may expose
  internal fields such as email and change with the deployment.
- `user.public.v1.ProfileService` is externally supported for web and app
  clients. It exposes only the public view of a User (id and name), reads
  through the same `query.Users`, and only grows backward compatibly; a
  breaking change becomes `user.public.v2` alongside v1. `GetProfile` is
  marked `NO_SIDE_EFFECTS`, so clients may call it with a cacheable HTTP GET.
  Profiles are public, so it carries no authentication.

One listener, `connect.addr`, serves both contracts. Exposure is a deployment
decision: the gateway authenticates external callers and routes only the
`*.public.v*` services to the server, so every other procedure stays inside
the deployment. Caller identity between services and per-procedure access
policy belong to the platform, such as a service mesh's authorization policy;
a service authorizes access to its own resources in the Application.

The shared ConnectRPC interceptor runs on every RPC except health checks. It
creates request-scoped logging context, recovers panics, and owns the single
outer error log for an RPC request, unary or streaming.

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
Kafka consumer's `failure-policy` is `retry` by default, which handles a failed
message again in place, after a backoff and before the rest of its partition,
and drops it after 3 attempts; or `drop`, which commits it at once.

The template deliberately excludes outbox, a dead-letter store, durable
notification state, and idempotency storage. A production service must make an explicit
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

- the ConnectRPC interceptor owns RPC failures, at Info when the client caused
  them, Warn when transient, and Error for a server fault;
- the go-jimu Kafka consumer owns consumed-message failures, logging a retry at
  Warn and a dropped message at Error;
- a Domain Event Handler owns its terminal failure because its handler contract
  has no error return; and
- code that intentionally suppresses an error logs it at the suppression point.

Completion records belong to the runtime, not to Application: the ConnectRPC
interceptor records each request, and the message runtime wraps every
subscribed handler in `message.Logging` to record each handled message at
Debug.

## API logging

The shared ConnectRPC interceptor (`internal/pkg/connectrpc`) owns the logs of
every RPC; a new procedure needs no logging code of its own.

**Access log.** The interceptor writes one `request complete` record per RPC,
unary or streaming, when it ends. Health checks are not logged: probes are not
requests.

| Attribute | Value |
| --- | --- |
| `request_id` | the request's ID; see "Request ID" below |
| `operation` | the procedure, such as `/user.private.v1.UserService/Get` |
| `outcome` | `success` or `failed` |
| `connect_code` | `ok`, or the Connect code sent to the client |
| `elapsed_ms` | duration in milliseconds |
| `error` | failures only: the cause behind the Connect error, through `sloghelper.Error`, with its oops context and stack; never the message sent to the client alone |

Its level follows what the outcome means for the service:

| Outcome | Level |
| --- | --- |
| success | Info |
| client-caused: `canceled`, `invalid_argument`, `not_found`, `already_exists`, `permission_denied`, `unauthenticated`, `failed_precondition`, `aborted`, `out_of_range` | Info |
| transient: `deadline_exceeded`, `resource_exhausted`, `unavailable` | Warn |
| server fault: any other code, including a recovered panic (`internal`) | Error |

Success is Info, above sloghelper's Debug for success, on purpose: the access
log is the record of traffic that production keeps. Error alerts therefore
mean server faults, never client mistakes.

**What handlers log.** Transport, Application, and the layers below do not log
a request's start or completion, nor an error they return: the access log
records both. A handler may log a fact the outcome does not show, through
`sloghelper.FromContext(ctx)` and a `...Context(ctx, ...)` or `LogAttrs` call
with the request's ctx, so the record names the request: Debug for a
diagnostic, Info for a business decision worth keeping. A suppressed error is
logged where it is suppressed (see "Error propagation and logging").
Attribute names follow `go doc github.com/go-jimu/components/sloghelper`.

**What no log contains.** These never appear in a log attribute, nor in the
oops context of an error (`oops.With(...)`), which the access log prints:

- request or response messages, and headers such as `Authorization` and
  `Cookie`;
- passwords, tokens, keys, and DSNs;
- personal data, such as an email address, phone number, or name: log the
  `user_id` instead.

**Request ID.** The interceptor takes the request ID from `X-Request-ID`, or
generates one (`<host>/<random>-<sequence>`) when it is missing, and puts it in
the request's ctx (`internal/pkg/requestid`). The process logger adds
`request_id` to every record logged with a ctx that carries it, so the code
never adds it by hand. The Domain Event dispatcher copies it onto its
handlers' ctx, so a handler's logs, and the dispatcher's own logs about the
event, name the request that raised it. Integration Messages do not carry it
yet: a consumer's logs have no `request_id`.

For external traffic the gateway owns the ID: it generates it, replaces one an
external client sent, and writes it into the response when clients need it.
The service accepts the header as it comes, does not echo it, and does not
validate it: the JSON log handler escapes it, and the server's 16 KiB header
limit bounds its size.

## Runtime composition and lifecycle

Each bounded context exposes an Fx module that owns its constructors and
registrations. `cmd/server/main.go` loads and supplies aggregate configuration,
selects modules, configures process timeouts, and runs Fx; it does not validate
or construct individual components.

Each runtime component validates its own option. Constructors register the
hooks of the resource they open: the database package pings the xorm engine,
and messagebus pings and closes the Kafka client.

Shutdown drains only the process's entry points, so that every running
execution reaches its end: committed or rolled back, acknowledged or left
for redelivery. Running work keeps a ctx that shutdown does not cancel and
is bounded by its own timeout; only the end of the stop budget abandons it.
`pkg.Lifecycle` is listed after every bounded-context module and registers
the dispatcher drain, the Kafka consumer, and the ConnectRPC server, in that
order. Fx stops hooks in reverse, so shutdown runs:

1. Entry points stop taking work and finish what is running. The server
   reports `NOT_SERVING` for readiness, keeps serving for
   `connect.shutdown-delay` so load balancers stop routing to it, then closes
   its listener and drains in-flight requests. The consumer polls no more and
   lets its running handlers finish and commit (`Consumer.Shutdown`). A task
   worker or scheduler added later belongs to this layer. Entry points do not
   hand work to each other in-process: Integration Messages go to the broker
   and tasks to their queue.
2. The dispatcher drains. Every entry point above runs use cases that raise
   Domain Events, so it closes last; it keeps handling events, the follow-ups
   its own handlers dispatch included, until it is idle. It is the only step
   that loses work when it runs out of time, and it logs each dropped event
   at Error with its kind and key; the other steps fall back on retries,
   redelivery, or a requeue.
3. The process exits. The Kafka client closes after the dispatcher, because
   it leaves the consumer group and the dispatcher's handlers publish through
   it. Connection pools, such as MySQL's, are not closed: closing adds no
   guarantee to running work, and MySQL rolls back any transaction still open
   when the process exits.

Each step can still use everything stopped after it. Fx runs the stop hooks
one after another on one deadline, `fx.StopTimeout` (30s in `cmd/server`),
so their durations add up: the shutdown delay (5s) plus the request drain,
the consumer's running handlers (`kafka.handler-timeout`, 5s), and the
dispatcher's handlers (`eventbus.handler-timeout`, 10s, more for a chain of
follow-ups). The dispatcher gets whatever remains. The orchestrator must wait
longer than the stop timeout before it kills the process, such as Kubernetes
`terminationGracePeriodSeconds: 40`; a longer delay or handler timeout raises
both.

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
- Runtime tests cover option validation, request logging, and start/stop
  behavior without requiring real MySQL or Kafka.
- The architecture test parses imports and rejects cross-context or
  outward-pointing dependencies in Domain, Application, Infrastructure, and
  Transport, and keeps the private and public RPC contracts in Transport.
