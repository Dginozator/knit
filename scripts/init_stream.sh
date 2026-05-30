#!/bin/bash
# Initialize Yandex Data Streams stream

set -e

STREAM_NAME="${YDS_STREAM:-messenger-stream}"
ENDPOINT="${YDS_ENDPOINT:-endpoint.yaml.rus.cloud-apps.store}"
REGION="${YDS_REGION:-ru-central1}"
API_KEY="${YDS_API_KEY}"
SHARD_COUNT="${YDS_SHARD_COUNT:-1}"

if [ -z "$API_KEY" ]; then
    echo "Error: YDS_API_KEY environment variable is not set"
    exit 1
fi

echo "Initializing stream: $STREAM_NAME"
echo "Endpoint: $ENDPOINT"
echo "Region: $REGION"
echo "Shards: $SHARD_COUNT"

# Using AWS CLI with Yandex Cloud endpoint
aws kinesis create-stream \
    --endpoint-url "https://$ENDPOINT" \
    --region "$REGION" \
    --stream-name "$STREAM_NAME" \
    --shard-count "$SHARD_COUNT" \
    --profile yandex

echo "Stream created successfully!"

# Wait for stream to become active
echo "Waiting for stream to become active..."
aws kinesis describe-stream \
    --endpoint-url "https://$ENDPOINT" \
    --region "$REGION" \
    --stream-name "$STREAM_NAME" \
    --profile yandex \
    --query 'StreamDescription.StreamStatus'

echo "Stream is ready!"
