FROM golang:1.18-alpine3.16 as builder
WORKDIR /go/src/app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY ./cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64  \
    go build -ldflags '-w -s -extldflags "-static"' -o /app ./cmd/mk_daemon

FROM scratch
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
