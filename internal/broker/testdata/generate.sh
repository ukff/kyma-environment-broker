#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

controls=("name" "kubeconfig" "shootName" "shootDomain" "region" "machineType" "autoScalerMin" "autoScalerMax" "zonesCount" "modules" "networking" "oidc" "administrators")
key="modules"

for file in *; do
    if [[ $file != update* ]] && [[ $file != generate* ]]; then
        echo $file
        jq --argjson groupInfo "$(<generate.json)" '.properties += $groupInfo' "$file" > tmp.$$ && mv tmp.$$ "$file"
        jq -S . "$file" > tmp.$$ && mv tmp.$$ "$file"
        #jq '._controlsOrder |= (reduce .[] as $item ([]; if $item == "networking" then . + ["modules", $item] else . + [$item] end))' "$file" > tmp.$$ && mv tmp.$$ "$file"

        # Read the _controlsOrder array from JSON file
        readarray -t current < <(jq -r '._controlsOrder[]' $file)

        # Create associative array with values from controls array and their indexes as elements
        declare -A controls_assoc
        for i in "${!controls[@]}"; do
          controls_assoc["${controls[$i]}"]=$i
        done

        # Create the updated array
        declare -a updated=()
        for item in "${controls[@]}"; do
          if [[ " ${current[@]} " =~ " ${item} " ]] || [[ "${item}" == "${key}" ]]; then
            updated+=("$item")
          fi
        done

        # Convert bash array to JSON array
        json_updated=$(printf '%s\n' "${updated[@]}" | jq -R . | jq -s .)

        # Write array back into JSON file
        jq '._controlsOrder='"$json_updated" $file > tmp.json && mv tmp.json $file
    fi
done