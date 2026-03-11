FROM node:22-alpine

# Upgrade all packages to latest + install system deps
RUN apk upgrade --no-cache && apk add --no-cache \
    dumb-init \
    curl \
    libc6-compat \
    vips-dev \
    python3 \
    make \
    g++

# Create non-root user
RUN addgroup --system --gid 1001 strapi \
    && adduser --system --uid 1001 strapi

# Set production defaults
ENV NODE_ENV=production

WORKDIR /app

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:1337/_health || exit 1

EXPOSE 1337

ENTRYPOINT ["dumb-init", "--"]
