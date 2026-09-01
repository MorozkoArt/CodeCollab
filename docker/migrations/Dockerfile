FROM ghcr.io/kukymbr/goose-docker:3.24.1

WORKDIR /app

COPY backend/db/migrations ./db/migrations

ENTRYPOINT ["goose", "-dir", "/app/db/migrations"]
CMD ["up"]