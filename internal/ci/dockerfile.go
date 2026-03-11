package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fwartner/pnp/internal/config"
)

const laravelDockerfile = `FROM php:8.3-fpm-alpine AS base

RUN apk add --no-cache \
    postgresql-dev \
    libzip-dev \
    oniguruma-dev \
    && docker-php-ext-install pdo_pgsql zip mbstring opcache pcntl

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /app

FROM base AS composer-deps
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist

FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM base AS production
COPY --from=composer-deps /app/vendor ./vendor
COPY . .
RUN composer dump-autoload --optimize

COPY --from=node-build /app/public/build ./public/build

RUN php artisan config:cache \
    && php artisan route:cache \
    && php artisan view:cache

EXPOSE 80
CMD ["php-fpm"]
`

const laravelOctaneFrankenphpDockerfile = `FROM dunglas/frankenphp:latest-php8.3-alpine AS base

RUN install-php-extensions \
    pdo_pgsql \
    zip \
    mbstring \
    opcache \
    pcntl \
    redis

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /app

FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM base AS production
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist
COPY . .
RUN composer dump-autoload --optimize

COPY --from=node-build /app/public/build ./public/build

RUN php artisan config:cache \
    && php artisan route:cache \
    && php artisan view:cache

EXPOSE 8000
CMD ["php", "artisan", "octane:start", "--server=frankenphp", "--host=0.0.0.0", "--port=8000"]
`

const laravelOctaneSwooleDockerfile = `FROM php:8.3-cli-alpine AS base

RUN apk add --no-cache \
    postgresql-dev \
    libzip-dev \
    oniguruma-dev \
    linux-headers \
    $PHPIZE_DEPS \
    && docker-php-ext-install pdo_pgsql zip mbstring opcache pcntl \
    && pecl install swoole redis \
    && docker-php-ext-enable swoole redis

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /app

FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM base AS production
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist
COPY . .
RUN composer dump-autoload --optimize

COPY --from=node-build /app/public/build ./public/build

RUN php artisan config:cache \
    && php artisan route:cache \
    && php artisan view:cache

EXPOSE 8000
CMD ["php", "artisan", "octane:start", "--server=swoole", "--host=0.0.0.0", "--port=8000"]
`

const laravelOctaneRoadrunnerDockerfile = `FROM php:8.3-cli-alpine AS base

RUN apk add --no-cache \
    postgresql-dev \
    libzip-dev \
    oniguruma-dev \
    $PHPIZE_DEPS \
    && docker-php-ext-install pdo_pgsql zip mbstring opcache pcntl sockets \
    && pecl install redis \
    && docker-php-ext-enable redis

COPY --from=composer:2 /usr/bin/composer /usr/bin/composer
COPY --from=ghcr.io/roadrunner-server/roadrunner:latest /usr/bin/rr /usr/bin/rr

WORKDIR /app

FROM node:20-alpine AS node-build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci
COPY . .
RUN npm run build

FROM base AS production
COPY composer.json composer.lock ./
RUN composer install --no-dev --no-scripts --no-autoloader --prefer-dist
COPY . .
RUN composer dump-autoload --optimize

COPY --from=node-build /app/public/build ./public/build

RUN php artisan config:cache \
    && php artisan route:cache \
    && php artisan view:cache

EXPOSE 8000
CMD ["php", "artisan", "octane:start", "--server=roadrunner", "--host=0.0.0.0", "--port=8000"]
`

const nextjsDockerfile = `FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:20-alpine AS production
WORKDIR /app
ENV NODE_ENV=production

COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000
CMD ["node", "server.js"]
`

const strapiDockerfile = `FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:20-alpine AS production
WORKDIR /app
ENV NODE_ENV=production

COPY --from=deps /app/node_modules ./node_modules
COPY --from=builder /app ./

EXPOSE 1337
CMD ["npm", "run", "start"]
`

// GenerateDockerfile creates a Dockerfile in the project directory based on the project type.
func GenerateDockerfile(projectType string, octaneCfg config.OctaneConfig, projectDir string) error {
	var content string

	switch projectType {
	case "laravel-web", "laravel-api":
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
	case "strapi":
		content = strapiDockerfile
	default:
		return fmt.Errorf("unsupported project type for Dockerfile: %s", projectType)
	}

	tmpl, err := template.New("dockerfile").Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse Dockerfile template: %w", err)
	}

	outPath := filepath.Join(projectDir, "Dockerfile")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create Dockerfile: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, nil); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	return nil
}
