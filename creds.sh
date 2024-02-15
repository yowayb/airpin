#!/bin/sh
#
# NB: This script must be run as `source creds.sh` in order to cache the 
#     tokens for the session, so as not to have to request the tokens for every
#     test run.

# Get the tokens from Google Secrets Manager if they haven't been set already.
export AIRTABLE_TOKEN=$(gcloud secrets versions access latest --secret=airtable-token)
export PINTEREST_TOKEN=$(gcloud secrets versions access latest --secret=pinterest-token)


