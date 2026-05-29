# Go-Jimu Template

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A production-ready GitHub template for Web development in Go, implementing Domain-Driven Design (DDD) principles and Clean Architecture.

## Features

- **Domain-Driven Design (DDD)**: Structured with clear separation of concerns (Domain, Application, Infrastructure, Interfaces).
- **Clean Architecture**: Dependencies point inwards, ensuring the core logic is independent of external frameworks.
- **Dependency Injection**: Uses [Uber fx](https://github.com/uber-go/fx) for managing application lifecycle and dependencies.
- **gRPC & ConnectRPC**: Built-in support for gRPC and [ConnectRPC](https://connectrpc.com/) for type-safe APIs.
- **Database**: Integration with [xorm](https://xorm.io/) and MySQL.
- **Validation**: Struct validation using [go-playground/validator](https://github.com/go-playground/validator).
- **Logging**: Structured logging with `slog`.
- **Configuration**: Flexible configuration management.

## Project Structure

```
├── cmd/                # Application entry points
├── configs/            # Configuration files
├── internal/
│   ├── business/       # Business modules (DDD context boundaries)
│   │   └── user/       # Example 'user' domain
│   │       ├── domain/         # Domain entities, value objects, repository interfaces
│   │       ├── application/    # Application services (Use Cases), Commands, Queries
│   │       ├── infrastructure/ # Implementation of repositories, external adapters
│   │       └── user.go         # Module wiring
│   └── pkg/            # Shared infrastructure code (DB, HTTP, Log, etc.)
├── pkg/                # Public packages
│   └── gen/            # Generated code (Protobuf, etc.)
├── proto/              # Protocol Buffer definitions
├── scripts/            # Database initialization scripts, etc.
└── Makefile            # Build and utility commands
```

## Prerequisites

- Go 1.24+
- MySQL 8.0+
- Make

## Getting Started

### 1. Install Tools

Install the necessary development tools (Buf, Protoc plugins):

```bash
make tools
```

### 2. Database Setup

Ensure you have a MySQL instance running. Create the database and tables using the script provided:

```bash
mysql -u root -p < scripts/sql/init.sql
```

Update `configs/default.yml` with your database credentials if necessary.

### 3. Run the Application

Start the server:

```bash
make server
```

The application will start with default configurations.

### 4. Running Tests

Run unit tests:

```bash
make unittest
```

## Configuration

The application uses a configuration file located at `configs/default.yml`. You can override these settings via environment variables or by providing a different configuration file.

## Development

### Adding a New Module

1.  Create a new directory in `internal/business/<module_name>`.
2.  Define your Domain Model in `domain/`.
3.  Define Repository interfaces in `domain/repository.go`.
4.  Implement Use Cases in `application/`.
5.  Implement Repositories in `infrastructure/`.
6.  Wire everything up in `internal/business/<module_name>/<module_name>.go` using `fx.Module`.
7.  Register the new module in `cmd/main.go`.

### API Definition

1.  Define your API in `proto/<module>/v1/*.proto`.
2.  Generate Go code:
    ```bash
    buf generate
    ```
    (Ensure you have `buf` installed and configured).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.