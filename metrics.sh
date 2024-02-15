#!/bin/sh
# Get metrics definitions JSON from Pinterest
curl -X GET https://api.pinterest.com/v5/resources/delivery_metrics\?report_type=ASYNC \
  --header "Authorization: Bearer ${PINTEREST_TOKEN}" \
  | jq . > metrics.json
