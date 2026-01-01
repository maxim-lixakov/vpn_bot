FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app ./cmd/app

FROM alpine:3.20
RUN apk add --no-cache ca-certificates postgresql-client
WORKDIR /app
COPY --from=build /out/app /app/app
ENTRYPOINT ["/app/app"]
