#!/bin/bash

curl -H "Content-Type: application/json" \
-s "http://62.113.104.230:3000/api/datasources" \
-u admin:TSr9Msh%sSEYai | jq -c -M '.[]' |  split -l 1 - ./grafana/

for i in data_sources/*; do \
	curl -X "POST" "http://localhost:3000/api/datasources" \
    -H "Content-Type: application/json" \
     --user admin:admin \
     --data-binary @$i
done
