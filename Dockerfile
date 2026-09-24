FROM golang:1.26-alpine AS build
ENV RUN apk add --no-cache git
WORKDIR /src
COPY . .
RUN go vet ./... && CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gw ./cmd/gw

FROM scratch
# CA roots: online mode verifies Mojang's session service over TLS.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/gw /gw
USER 1000:1000
EXPOSE 25565
ENTRYPOINT ["/gw"]
