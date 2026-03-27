FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -o /out/oper-plan-api ./cmd/api

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget \
  && addgroup -S app \
  && adduser -S -G app app \
  && mkdir -p /app/storage \
  && chown -R app:app /app

COPY --from=builder /out/oper-plan-api /app/oper-plan-api

USER app

EXPOSE 8080

CMD ["/app/oper-plan-api"]
