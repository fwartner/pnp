FROM node:20-alpine

# System deps for native modules (sharp, better-sqlite3, etc.)
RUN apk add --no-cache \
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

EXPOSE 1337
