# Modules

By default, Kyma Environment Broker applies the Kyma CR, including the default modules selected by Kyma, to a cluster.
Modules configuration is available by setting "modules" object in JSON schema.
There are two available fields, full JSON Schema can be found on bottom of document.
- default (bool) - Defines if use default module settings
- list (array) - Defines a custom list of modules

API for module configuration is built on the "oneOf" feature from the JSON schema. If the "modules" object is passed to API, it must have only one valid option. Thus, to pass JSON API Validator, you must set only one field. See examples below.

Correct API calls:

- In the default mode, the "modules" object is set to `default: true` and modules are pre-selected by Kyma. The same happens when the "modules" section is not set at all (mapped to nil) in the payload.

```
modules: {
    default: true
}
```

- If you want to use your custom modules configuration, you must pass "list" with the modules you want installed.

```
"modules": {
    "list": [
        {
            "name": "btp-operator",
            "customResourcePolicy": "CreateAndDelete"
        },
        {
            "name": "keda",
            "channel": "fast",
        }
    ]
}
```

- The following calls result in Kyma runtime installation without any modules.

```
"modules": {
    "list": []
}
```

```
"modules": {
    "default": false
}
```

Incorrect API calls:

A call with the "modules" section not set fails due to using the "oneOf" feature in the JSON Schema.

```
modules: {}
```

A call with the "modules" section with both parameters filled in fails due to using the "oneOf" feature in the JSON Schema.

```
modules: {
    default: false,
    "list": [
        {
            "name": "btp-operator",
            "customResourcePolicy": "CreateAndDelete"
        },
        {
            "name": "keda",
            "channel": "fast",
        }
    ]
}
```

JSON Schema

```
"modules": {
    "_controlsOrder": [
      "default",
      "list"
    ],
    "description": "Use default modules or provide your custom list of modules.",
    "oneOf": [
      {
        "additionalProperties": false,
        "description": "Default modules",
        "properties": {
          "default": {
            "default": true,
            "description": "Check the default modules at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud.",
            "readOnly": true,
            "type": "boolean"
          }
        },
        "title": "Default",
        "type": "object"
      },
      {
        "additionalProperties": false,
        "description": "Define custom module list",
        "properties": {
          "list": {
            "description": "Select a module technical name from the list available at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud. You can only use a modules technical name once.",
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
                    "regular": "Regular - default version"
                  },
                  "default": "regular",
                  "description": "Select your preferred release channel.",
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
                  "description": "Select a module technical name from the list available at: https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud. You can only use a modules technical name once.",
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
```