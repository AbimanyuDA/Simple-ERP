FROM golang:1.26.6-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate
RUN CGO_ENABLED=0 go build -trimpath -o /out/bootstrap-admin ./cmd/bootstrap-admin

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
USER app
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/migrate /usr/local/bin/migrate
COPY --from=build /out/bootstrap-admin /usr/local/bin/bootstrap-admin
COPY --from=build /src/migrations ./migrations
EXPOSE 8080
ENTRYPOINT ["api"]
