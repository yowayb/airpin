#!/bin/sh
#
# NB: Set credentials by sourcing `creds.sh` first.

# Build and serve the function locally.
pushd cmd
go build
./cmd >../cmd_stdout.log 2>../cmd_stderr.log & 
pid=$!
sleep 2
echo "Serving function with PID $pid."

# Send a CloudEvent to the locally running function:
# "7-day" base64-encoded is "Ny1kYXk="
# "month" base64-encoded is "bW9udGg="
curl localhost:8080/airpin \
  -X POST \
  -H "Content-Type: application/json" \
  -H "ce-id: 123451234512345" \
  -H "ce-specversion: 1.0" \
  -H "ce-time: 2020-01-02T12:34:56.789Z" \
  -H "ce-type: google.cloud.pubsub.topic.v1.messagePublished" \
  -H "ce-source: //pubsub.googleapis.com/projects/complete-road-241116/topics/airpin" \
  -d '{
        "message": {
          "data": "Ny1kYXk="
        },
        "subscription": "projects/complete-road-241116/subscriptions/airpin"
      }'
echo 'Sent CloudEvent.'

kill $pid
echo "Stopped function with PID $pid."
popd
