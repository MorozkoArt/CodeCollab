FROM ghcr.io/kukymbr/goose-docker:3.24.1

WORKDIR /app

COPY db/migrations ./db/migrations

ENTRYPOINT ["goose", "-dir", "/app/db/migrations"]
CMD ["up"]