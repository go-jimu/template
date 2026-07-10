# AGENTS.md - Project Context and Coding Guidelines

## Start here

This repository is an executable DDD/Clean Architecture template. Before
editing, read the smallest relevant sources in this order:

1. `docs/ddd/model.md` for business language, invariants, authority, and
   cross-context meaning.
2. `docs/ddd/design.md` for accepted Aggregate, CQRS, event/message,
   persistence, error, and runtime decisions.
3. `docs/architecture.md` for change routing and dependency direction.
4. The neighboring implementation, module wiring, and tests for compatibility
   evidence.

For the current state, `model.md` owns business meaning and `design.md` owns
tactical choices. A new user request may change either, but do not silently
invent or overwrite a rule. Code and tests are evidence, not authority over a
contradictory model or design.

## Project purpose

The example teaches where a decision belongs:

- User is a real Aggregate with credential behavior.
- Commands and queries use different Repository shapes.
- ConnectRPC is an inbound Transport adapter.
- Domain Events and Integration Messages have different language and scope.
- Kafka, xorm, servers, configuration, and lifecycle stay outside Domain.
- Fx assembles components without turning `main` into a service locator.

Do not introduce Gin, GORM, a second DI container, a local replacement for an
adopted go-jimu event/message interface, or another library serving the same
role.

## Repository map

~~~text
cmd/                               process composition only
configs/                           component-owned runtime options
docs/architecture.md               architecture navigation and change routing
docs/ddd/model.md                  business semantic authority
docs/ddd/design.md                 tactical design authority
internal/business/<context>/
  domain/                          Aggregate, behavior, errors, Domain Events, write Repository
  application/command/             write orchestration
  application/query/               read DTOs and QueryRepository
  application/eventhandler/        same-context Domain Event reactions/publication boundary
  transport/connectrpc/            inbound API mapping
  transport/messagesubscriber/     inbound Integration Message mapping
  infrastructure/                  xorm, DO conversion, ACLs, outbound adapters
  <context>.go                     bounded-context Fx module
internal/pkg/                      shared technical runtime and lifecycle
proto/                             source API and published message contracts
pkg/gen/                           generated code; never edit manually
scripts/sql/                       ordered MySQL schema scripts
~~~

## Dependency rules

Dependencies point inward:

~~~text
Transport -> Application -> Domain
Infrastructure ---------> Domain/Application ports
Runtime ----------------> bounded-context modules and adapters
~~~

A bounded context never imports another context's internal Domain,
Application, Repository, or Infrastructure packages. Cross-context interaction
uses an Integration Message or another explicitly accepted published boundary.

### Domain

- Aggregates own invariants and expose behavior methods.
- Factories create valid objects and record Domain Events after state changes.
- Domain never persists, dispatches, publishes, logs, or owns goroutines.
- A write Repository represents one Aggregate collection and normally exposes
  only `Get` and `Save`.
- Domain never imports generated protocol, ConnectRPC, ORM/database, broker,
  runtime, `internal/pkg`, Infrastructure, or another bounded context.
- General-purpose libraries such as event collections, UUID, crypto,
  validation, and oops are allowed when they do not represent an external
  system.
- User fields are exported intentionally so Infrastructure can perform explicit
  DO-to-Aggregate conversion. This is not permission for Application or
  Transport to mutate state directly.
- Do not add `Restore`, `Rehydrate`, reflection, unsafe setters, or similar
  construction hooks solely to work around persistence conversion.

### Application

- Commands orchestrate: load/create one Aggregate, invoke Domain behavior,
  save, then drain and dispatch Domain Events.
- Queries return read DTOs through an Application-owned QueryRepository and do
  not load Aggregates for product views.
- Command/query inputs are protocol independent.
- Application does not import Infrastructure, Transport, database sessions,
  Kafka clients, or `internal/pkg` runtime packages.
- Domain Event Handlers consume same-context Domain Events. A selected event may
  be translated into published language and sent through
  `message.Publisher`.
- Every command, query, and Domain Event Handler emits one completion record per
  call. Returning failures may log `outcome=failed` at info level but must not
  print the error when an outer boundary will do so.

### Transport

`transport` means inbound adapters in this repository; it does not mean
`http.RoundTripper`, a network client, or a place for runtime server mechanics.

- Connect handlers validate/map protocol input, delegate once, map output, and
  translate errors.
- Message Subscribers validate/map an Integration Message and delegate once to
  an Application command.
- Message Subscriber failures are returned without an inner error log; the
  Kafka observer is the outer boundary.
