FROM golang:1.22 AS builder

ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/nivora-server ./cmd/nivora-server && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/nivora-worker ./cmd/nivora-worker && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/nivora-runner ./cmd/nivora-runner && \
    CGO_ENABLED=0 GOOS=linux go build -o /out/nivora ./cmd/nivora

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

ARG NIVORA_BINARY=nivora-server
COPY --from=builder /out/${NIVORA_BINARY} /nivora

ENTRYPOINT ["/nivora"]
CMD []
