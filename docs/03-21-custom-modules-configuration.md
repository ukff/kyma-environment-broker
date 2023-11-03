# Module configuration

By default, KEB will apply to cluster the Kyma CR with default modules selected by Kyma.
Configuration is available by setting "modules" object in JSON schema.
There are two available fields, full JSON Schema can be found on bottom of document.
- default (bool)
- list (array)

API for modules configuration is build on "oneOf" feature from JSON schema. It means if "modules" is passed to API - it must have one and only one valid option, thus to pass JSON API Validator you must set only one field. See examples below.

# Correct API calls:

- default mode, if set to true, then modules will be selected by Kyma, also same happens when "modules" section is not set at all (mapped to nil) in payload

```
modules: {
    default: true
}
```

- if you want to use your custom modules configuration, you need pass 'list' with modules which you want install

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

- this calls will cause installation Kyma without any modules

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

# Incorrect API calls:

Any call with "modules" section not set will fail, it is due usage of "oneOf" feature in JSON Schema.

```
modules: {}
```

Any call with "modules" section and filled with both params will fail, it is due usage of "oneOf" feature in JSON Schema.

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

# JSON Schema

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