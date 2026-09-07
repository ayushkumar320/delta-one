FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/auth ./services/auth/cmd/server \
 && CGO_ENABLED=0 go build -o /out/catalog ./services/catalog/cmd/server \
 && CGO_ENABLED=0 go build -o /out/booking ./services/booking/cmd/server \
 && CGO_ENABLED=0 go build -o /out/payment ./services/payment/cmd/server \
 && CGO_ENABLED=0 go build -o /out/notification ./services/notification/cmd/server \
 && CGO_ENABLED=0 go build -o /out/gateway ./services/gateway/cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out /usr/local/bin
USER nobody
