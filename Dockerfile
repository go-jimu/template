# syntax=docker/dockerfile:1

# The builder runs on the build host's platform and cross-compiles for the
# target, so a multi-platform build needs no emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-trixie AS builder
ARG TARGETOS TARGETARCH
WORKDIR /src
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/template ./cmd/server

# distroless/static carries CA certificates and tzdata, with no shell or
# package manager. Its nonroot user is UID 65532, numeric so Kubernetes
# runAsNonRoot can verify it.
FROM gcr.io/distroless/static-debian13:nonroot
WORKDIR /app
COPY --from=builder /out/template /app/template
# The server reads ./configs/server/defaults.yml; a deployment mounts its own
# over /app/configs/server.
COPY configs/server/defaults.yml /app/configs/server/defaults.yml
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/app/template"]
