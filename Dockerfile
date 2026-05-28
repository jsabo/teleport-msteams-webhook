FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o teleport-msteams-webhook \
    ./cmd/teleport-msteams-webhook

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /build/teleport-msteams-webhook /teleport-msteams-webhook
ENTRYPOINT ["/teleport-msteams-webhook"]
