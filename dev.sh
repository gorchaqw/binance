#!/bin/sh

#scp db.sql root@62.113.104.230:/root/db.sql
#scp loki-config.yaml root@62.113.104.230:/root/loki-config.yaml
#scp prometheus-config.yaml root@62.113.104.230:/root/prometheus-config.yaml
# 10826 grafana prometheus go metrics


export BINANCE_VERSION="dev" &&
export BINANCE_PG_INIT_DB="./db.sql" &&
export BINANCE_LOKI="./loki-config.yaml" &&
export BINANCE_PROMETHEUS="./prometheus-config.yaml" &&
export BINANCE_APP_PORT="8080" &&
export BINANCE_APP_NAME="Binance_DEV" &&
export BINANCE_LOG_LEVEL="DEBUG"

docker build -t binance/binance:dev .
docker-compose up --remove-orphans


#  export BINANCE_APP_PORT="8080"
#  export BINANCE_VERSION="0.3.17"
#  export BINANCE_PG_INIT_DB="/root/db.sql"
#  export BINANCE_LOG_LEVEL="DEBUG"
#
#  docker-compose --context beget up --remove-orphans -d