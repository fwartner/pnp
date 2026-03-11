FROM dunglas/frankenphp:latest-php8.4-alpine

# Upgrade all packages to latest
RUN apk upgrade --no-cache && apk add --no-cache \
    dumb-init \
    curl

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
    && echo "opcache.memory_consumption=256" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.interned_strings_buffer=16" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.max_accelerated_files=20000" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.validate_timestamps=0" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.jit=1255" >> "$PHP_INI_DIR/conf.d/opcache.ini" \
    && echo "opcache.jit_buffer_size=128M" >> "$PHP_INI_DIR/conf.d/opcache.ini"

WORKDIR /app

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8000/up || exit 1

EXPOSE 8000

ENTRYPOINT ["dumb-init", "--"]
CMD ["php", "artisan", "octane:start", "--server=frankenphp", "--host=0.0.0.0", "--port=8000"]
