# Service `ginit`

## Short summary

The `ginit` service is the service manager for the Gecholog container. `ginit` monitors configuration files for changes, manages the start and restart of services and consolidates the Gecholog system log.

## Options

| Option             | Description                                               |
|--------------------|-----------------------------------------------------------|
| -a                 | alias, set the name of the service                        |
| -o                 | specify filepath to configuration file                    |
| --validate         | print config validation info (accepts stdin)              |
| --version          | print version                                             |

## Example

This is how to validate a new config file in the docker context

    docker exec gecholog ./ginit -o app/conf/new_ginit_config.json --validate

## Configuration file

Example of the configuration file for the `ginit` service can be found [here](../../config/ginit_config.json).

| Field              | Description                                               |
|--------------------|-----------------------------------------------------------|
| log_level          | one of `DEBUG` `INFO` `WARN` `ERROR`                      | 
| max_starts         | max consecutive attempts to start child service (0 inf)   |
| services           | array with order child services are started               | 
| version            | the config file conforms to this specification            | 

For each service 

| Field                          | Description                                               |
|--------------------------------|-----------------------------------------------------------|
| additional_arguments           | optional arguments                                        |
| file                           | child service executable filepath                         |
| healthy_output                 | text string in system log to match when healthy           |  
| config_command                 | flag for configuration file                               | 
| configuration_file             | filepath to configuration file                            | 
| die_promise                    | kills ginit (and thus container) if unable to start       | 
| disable_config_file_monitoring | toggle of configuration file monitoring                   | 
| name                           | child service name in system log                          | 
| validate_command               | command to check config file                              | 
