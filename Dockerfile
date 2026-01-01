FROM docker.io/golang:1.24.0 AS builder
WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o /bikinota .

FROM gcr.io/distroless/static
WORKDIR /app
COPY --from=builder /bikinota /bikinota

USER 65532:65532

ENTRYPOINT ["/bikinota"]
