# Module configuration

By default, KEB will apply to cluster the Kyma CR with default modules.
If "modules" section is not set at all (mapped to nil), or "modules" section is empty(not nil but zero values) - then default modules will be applied.
Configuration is available by setting "modules" object in JSON schema.
There are two available fields:
- default (bool)
- list (array)
API for modules configuration is build on "oneOf" feature from JSON schema. It means if "modules" is passed to API - it must have one and only one valid option, thus to pass JSON API Validator you must set only one field. See examples below.

Correct API calls:

- install default modules

```
modules: {
    default: true
}
```

- custom modules configuration - dont install any module
```
modules: {
    list: [] // no modules
}
```

- custom modules configuration - install only btp-operator module
```
modules: {
    list: [btp-opertor]
}
```

Incorrect API calls:

- this will pass JSON validation, but it will fail due to not passed any custom modules. If someone want to use custom modules or empty modules should use only "list" field.
```
modules: {
    default: false
}
```

- any call with "modules" and filled with both params will fail, it is due usage of "oneOf" feature in JSON Schema.
```
modules: {
    default: false,
    list: [btp-opertor]
}
```
