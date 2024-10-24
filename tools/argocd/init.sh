#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

ACCOUNT_ID=$(echo "${AWS_ROLE_ARN}" | cut -d':' -f5)
mkdir -p /home/argocd/.docker
cat >/home/argocd/.docker/config.json <<EOF
{
    "credHelpers": {
        "${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com": "ecr-login"
    }
}
EOF
