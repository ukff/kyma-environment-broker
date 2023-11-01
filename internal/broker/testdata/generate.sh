#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

order=("name" "kubeconfig" "shootName" "shootDomain" "region" "machineType" "autoScalerMin" "autoScalerMax" "zonesCount" "modules" "networking" "oidc" "administrators")
key="modules"

for file in *; do
    if [[ $file != update* ]] && [[ $file != generate* ]]; then
        echo $file
        jq --argjson groupInfo "$(<generate.json)" '.properties += $groupInfo' "$file" > tmp.$$ && mv tmp.$$ "$file"
        jq -S . "$file" > tmp.$$ && mv tmp.$$ "$file"
        #jq '._controlsOrder |= (reduce .[] as $item ([]; if $item == "networking" then . + ["modules", $item] else . + [$item] end))' "$file" > tmp.$$ && mv tmp.$$ "$file"
        controls=("name" "kubeconfig" "shootName" "shootDomain" "region" "machineType" "autoScalerMin" "autoScalerMax" "zonesCount" "modules" "networking" "oidc" "administrators")
        key="modules"

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


USAGE=$(cat <<-END
{
  "modules": {
    "_controlsOrder": [
      "useDefault",
      "list"
    ],
    "description": "Use default modules or provide your custom list of modules.",
    "oneOf": [
      {
        "additionalProperties": false,
        "description": "Check the default modules at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud.\n",
        "properties": {
          "useDefault": {
            "default": true,
            "description": "Select a modules technical key from the list available at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud. You can only use a modules technical key once.",
            "readOnly": true,
            "type": "boolean"
          }
        },
        "title": "Default",
        "type": "object"
      },
      {
        "additionalProperties": false,
        "description": "Check the default modules at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud.\n",
        "properties": {
          "list": {
            "description": "Select a modules technical key from the list available at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud. You can only use a modules technical key once.",
            "items": {
              "_controlsOrder": [
                "name",
                "channel",
                "customResourcePolicy"
              ],
              "properties": {
                "channel": {
                  "_enumDisplayName": {
                    "fast": "Fast - latest version",
                    "regular": "Regular - default version\n"
                  },
                  "default": "regular",
                  "description": "Select your preferred release channel.\n",
                  "enum": [
                    "regular",
                    "fast"
                  ],
                  "type": "string"
                },
                "customResourcePolicy": {
                  "_enumDisplayName": {
                    "CreateAndDelete": "CreateAndDelete - default module resource is created or deleted.",
                    "Ignore": "Ignore - module resource is not created."
                  },
                  "default": "CreateAndDelete",
                  "description": "Select your preferred CustomResourcePolicy setting.",
                  "enum": [
                    "CreateAndDelete",
                    "Ignore"
                  ],
                  "type": "string"
                },
                "name": {
                  "description": "Select a modules technical key from the list available at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud. You can only use a modules technical key once.",
                  "minLength": 1,
                  "title": "name",
                  "type": "string"
                }
              }
            },
            "type": "array",
            "uniqueItems": true
          }
        },
        "title": "Custom",
        "type": "object"
      }
    ],
    "type": "object"
  }
}
END)