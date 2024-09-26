# Service `nats2log`

## Short summary

The `nats2log` service listens to the internal service bus of the Gecholog container and writes the final logs to one of the following destinations: a file, an elastic api, a static rest api or an azure log analytics api. By default, Gecholog attempts to start two instances of the `nats2log` service, the second one with the alias `nats2file`. However, without additional configuration, only `nats2file` will be running.

## Options

| Option             | Description                                               |
|--------------------|-----------------------------------------------------------|
| -a                 | alias, set the name of the service                        |
| -o                 | specify filepath to configuration file                    |
| --validate         | print config validation info (accepts stdin)              |
| --version          | print version                                             |

## Example

This is how to validate a new config file in the docker context

    docker exec -e NATS_TOKEN=set gecholog ./nats2log -o app/conf/new_nats2log_config.json --validate

## Configuration file

Example of the configuration file for the `nats2log` service can be found [here](../../config/nats2log_config.json) and for the `nats2file` service [here](../../config/nats2file_config.json).

| Field | Description  | 
|----------------------|------------------------|
| azure_log_analytics_writer | details for sending logs to azure log analytics |
| elastic_writer | details for sending logs to elastic |
| file_writer | details to write logs to file(s) |
| log_level | one of `DEBUG` `INFO` `WARN` `ERROR` | 
| mode | one of `azure_log_analytics_writer` `elastic_writer` `file_writer` `rest_api_writer`| 
| rest_api_writer | details for sending logs to custom rest endpoint |
| retries | number of retries dispatching the log to store |
| retry_delay_milliseconds | delay before retry in milliseconds |
| service_bus | internal service bus configuration | 
| tls | TLS settings | 
| version | the config file conforms to this specification | 

Settings for `azure_log_analytics_writer`

| Field | Description  | 
|----------------------|------------------------|
| log_type | logtype name | 
| workspace_id | azure log analytics workspace id | 
| shared_key | azure log analytics access key | 

Settings for `elastic_writer`

| Field | Description  | 
|----------------------|------------------------|
| url | URL to elastic | 
| port | elastic port | 
| index | index to post logs to | 
| username | `elastic_writer` uses -u option to POST. This is username | 
| password | `elastic_writer` uses -u option to POST. This is pwd | 

Settings for `file_writer`

| Field | Description  | 
|----------------------|------------------------|
| filename | filepath in container to where traffic logfile is written | 
| write_mode | one of `append` `new` `overwrite` | 

Settings for `rest_api_writer`

| Field | Description  | 
|----------------------|------------------------|
| endpoint | POST to url+port:endpoint | 
| headers | map[name]-array of values for outbound headers | 
| port | POST to url+port:endpoint | 
| url | POST to url+port:endpoint | 
