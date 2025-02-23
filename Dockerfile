FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o postgrest-cache ./main/main.go
FROM scratch

COPY --from=builder /app/postgrest-cache /postgrest-cache

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080



ENV SUPA_URL="http://localhost:3000"
ENV PORT="8080"
ENV REDIS_ADDR="localhost:6379"
ENV REDIS_PASSWORD=""
ENV REDIS_DB="0"
ENV CACHE_TTL_MINUTES="5"
ENV CACHE_TABLES="*"

ENTRYPOINT ["/postgrest-cache"]