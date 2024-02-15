#!/bin/sh
#
# Get templates

curl -X GET https://api.pinterest.com/v5/ad_accounts/549762139336/templates \
  -H "Authorization: Bearer ${PINTEREST_TOKEN}" > templates.json
