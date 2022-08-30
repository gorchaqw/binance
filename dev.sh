#!/bin/sh

#scp db.sql root@37.46.130.3:/root/db.sql

export BINANCE_VERSION="dev" && export BINANCE_PG_INIT_DB="./db.sql" && export BINANCE_APP_PORT="8080"

export BINANCE_APP_PORT="8080" && export BINANCE_VERSION="0.2.27" && export BINANCE_PG_INIT_DB="/root/db.sql"

docker build -t binance/binance:dev .
docker-compose up --remove-orphans