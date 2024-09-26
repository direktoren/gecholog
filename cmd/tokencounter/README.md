# Service `tokencounter`

## Short summary

The `tokencounter` service is a processor (micro-service) that runs by default in the Gecholog container. `tokencounter` extracts token consumption fields from the response payload and creates a new field in the log that enables standardized reporting across different models. `tokencounter` can also throttle the traffic based on token consumption.

## Options

| Option             | Description                                               |
|--------------------|-----------------------------------------------------------|
| -a                 | alias, set the name of the service                        |
| -o                 | specify filepath to configuration file                    |
| --validate         | print config validation info (accepts stdin)              |
| --version          | print version                                             |

## Example

This is how to validate a new config file in the docker context

    docker exec -e NATS_TOKEN=set gecholog ./tokencounter -o app/conf/new_tokencounter_config.json --validate

## Configuration file

Example of the configuration file for the `tokencounter` service can be found [here](../../config/tokencounter_config.json).

| Field | Description  | 
|----------------------|------------------------|
| cap_period_seconds | period in seconds after which token cap counter resets  | 
| log_level | one of `DEBUG` `INFO` `WARN` `ERROR` | 
| service_bus | internal service bus configuration | 
| token_caps | array token caps | 
| usage_fields | array of patterns for tokens | 
| version | the config file conforms to this specification | 

Usage field settings

| Field | Description  | 
|----------------------|------------------------|
| router | specify router where pattern applies, or `default` | 
| patterns | arrays of search patterns | 

| Field | Description  | 
|----------------------|------------------------|
| field | field name | 
| pattern  | json-search path to token object | 

json-search path uses [gjson](https://github.com/tidwall/gjson) syntax.

Token cap settings

| Field | Description  | 
|----------------------|------------------------|
| router | specify router where cap applies | 
| fields | arrays of cap fields | 


| Field | Description  | 
|----------------------|------------------------|
| field | field name | 
| value | token cap. 0 means no cap | 
