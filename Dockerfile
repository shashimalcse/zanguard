FROM golang:1.23-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/zanguard ./cmd/server/main.go

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /out/zanguard /app/zanguard

EXPOSE 1997

ENTRYPOINT ["/app/zanguard"]
