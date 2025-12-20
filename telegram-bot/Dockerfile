FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/bot ./cmd/bot

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/bot /app/bot
ENTRYPOINT ["/app/bot"]
