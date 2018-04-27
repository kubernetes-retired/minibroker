#!/usr/bin/env bash

set -xeuo pipefail

helm init --client-only

#####
# set up the repo dir, and package up all charts
#####
REPO_ROOT=https://minibroker.blob.core.windows.net
AZURE_STORAGE_CONTAINER=charts
REPO_DIR=bin/charts
mkdir -p $REPO_DIR
echo "entering $REPO_DIR"
cd $REPO_DIR
# download the existing repo's index.yaml so that we can merge it later
echo "downloading existing index.yaml"
curl -sLO ${REPO_ROOT}/${AZURE_STORAGE_CONTAINER}/index.yaml
for dir in `ls ../../charts`;do
    if [ ! -f ../../charts/$dir/Chart.yaml ];then
        echo "skipping $dir because it lacks a Chart.yaml file"
    else
        echo "packaging $dir"
        helm dep build ../../charts/$dir
        helm package ../../charts/$dir
    fi
done

#####
# index the charts, merging with the old index.yaml so charts are versioned
#####
helm repo index --url "$REPO_ROOT/$AZURE_STORAGE_CONTAINER" --merge index.yaml .

#####
# upload to Azure blob storage
#####

if [ -z "$AZURE_STORAGE_CONNECTION_STRING" ]; then
    echo "AZURE_STORAGE_CONNECTION_STRING env var required to publish"
    exit 1
fi

echo "uploading from $PWD"
az storage blob upload-batch --destination $AZURE_STORAGE_CONTAINER --source .
