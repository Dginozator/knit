#!/bin/bash
# Get IAM token for Yandex Cloud (Deprecated - using API key instead)
# This script is kept for reference but API key authentication is preferred

set -e

YC_TOKEN=""

# Get IAM token from OAuth token
get_iam_token() {
    local oauth_token="$1"
    
    curl -s -X POST \
        -H 'Content-Type: application/json' \
        -d "{\"yandexPassportOauthToken\": \"$oauth_token\"}" \
        "https://iam.api.cloud.yandex.net/iam/v1/tokens"
}

# Get token from service account key file
get_token_from_key() {
    local key_file="$1"
    
    curl -s -X POST \
        -H 'Content-Type: application/json' \
        -d @"$key_file" \
        "https://iam.api.cloud.yandex.net/iam/v1/tokens"
}

# Usage
if [ $# -lt 1 ]; then
    echo "Usage: $0 <oauth_token|key_file>"
    echo "For service account: $0 service-account-key.json"
    echo "For user OAuth: $0 <oauth_token>"
    exit 1
fi

if [ -f "$1" ]; then
    echo "Getting IAM token from service account key..."
    get_token_from_key "$1"
else
    echo "Getting IAM token from OAuth token..."
    get_iam_token "$1"
fi
