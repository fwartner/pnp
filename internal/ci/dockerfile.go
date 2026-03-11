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
ARG COMPOSER_AUTH
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist --ignore-platform-reqs

# ---- Node build (frontend assets) ----
FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# ---- Production image ----
FROM php:8.3-fpm-alpine

# System deps
RUN apk add --no-cache \
    nginx \
    supervisor \
    postgresql-dev \
    libzip-dev \
    oniguruma-dev \
    icu-dev \
    freetype-dev \
    libjpeg-turbo-dev \
    libpng-dev \
    $PHPIZE_DEPS

# PHP extensions
RUN docker-php-ext-configure gd --with-freetype --with-jpeg \
    && docker-php-ext-install \
        pdo_pgsql \
        zip \
        mbstring \
        opcache \
        pcntl \
        intl \
        gd \
        bcmath \
    && pecl install redis \
    && docker-php-ext-enable redis \
    && apk del $PHPIZE_DEPS

# PHP production settings
RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini" \
    && echo "opcache.enable=1" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.memory_consumption=128" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.validate_timestamps=0" >> "$PHP_INI_DIR/conf.d/opcache.ini"

WORKDIR /app

# Copy application
COPY --from=composer-deps /app/vendor ./vendor
COPY . .

# Copy built frontend assets
COPY --from=node-build /app/public/build ./public/build

# Composer autoload
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer dump-autoload --optimize && rm /usr/bin/composer

# Storage & cache directories with correct permissions
RUN mkdir -p storage/framework/{sessions,views,cache} \
    storage/logs bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache \
    && chmod -R 775 storage bootstrap/cache

# Nginx config
RUN echo 'server { \
    listen 80; \
    server_name _; \
    root /app/public; \
    index index.php; \
    client_max_body_size 64M; \
    location / { \
        try_files $uri $uri/ /index.php?$query_string; \
    } \
    location ~ \.php$ { \
        fastcgi_pass 127.0.0.1:9000; \
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name; \
        include fastcgi_params; \
        fastcgi_buffering off; \
    } \
    location ~ /\.(?!well-known) { \
        deny all; \
    } \
}' > /etc/nginx/http.d/default.conf

# Supervisord config (nginx + php-fpm)
RUN echo '[supervisord]' > /etc/supervisord.conf \
    && echo 'nodaemon=true' >> /etc/supervisord.conf \
    && echo 'logfile=/dev/null' >> /etc/supervisord.conf \
    && echo 'logfile_maxbytes=0' >> /etc/supervisord.conf \
    && echo '[program:php-fpm]' >> /etc/supervisord.conf \
    && echo 'command=php-fpm --nodaemonize' >> /etc/supervisord.conf \
    && echo 'stdout_logfile=/dev/stdout' >> /etc/supervisord.conf \
    && echo 'stdout_logfile_maxbytes=0' >> /etc/supervisord.conf \
    && echo 'stderr_logfile=/dev/stderr' >> /etc/supervisord.conf \
    && echo 'stderr_logfile_maxbytes=0' >> /etc/supervisord.conf \
    && echo '[program:nginx]' >> /etc/supervisord.conf \
    && echo 'command=nginx -g "daemon off;"' >> /etc/supervisord.conf \
    && echo 'stdout_logfile=/dev/stdout' >> /etc/supervisord.conf \
    && echo 'stdout_logfile_maxbytes=0' >> /etc/supervisord.conf \
    && echo 'stderr_logfile=/dev/stderr' >> /etc/supervisord.conf \
    && echo 'stderr_logfile_maxbytes=0' >> /etc/supervisord.conf

EXPOSE 80

CMD ["supervisord", "-c", "/etc/supervisord.conf"]
`

// --- Laravel: FrankenPHP (Octane) ---

const laravelOctaneFrankenphpDockerfile = `# ---- Composer dependencies ----
FROM composer:2 AS composer-deps
WORKDIR /app
COPY composer.json composer.lock ./
ARG COMPOSER_AUTH
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist --ignore-platform-reqs

# ---- Node build (frontend assets) ----
FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# ---- Production image ----
FROM dunglas/frankenphp:latest-php8.3-alpine

RUN install-php-extensions \
    pdo_pgsql \
    zip \
    mbstring \
    opcache \
    pcntl \
    intl \
    gd \
    bcmath \
    redis

