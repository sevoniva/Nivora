FROM golang:1.22 AS builder

ARG GOPROXY=https://proxy.golang.org,direct
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV GOPROXY=${GOPROXY}
WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -o /out/nivora-server ./cmd/nivora-server && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -o /out/nivora-worker ./cmd/nivora-worker && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -o /out/nivora-runner ./cmd/nivora-runner && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -o /out/nivora ./cmd/nivora

FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="Nivora"
LABEL org.opencontainers.image.description="Nivora DevOps delivery control plane"
LABEL org.opencontainers.image.source="https://github.com/sevoniva/nivora"

WORKDIR /workspace

COPY --from=builder /out/nivora /usr/local/bin/nivora
COPY --from=builder /out/nivora-server /usr/local/bin/nivora-server
COPY --from=builder /out/nivora-worker /usr/local/bin/nivora-worker
COPY --from=builder /out/nivora-runner /usr/local/bin/nivora-runner

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/nivora"]
CMD ["server", "--config", "/etc/nivora/server.yaml"]
