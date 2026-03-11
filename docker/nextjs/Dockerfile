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
RUN addgroup --system --gid 1001 nodejs \
    && adduser --system --uid 1001 nextjs

# Set production defaults
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

WORKDIR /app

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:3000/ || exit 1

EXPOSE 3000

ENTRYPOINT ["dumb-init", "--"]
