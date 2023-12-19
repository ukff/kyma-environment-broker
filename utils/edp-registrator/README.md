# Event Data Platform Tools

This folder contains tools that allow you to get information about subaccounts registered in the Event Data Platform (EDP) and execute registration.

## EDP Tool

The EDP tool allows you to connect to the EDP and execute the following commands:
 - `get` - gathers information about a registered subaccount; if not found, returns the message `Not found`
 - `register` - performs registration 
 - `deregister` - removes the registration

The above command implementation contains a copied code from the implementation of existing steps. Before running, check if the command implementation is up-to-date.

### Build

Run the following command to build the binary:

```
go build -o edp main.go
```

### Running

#### Set Environment Variables

Before using the `edp` tool, you must set environment variables:

1. Copy an existing template file, for example:

`cp env.dev.template env.dev`

2. Set the missing secret value in the file.
3. Export the environment variables:

`export $(grep -v '^#' env.dev | xargs)`

#### Run a Command

1. Get metadata from the EDP:
```shell
./edp get <subaccountID>
```

2. Register a subaccount in the EDP:
```shell
./edp register <subaccount ID> <platform region> <plan>
```
for example:
```shell
./edp register 41ba3cf2-041d-4223-adfe-c5de3458acbe cf-us21 standard
```

3. Deregister a subaccount from the EDP:
```shell
./edp deregister <subaccount>
```