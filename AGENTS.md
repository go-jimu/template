# AGENTS.md - Project Context & Coding Guidelines

**Purpose:** This document provides high-level context, architectural decisions, and coding conventions for AI Agents interacting with this codebase. Read this to understand the "vibe" and style of the project before writing code.

## 1. Project Overview

*   **Name:** Go-Jimu Template
*   **Architecture:** Domain-Driven Design (DDD) + Clean Architecture.
*   **Key Paradigm:** Dependency Injection via **Uber fx**.
*   **Communication:** gRPC & HTTP (via **ConnectRPC**).
*   **Database:** MySQL (via **xorm**).

## 2. Directory Structure & Responsibilities

The project follows a strict separation of concerns.

```text
/cmd                -> Application entry points (main.go). Wires modules.
/configs            -> Configuration files (YAML).
/internal
  /business         -> DDD Bounded Contexts (e.g., 'user', 'order').
    /<domain>
      /domain       -> PURE. Entities, Value Objects, Repository Interfaces. NO external imports.
      /application  -> ORCHESTRATION. Use Cases (Commands/Queries), DTO assembling.
      /infrastructure -> ADAPTERS. Repository implementations, external API clients.
      <domain>.go   -> FX MODULE. Wires the layers together using fx.Module.
  /pkg              -> Shared infrastructure (DB drivers, Loggers, HTTP server wrappers).
/pkg/gen            -> GENERATED CODE. Do not edit manually. (Protobuf/Connect definitions).
/proto              -> API Contracts (Protobuf files).
```

## 3. Technology Stack & Libraries

Use these libraries. **Do not introduce alternatives** (e.g., do not use GORM, Gin, or standard `net/http` mux).

*   **DI:** `go.uber.org/fx`
*   **RPC/HTTP:** `connectrpc.com/connect`
*   **Router:** `github.com/go-chi/chi/v5` (wrapped in internal pkg)
*   **ORM:** `xorm.io/xorm`
*   **Validation:** `github.com/go-playground/validator/v10`
*   **Logging:** `log/slog` (Standard lib) with `samber/oops` for errors.
*   **Protobuf:** `buf` (tooling), `google.golang.org/protobuf`.

## 4. Implementation Patterns (The "Vibe")

### A. Dependency Injection (fx)
Every logical component must be provided via `fx`.
*   **Constructors:** Return interfaces where possible. `func NewRepository(...) domain.Repository`.
*   **Modules:** Each business domain has a `Module` variable in `internal/business/<name>/<name>.go`.
*   **Lifecycle:** Use `fx.Lifecycle` only if background workers/hooks are needed.

### B. Domain Layer (Pure)
*   **Entities:** Rich structs with methods.
*   **Validation:** Use `validate` tags on structs. Call `Validate()` in factory methods (`NewUser`).
*   **Events:** Use `mediator.EventCollection` to store domain events inside the aggregate root.

### C. Application Layer (Use Cases)
*   **Split:** Separate `Commands` (Write) and `Queries` (Read).
*   **Handlers:** Thin wrappers around domain logic.
*   **DTOs:** Convert Domain Entities to Proto responses here (Assemblers).

### D. Infrastructure Layer (Repositories)
*   **XORM:** Use `xorm.Engine`.
*   **DO (Data Objects):** Separate struct for Database Table (`UserDO`) vs Domain Entity (`User`).
*   **Conversion:** Explicit converters between DO and Entity.

### E. API Layer (ConnectRPC)
*   **Definition:** Defined in `proto/`.
*   **Handlers:** Implemented in `internal/business/<domain>/<domain>.go` or separate handler files if complex.
*   **Registration:** Registered via `fx.Invoke` in the module definition.

## 5. Workflow: Adding a New Feature

1.  **Proto:** Define the RPC in `proto/<domain>/<api_version>/<file>.proto`. Run `buf generate` (or `make tools` then `buf generate`).
2.  **Domain:** Define the Entity and Repository Interface in `internal/business/<domain>/domain/`.
3.  **Infrastructure:** Implement the Repository in `internal/business/<domain>/infrastructure/`.
4.  **Application:** Create the Handler/UseCase in `internal/business/<domain>/application/`.
5.  **Wiring:** Update `internal/business/<domain>/<domain>.go` to `fx.Provide` the new components and `fx.Invoke` the registration.

## 6. Coding Conventions

*   **Error Handling:** Use `oops.With(...).Wrap(err)` to add context. Return errors explicitly.
*   **Logging:** Use `sloghelper.FromContext(ctx)` to get a logger with trace IDs.
*   **Comments:** Public functions must have comments (needed for linter).
*   **Interface Compliance:** Always verify interface implementation: `var _ domain.Repository = (*userRepository)(nil)`.
