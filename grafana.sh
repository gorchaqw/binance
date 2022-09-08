#!/bin/bash

#curl -H "Content-Type: application/json" \
#-s "http://62.113.104.230:3000/api/datasources" \
#-u admin:TSr9Msh%sSEYai | jq -c -M '.[]' |  split -l 1 - ./grafana/

for i in grafana/data_source/*; do \
	curl -X "POST" "http://62.217.178.7:3000/api/datasources" \
    -H "Content-Type: application/json" \
     --user admin:TSr9Msh%sSEYai \
     --data-binary @$i
done
