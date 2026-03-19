package types

import "github.com/fwartner/pnp/internal/config"

// Shared Laravel constants

const laravelValuesYAML = `spec:
  source:
    repoURL: << .RepoURL >>
    targetRevision: main
    path: << .ChartPath >>
  destination:
    namespace: << .Namespace >>
domain: << .Domain >>
subdomain: << .Subdomain >>
image:
  repository: << .Image >>
  tag: << .Tag >>
app:
  key: << .AppKey >>
database:
  name: << .DBName >>
  username: << .DBUsername >>
horizon:
  enabled: << .HorizonEnabled >>
reverb:
  enabled: << .ReverbEnabled >>
  port: << .ReverbPort >>
octane:
  enabled: << .OctaneEnabled >>
  server: << .OctaneServer >>
mail:
  from: info@pixelandprocess.de
`

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

// LaravelDockerfileFor returns the appropriate Dockerfile for a Laravel project.
func LaravelDockerfileFor(cfg config.ProjectConfig) string {
	if cfg.Octane.Enabled {
		switch cfg.Octane.Server {
		case "swoole":
			return laravelOctaneSwooleDockerfile
		case "roadrunner":
			return laravelOctaneRoadrunnerDockerfile
		default:
			return laravelOctaneFrankenphpDockerfile
		}
	}
	return laravelStandardDockerfile
}

const laravelStandardDockerfile = `# ---- Composer dependencies ----
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
FROM ghcr.io/pixel-process-ug/laravel-base:latest

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

// LaravelScaffoldFiles returns the scaffold files shared by laravel-web and laravel-api.
func LaravelScaffoldFiles(data ScaffoldData) map[string]string {
	return map[string]string{
		"composer.json": `{
    "name": "` + data.ShortName + `",
    "type": "project",
    "description": "` + data.Name + ` — managed by pnp",
    "require": {
        "php": "^8.2",
        "laravel/framework": "^12.0"
    },
    "require-dev": {
        "phpunit/phpunit": "^11.0"
    },
    "autoload": {
        "psr-4": {
            "App\\": "app/"
        }
    },
    "minimum-stability": "stable",
    "prefer-stable": true
}
`,
		".env.example": `APP_NAME=` + data.Name + `
APP_ENV=local
APP_KEY=
APP_DEBUG=true
APP_URL=https://` + data.Domain + `

DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=app
DB_USERNAME=app
DB_PASSWORD=
`,
		"artisan": `#!/usr/bin/env php
<?php

define('LARAVEL_START', microtime(true));

require __DIR__.'/vendor/autoload.php';

$app = require_once __DIR__.'/bootstrap/app.php';

$kernel = $app->make(Illuminate\Contracts\Console\Kernel::class);

$status = $kernel->handle(
    $input = new Symfony\Component\Console\Input\ArgvInput,
    new Symfony\Component\Console\Output\ConsoleOutput
);

$kernel->terminate($input, $status);

exit($status);
`,
		"app/.gitkeep":               "",
		"routes/.gitkeep":            "",
		"config/.gitkeep":            "",
		"database/migrations/.gitkeep": "",
		"resources/.gitkeep":         "",
		"storage/logs/.gitkeep":      "",
		"public/index.php": `<?php

use Illuminate\Http\Request;

define('LARAVEL_START', microtime(true));

require __DIR__.'/../vendor/autoload.php';

$app = require_once __DIR__.'/../bootstrap/app.php';

$kernel = $app->make(Illuminate\Contracts\Http\Kernel::class);

$response = $kernel->handle(
    $request = Request::capture()
)->send();

$kernel->terminate($request, $response);
`,
	}
}
