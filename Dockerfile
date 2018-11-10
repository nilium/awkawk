FROM golang:1.11.2 as build

ENV CGO_ENABLED=0

COPY . /build
WORKDIR /build
RUN go build

FROM busybox:1.29.3-musl

ENV LISTEN="0.0.0.0:8080"
EXPOSE 8080

COPY --from=build /build/awkawk /usr/local/bin/awkawk
ENTRYPOINT ["/usr/local/bin/awkawk", "-logtostderr"]
