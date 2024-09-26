# Service `gui`

## Short summary

The `gui` service is the web-based interface of the Gecholog container. It accepts `http` and `https` traffic and requires the `GUI_SECRET` as password.

## Options

| Option             | Description                                               |
|--------------------|-----------------------------------------------------------|
| -a                 | alias, set the name of the service                        |
| -o                 | specify filepath to configuration file                    |
| --validate         | print config validation info (accepts stdin)              |
| --version          | print version                                             |

## Example

This is how to validate a new config file in the docker context

    docker exec -e NATS_TOKEN=set gecholog ./gui -o app/conf/new_gui_config.json --validate

## Configuration file

Example of the configuration file for the `gui` service can be found [here](../../config/gui_config.json).

| Field                       | Description                                               |
|-----------------------------|-----------------------------------------------------------|
| archive_directory           | container path to folder for stored files                 | 
| failed_authentication_limit | failed login attempts before dying (0 infinite)           | 
| gl                          | details for the gl service                                |
| gui_port                    | port number for the service. Default 8080                 |
| log_level                   | one of `DEBUG` `INFO` `WARN` `ERROR`                      | 
| nats2file                   | details for the nats2file service                         |
| nats2log                    | details for the nats2log service                          |
| secret                      | login password                                            | 
| service_bus                 | internal service bus configuration                        |
| tls                         | TLS settings                                              |
| tokencounter                | `NOT IN USE`         |
| version                     | the config file conforms to this specification            | 
| working_directory           | container path to folder for staged files                 | 

For each service 

| Field                          | Description                                         |
|--------------------------------|-----------------------------------------------------|
| additional_arguments           | optional arguments                                  |
| config_command                 | flag for configuration file                         | 
| name                           | service name in the ui                              | 
| production_checksum_file       | service checksum file filepath                      | 
| production_file                | filepath to config file in production               | 
| template_file                  | filepath to config file for restore option          | 
| validate_command               | command to check config file                        | 
| validation_executable          | child service executable file                       |
