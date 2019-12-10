# Step 1: build executable binary
FROM golang:1.13.5-alpine3.10@sha256:ddf407284440a94889dc139bbe1be1daa19d99e166d6b1f2dfc6919846810b4e as builder

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