# PHP production settings
RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini" \
    && echo "opcache.enable=1" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.memory_consumption=128" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.validate_timestamps=0" >> "$PHP_INI_DIR/conf.d/opcache.ini"

WORKDIR /app

COPY --from=composer-deps /app/vendor ./vendor
COPY . .
COPY --from=node-build /app/public/build ./public/build

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer dump-autoload --optimize && rm /usr/bin/composer

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
ARG COMPOSER_AUTH
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist --ignore-platform-reqs

# ---- Node build (frontend assets) ----
FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# ---- Production image ----
FROM php:8.3-cli-alpine

RUN apk add --no-cache \
    postgresql-dev \
    libzip-dev \
    oniguruma-dev \
    icu-dev \
    freetype-dev \
    libjpeg-turbo-dev \
    libpng-dev \
    linux-headers \
    $PHPIZE_DEPS \
    && docker-php-ext-configure gd --with-freetype --with-jpeg \
    && docker-php-ext-install \
        pdo_pgsql zip mbstring opcache pcntl intl gd bcmath \
    && pecl install swoole redis \
    && docker-php-ext-enable swoole redis \
    && apk del $PHPIZE_DEPS

# PHP production settings
RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini" \
    && echo "opcache.enable=1" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.memory_consumption=128" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.validate_timestamps=0" >> "$PHP_INI_DIR/conf.d/opcache.ini"

WORKDIR /app

COPY --from=composer-deps /app/vendor ./vendor
COPY . .
COPY --from=node-build /app/public/build ./public/build

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer dump-autoload --optimize && rm /usr/bin/composer

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
ARG COMPOSER_AUTH
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist --ignore-platform-reqs

# ---- Node build (frontend assets) ----
FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --ignore-scripts
COPY . .
RUN npm run build

# ---- Production image ----
FROM php:8.3-cli-alpine

RUN apk add --no-cache \
    postgresql-dev \
    libzip-dev \
    oniguruma-dev \
    icu-dev \
    freetype-dev \
    libjpeg-turbo-dev \
    libpng-dev \
    $PHPIZE_DEPS \
    && docker-php-ext-configure gd --with-freetype --with-jpeg \
    && docker-php-ext-install \
        pdo_pgsql zip mbstring opcache pcntl intl gd bcmath sockets \
    && pecl install redis \
    && docker-php-ext-enable redis \
    && apk del $PHPIZE_DEPS

COPY --from=ghcr.io/roadrunner-server/roadrunner:latest /usr/bin/rr /usr/bin/rr

# PHP production settings
RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini" \
    && echo "opcache.enable=1" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.memory_consumption=128" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.validate_timestamps=0" >> "$PHP_INI_DIR/conf.d/opcache.ini"

WORKDIR /app

COPY --from=composer-deps /app/vendor ./vendor
COPY . .
COPY --from=node-build /app/public/build ./public/build

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
RUN composer dump-autoload --optimize && rm /usr/bin/composer

RUN mkdir -p storage/framework/{sessions,views,cache} \
    storage/logs bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache \
    && chmod -R 775 storage bootstrap/cache

EXPOSE 8000

CMD ["php", "artisan", "octane:start", "--server=roadrunner", "--host=0.0.0.0", "--port=8000"]
`

// --- Next.js ---

const nextjsDockerfile = `FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* yarn.lock* pnpm-lock.yaml* ./
RUN \
    if [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm install --frozen-lockfile; \
    elif [ -f package-lock.json ]; then npm ci; \
    else npm install; \
    fi

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Required for standalone output mode
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:20-alpine AS production
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs \
    && adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public

# Standalone output — set output: 'standalone' in next.config.js
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
`

// --- Strapi ---

const strapiDockerfile = `FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* yarn.lock* pnpm-lock.yaml* ./
RUN \
    if [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
    elif [ -f pnpm-lock.yaml ]; then corepack enable pnpm && pnpm install --frozen-lockfile; \
    elif [ -f package-lock.json ]; then npm ci; \
    else npm install; \
    fi

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NODE_ENV=production
RUN npm run build

FROM node:20-alpine AS production
WORKDIR /app
ENV NODE_ENV=production

RUN addgroup --system --gid 1001 strapi \
    && adduser --system --uid 1001 strapi

COPY --from=builder --chown=strapi:strapi /app ./

USER strapi

EXPOSE 1337

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
