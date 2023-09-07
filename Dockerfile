FROM golang:1.20-alpine AS build-stage

WORKDIR /SheduleBot

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /app

FROM alpine AS build-release-stage

WORKDIR /

COPY --from=build-stage /app /app

COPY --from=build-stage /SheduleBot/config.yaml .

ENTRYPOINT ["/app"]
