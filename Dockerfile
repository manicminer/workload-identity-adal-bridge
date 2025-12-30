ARG VERSION

FROM golang:1.25 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-X 'main.version=${VERSION}'" -o workload-identity-adal-bridge
CMD ["/workload-identity-adal-bridge"]

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S appuser
RUN adduser -S -G appuser -H -s /sbin/nologin appuser
COPY --from=build --chown=appuser:appuser /app/workload-identity-adal-bridge /workload-identity-adal-bridge
ADD metadata /metadata/
USER appuser
ENTRYPOINT ["/workload-identity-adal-bridge", "serve"]
