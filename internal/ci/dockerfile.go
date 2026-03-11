package ci

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fwartner/pnp/internal/config"
)

// --- Laravel: nginx + PHP-FPM (standard, no Octane) ---

const laravelDockerfile = `# ---- Composer dependencies ----
FROM composer:2 AS composer-deps
WORKDIR /app
COPY composer.json composer.lock ./
RUN --mount=type=secret,id=composer_auth,env=COMPOSER_AUTH \
    composer install --no-dev --no-scripts --no-autoloader --prefer-dist --ignore-platform-reqs

# ---- Node build (frontend assets) ----
FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# ---- Production image ----
FROM ghcr.io/fwartner/pnp/laravel-fpm:latest

WORKDIR /app

# Copy application
COPY --from=composer-deps /app/vendor ./vendor
COPY . .

# Copy built frontend assets
COPY --from=node-build /app/public/build ./public/build

# Composer autoload
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer dump-autoload --optimize --ignore-platform-reqs && rm /usr/bin/composer

# Storage & cache directories with correct permissions
RUN mkdir -p storage/framework/{sessions,views,cache} \
    storage/logs bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache \
    && chmod -R 775 storage bootstrap/cache

EXPOSE 80

CMD ["supervisord", "-c", "/etc/supervisord.conf"]
`

// --- Laravel: FrankenPHP (Octane) ---

const laravelOctaneFrankenphpDockerfile = `# ---- Composer dependencies ----
FROM composer:2 AS composer-deps
WORKDIR /app
COPY composer.json composer.lock ./
RUN --mount=type=secret,id=composer_auth,env=COMPOSER_AUTH \
    composer install --no-dev --no-scripts --no-autoloader --prefer-dist --ignore-platform-reqs

# ---- Node build (frontend assets) ----
FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# ---- Production image ----
FROM ghcr.io/fwartner/pnp/laravel-frankenphp:latest

WORKDIR /app

COPY --from=composer-deps /app/vendor ./vendor
COPY . .
COPY --from=node-build /app/public/build ./public/build

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer dump-autoload --optimize --ignore-platform-reqs && rm /usr/bin/composer

RUN mkdir -p storage/framework/{sessions,views,cache} \
    storage/logs bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache \
    && chmod -R 775 storage bootstrap/cache

EXPOSE 8000

CMD ["php", "artisan", "octane:start", "--server=frankenphp", "--host=0.0.0.0", "--port=8000"]
`

// --- Laravel: Swoole (Octane) ---

const laravelOctaneSwooleDockerfile = `# ---- Composer dependencies ----
FROM composer:2 AS composer-deps
WORKDIR /app
COPY composer.json composer.lock ./
RUN --mount=type=secret,id=composer_auth,env=COMPOSER_AUTH \
    composer install --no-dev --no-scripts --no-autoloader --prefer-dist --ignore-platform-reqs

# ---- Node build (frontend assets) ----
FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# ---- Production image ----
FROM ghcr.io/fwartner/pnp/laravel-swoole:latest

WORKDIR /app

COPY --from=composer-deps /app/vendor ./vendor
COPY . .
COPY --from=node-build /app/public/build ./public/build

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer dump-autoload --optimize --ignore-platform-reqs && rm /usr/bin/composer

RUN mkdir -p storage/framework/{sessions,views,cache} \
    storage/logs bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache \
    && chmod -R 775 storage bootstrap/cache

EXPOSE 8000

CMD ["php", "artisan", "octane:start", "--server=swoole", "--host=0.0.0.0", "--port=8000"]
`

// --- Laravel: RoadRunner (Octane) ---

