FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/runner ./cmd/runner

FROM alpine:3.20
RUN apk add --no-cache ca-certificates postgresql-client
WORKDIR /app
COPY --from=build /out/runner /app/runner
ENTRYPOINT ["/app/runner"]

