#!/usr/bin/env bash

# This script bumps the KEB images in the chart, utils and the KEB chart version.
# It has the following arguments:
#   - release tag (mandatory)
# ./bump_keb_chart.sh 0.0.0

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


RELEASE_TAG=$1
VALUES_YAML="resources/keb/values.yaml"

KEYS=$(yq e '.global.images | keys | .[]' $VALUES_YAML | grep 'kyma_environment')

# bump images in resources/keb/values.yaml

yq e ".version = \"$RELEASE_TAG\"" -i resources/keb/Chart.yaml
yq e ".appVersion = \"$RELEASE_TAG\"" -i resources/keb/Chart.yaml

