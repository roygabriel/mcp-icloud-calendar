FROM golang:1.25-alpine AS builder

ARG VERSION=dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o mcp-icloud-calendar .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /build/mcp-icloud-calendar /usr/local/bin/mcp-icloud-calendar

USER nonroot:nonroot

ENTRYPOINT ["mcp-icloud-calendar"]