- Transport does not call Repositories, mutate Aggregates, control
  transactions, dispatch Domain Events, or own broker/server lifecycle.
- Use the package name `messagesubscriber` for inbound message consumers. Do
  not use an ambiguous `messagehandler` package.

### Infrastructure

- xorm Data Objects are separate from Domain objects.
- DO-to-Aggregate and Aggregate-to-DO conversion is explicit in
  `infrastructure/converter.go`.
- Repository implementations satisfy Domain/Application interfaces with
  compile-time assertions.
- New User rows store version `1`. Updates compare ID/version and increment the
  version atomically.
- Infrastructure owns database, broker, cache, SDK, ACL, retry, topology, and
  other external mechanics.
- Infrastructure never drains or dispatches Domain Events.

### Runtime and composition

- Each component owns its Option/Config validation and resource lifecycle.
- Each bounded context exposes one Fx module for constructors and
  registrations.
- `cmd/main.go` loads/supplies aggregate configuration, selects modules,
  configures process timeouts, and runs Fx.
- Do not move individual component validation, handler registration, client
  construction, or helper functions into `main`.
- Every goroutine must have an Fx lifecycle stop path.
- Shutdown stops ingress and background work before the clients they still
  need.

## Events and messages

| Concept | Scope | Owner |
| --- | --- | --- |
| Domain Event | Same bounded context, internal language | Domain records; Application handles |
| Integration Message | Cross-context, stable published language | Proto contract and publication boundary |
| Message Subscriber | Inbound Integration Message adapter | Receiving context Transport |
| Kafka publisher/consumer | Technical delivery mechanism | `internal/pkg/messagebus` runtime |

Never publish a Domain Event struct directly as an Integration Message.
`UserCreated` and `UserRegisteredV1` are intentionally different types.

The template uses `DropAndCommitPolicy` and intentionally omits task processors,
outbox, retry, DLQ, durable notification state, and idempotency storage. Add
those only after an explicit production reliability decision.

## Errors and logging

- Use `oops.Wrap(err)` when an external database, broker, network, SDK, or
  library error enters code we control.
- Use `oops.With(...).Wrap(err)` at internal layer boundaries to add only useful
  fields owned by that layer.
- Preserve Domain error identity for `errors.Is` and protocol mapping.
- Never attach passwords, tokens, full DSNs, or raw secrets.
- Print an error exactly once at the outermost execution boundary and always
  pass it through `sloghelper.Error(err)`.
- ConnectRPC middleware owns RPC error logs; the Kafka observer owns consumed
  message errors; Domain Event Handlers log terminal errors because their
  handler contract cannot return one.
- Do not put normal completion logging in `defer`. A small recovery `defer` is
  acceptable when the log happens afterward.

## Change routing

| Intent | Read first | Change primarily |
| --- | --- | --- |
| Business rule or invariant | `model.md` | Domain and Domain tests |
| Command orchestration | Model scenario + `design.md` | `application/command` |
| Product read | Read-model design | `application/query` + QueryRepository adapter |
| Connect API | API Proto | `transport/connectrpc`; regenerate |
| Published message | Model relationship + integration Proto | publication boundary + Subscriber |
| MySQL mapping | Aggregate/persistence design | `infrastructure` + `scripts/sql` |
| Kafka/server/config/lifecycle | Runtime design | `internal/pkg/<component>` |
| Wiring | Runtime design | bounded-context module; root `cmd` only selects |

## Do not infer

The current example deliberately has no email uniqueness policy, account
activation, deletion lifecycle, Notification Aggregate, real notification
provider, or reliable-delivery workflow. A technical `deleted_at` column does
not create a Domain deletion lifecycle. Do not add common account-system
features unless the user asks for them.

## Workflow

1. Classify the request as business meaning, tactical design, or mechanical
   implementation.
2. Read the relevant model/design sections and the complete neighboring flow.
3. Inspect Proto, generated artifacts, SQL, runtime registration, and tests
   when the change touches those surfaces.
4. Make the smallest layer-owned change; preserve unrelated worktree changes.
5. Update model/design only when the accepted meaning or tactical choice
   changes.
6. Add behavior evidence at the lowest real boundary.
7. Run the relevant checks, normally `make verify` for a complete change.

Useful commands:

~~~bash
make generate   # lint and regenerate Proto contracts
make test       # race-enabled Go tests with coverage
make vet        # Go static analysis
make verify     # generate + test + vet
make server     # run the service
make client     # invoke the example client
~~~

Public Go identifiers require comments. Adapters require compile-time interface
assertions. Generated files under `pkg/gen` are never edited manually.
