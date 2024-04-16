FROM golang:1.22.1-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go generate

RUN CGO_ENABLED=0 GOOS=linux go build -o cdn .

FROM alpine:edge AS release-stage

WORKDIR /app

COPY --from=build /app/cdn .

CMD ["/app/cdn"]
