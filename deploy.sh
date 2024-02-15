#!/bin/sh
gcloud functions deploy airpin \
  --region=us-central1 \
  --runtime=go120 \
  --trigger-topic=airpin \
  --set-secrets=PINTEREST_TOKEN=pinterest-token:latest,AIRTABLE_TOKEN=airtable-token:latest
