FROM node:20-alpine

# System deps for native modules (sharp, etc.)
RUN apk add --no-cache \
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

EXPOSE 3000
