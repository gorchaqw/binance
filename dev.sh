#!/bin/sh

#scp db.sql root@37.46.130.3:/root/db.sql

export BINANCE_VERSION="dev" &&
export BINANCE_PG_INIT_DB="./db.sql" &&
export BINANCE_APP_PORT="8080" &&
export BINANCE_LOG_LEVEL="DEBUG"

docker build -t binance/binance:dev .
docker-compose up --remove-orphans