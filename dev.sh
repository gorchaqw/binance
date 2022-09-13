#!/bin/sh


# 10826 grafana prometheus go metrics

#
#export BINANCE_VERSION="dev" &&
#export BINANCE_PG_INIT_DB="./db.sql" &&
#export BINANCE_LOKI="./loki-config.yaml" &&
#export BINANCE_PROMETHEUS="./prometheus-config.yaml" &&
#export BINANCE_APP_PORT="8080" &&
#export BINANCE_APP_NAME="Binance_DEV" &&
#export BINANCE_LOG_LEVEL="DEBUG"
#
#docker build --no-cache -t binance/binance:dev .
#docker-compose up --remove-orphans -d
#

scp db.sql root@62.113.99.249:/root/db.sql
scp loki-config.yaml root@62.113.99.249:/root/loki-config.yaml
scp prometheus-config.yaml root@62.113.99.249:/root/prometheus-config.yaml

export BINANCE_APP_PORT="8080"
export BINANCE_APP_NAME="Binance_Beget_DEV"
export BINANCE_VERSION="dev"
export BINANCE_PG_INIT_DB="/root/db.sql"
export BINANCE_LOKI="/root/loki-config.yaml"
export BINANCE_PROMETHEUS="/root/prometheus-config.yaml"
export BINANCE_LOG_LEVEL="ERROR"

docker --context beget build --no-cache -t binance/binance:dev .
docker-compose --context beget up --remove-orphans -d

#docker-compose stop

#  export BINANCE_APP_PORT="8080"
#  export BINANCE_VERSION="0.3.17"
#  export BINANCE_PG_INIT_DB="/root/db.sql"
#  export BINANCE_LOG_LEVEL="DEBUG"
#
#  docker-compose --context beget-dev up --remove-orphans -d



#docker build -t binance/binance:dev .

#docker context create beget --docker "host=ssh://root@62.113.99.249"

#docker context update --docker "host=ssh://root@62.113.99.249" beget

#scp db.sql root@37.46.130.3:/root/db.sql
#scp loki-config.yaml root@37.46.130.3:/root/loki-config.yaml
#scp prometheus-config.yaml root@37.46.130.3:/root/prometheus-config.yaml

#export BINANCE_APP_PORT="8080"
#export BINANCE_APP_NAME="Binance_FirstVDS_DEV"
#export BINANCE_VERSION=dev
#export BINANCE_PG_INIT_DB="/root/db.sql"
#export BINANCE_LOKI="/root/loki-config.yaml"
#export BINANCE_PROMETHEUS="/root/prometheus-config.yaml"
#export BINANCE_LOG_LEVEL="ERROR"
#
#docker-compose --context firstvds up --remove-orphans -d
#docker-compose --context firstvds stop