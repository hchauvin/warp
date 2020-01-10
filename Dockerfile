# Step 1: build executable binary
FROM golang:1.13.6-alpine3.10@sha256:13a9dc7915b6899edcc21a7222675146d568fd1ec99d65d6b7fd1d42e355065f as builder

RUN adduser -D -g '' appuser

WORKDIR /project
COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o bin/warp ./cmd/warp

# Step 1: build end product
FROM gcr.io/distroless/static:latest@sha256:c6d5981545ce1406d33e61434c61e9452dad93ecd8397c41e89036ef977a88f4

COPY --from=builder /project/bin/warp /warp
COPY --from=builder /etc/passwd /etc/passwd

USER appuser

ENTRYPOINT ["/warp"]