const laravelOctaneRoadrunnerDockerfile = `# ---- Composer dependencies ----
FROM composer:2 AS composer-deps
WORKDIR /app
COPY composer.json composer.lock ./
RUN --mount=type=secret,id=composer_auth,env=COMPOSER_AUTH \
    composer install --no-dev --no-scripts --no-autoloader --prefer-dist --ignore-platform-reqs

# ---- Node build (frontend assets) ----
FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# ---- Production image ----
FROM ghcr.io/fwartner/pnp/laravel-roadrunner:latest

WORKDIR /app

COPY --from=composer-deps /app/vendor ./vendor
COPY . .
COPY --from=node-build /app/public/build ./public/build

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer dump-autoload --optimize --ignore-platform-reqs && rm /usr/bin/composer

RUN mkdir -p storage/framework/{sessions,views,cache} \
    storage/logs bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache \
    && chmod -R 775 storage bootstrap/cache

EXPOSE 8000

CMD ["php", "artisan", "octane:start", "--server=roadrunner", "--host=0.0.0.0", "--port=8000"]
`

// --- Next.js ---

const nextjsDockerfile = `FROM ghcr.io/fwartner/pnp/nextjs:latest AS base

FROM base AS deps
WORKDIR /app
COPY package.json package-lock.json* yarn.lock* pnpm-lock.yaml* ./
RUN \
    if [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm install --frozen-lockfile; \
    elif [ -f package-lock.json ]; then npm ci; \
    else npm install; \
    fi

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

RUN npm run build

FROM base AS production
WORKDIR /app

COPY --from=builder /app/public ./public

# Standalone output — set output: 'standalone' in next.config.js
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

CMD ["node", "server.js"]
`

// --- Strapi ---

const strapiDockerfile = `FROM ghcr.io/fwartner/pnp/strapi:latest AS base

FROM base AS deps
WORKDIR /app
COPY package.json package-lock.json* yarn.lock* pnpm-lock.yaml* ./
RUN \
    if [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm install --frozen-lockfile; \
    elif [ -f package-lock.json ]; then npm ci; \
    else npm install; \
    fi

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM base AS production
WORKDIR /app

COPY --from=builder --chown=strapi:strapi /app ./

USER strapi

CMD ["npm", "run", "start"]
`

// --- .dockerignore ---

const laravelDockerignore = `node_modules
vendor
.git
.github
.env
.env.*
.cluster.yaml
storage/logs/*
storage/framework/cache/*
storage/framework/sessions/*
storage/framework/views/*
bootstrap/cache/*
tests
phpunit.xml
.phpunit.result.cache
docker-compose*.yml
`

const nodeDockerignore = `node_modules
.git
.github
.env
.env.*
.cluster.yaml
.next
out
dist
build
coverage
docker-compose*.yml
`

// GenerateDockerfile creates a Dockerfile in the project directory based on the project type.
func GenerateDockerfile(projectType string, octaneCfg config.OctaneConfig, projectDir string) error {
	var content string
	var ignoreContent string

	switch projectType {
	case "laravel-web", "laravel-api":
		ignoreContent = laravelDockerignore
		if octaneCfg.Enabled {
			switch octaneCfg.Server {
			case "swoole":
				content = laravelOctaneSwooleDockerfile
			case "roadrunner":
				content = laravelOctaneRoadrunnerDockerfile
			default:
				content = laravelOctaneFrankenphpDockerfile
			}
		} else {
			content = laravelDockerfile
		}
	case "nextjs-fullstack", "nextjs-static":
		content = nextjsDockerfile
		ignoreContent = nodeDockerignore
	case "strapi":
		content = strapiDockerfile
		ignoreContent = nodeDockerignore
	default:
		return fmt.Errorf("unsupported project type for Dockerfile: %s", projectType)
	}

	// Write Dockerfile
	outPath := filepath.Join(projectDir, "Dockerfile")
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Write .dockerignore if it doesn't exist
	ignorePath := filepath.Join(projectDir, ".dockerignore")
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(ignorePath, []byte(ignoreContent), 0o644); err != nil {
			return fmt.Errorf("failed to write .dockerignore: %w", err)
		}
	}

	return nil
}
