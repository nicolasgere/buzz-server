FROM golang:1.12-alpine as build

RUN apk update
RUN apk add git

WORKDIR /build

COPY . .

RUN go build -o app main.go
RUN go build  -o migration migrate-bigtable.go


FROM alpine:3.7

RUN  apk update && \
     apk add libc6-compat && \
     apk add ca-certificates

COPY --from=build /build/app /usr/local/bin/app
COPY --from=build /build/migration /usr/local/bin/migration

ENTRYPOINT ["/usr/local/bin/app"]