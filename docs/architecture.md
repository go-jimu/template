# Architecture Reading Map

This file is a navigation index, not an additional architecture authority.
Use the following order when learning or changing the project:

1. [../AGENTS.md](../AGENTS.md) — repository rules and AI working protocol.
2. [ddd/model.md](ddd/model.md) — business language, authority, invariants, and
   context relationships.
3. [ddd/design.md](ddd/design.md) — accepted Aggregate, CQRS, event/message,
   persistence, error, and runtime decisions.
4. [../README.md](../README.md) — runnable example, commands, and extension
   workflow.

When these sources disagree, `model.md` owns business meaning, `design.md` owns
tactical choices, and `AGENTS.md` owns repository coding conventions. Code and
tests are implementation evidence; they do not silently redefine the model.

## Dependency direction

~~~text
Transport -> Application -> Domain
Infrastructure ---------> Domain/Application ports
Runtime ----------------> bounded-context modules and technical adapters
~~~

Cross-context code never imports another context's Domain or Application
packages. Collaboration uses published Integration Messages or an explicitly
designed read/protocol boundary.

## Example flow

~~~text
Connect request
  -> User Transport
  -> User Application command
  -> User Aggregate
  -> User Repository
  -> UserCreated Domain Event
  -> User Application publication boundary
  -> UserRegisteredV1 Integration Message
  -> Kafka runtime
  -> Notification Message Subscriber
  -> Notification Application command
~~~

## Change routing

| Change intent | First authority to read | Primary implementation owner |
| --- | --- | --- |
| Business term, invariant, lifecycle, or context authority | `docs/ddd/model.md` | Domain model after the fact is accepted |
| Aggregate, transaction, Repository, CQRS, or collaboration shape | `docs/ddd/design.md` | Domain/Application plus adapters |
| ConnectRPC request or response contract | `proto/<context>/v1` | Transport mapping; regenerate `pkg/gen` |
| Published Integration Message contract | `proto/<context>/integration/v1` | Publication boundary and Message Subscriber |
| Command behavior | Model and design sections for the scenario | `application/command` and Domain behavior |
| Product read | Read-model section in design | `application/query` and Infrastructure QueryRepository |
| MySQL mapping or SQL | Aggregate/persistence design | `infrastructure` and `scripts/sql` |
| Broker, server, config, or lifecycle mechanics | Runtime design | `internal/pkg/<component>` |
| Dependency injection and registration | Runtime design | bounded-context module; `cmd` only selects modules |
| Error mapping or logging | Error section in design | Transport/runtime outer boundary |

Do not start from a directory name and infer business semantics. Start from the
accepted fact or use case, then route the implementation to its owner.
