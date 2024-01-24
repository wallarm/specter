#!/bin/bash

# Generate CI_PIPELINE_ID if not set
if [ -z "${CI_PIPELINE_ID}" ]; then
    if command -v shuf > /dev/null; then
        CI_PIPELINE_ID="local-$(shuf -i 1-100000 -n 1)"
    else
        CI_PIPELINE_ID="local-$(awk 'BEGIN{srand();print int(rand()*(100000-1))+1 }')"
    fi
    echo "CI_PIPELINE_ID is not set, using random value ${CI_PIPELINE_ID}"
    export CI_PIPELINE_ID
fi

./bin/specter-darwin-arm64 --upload
