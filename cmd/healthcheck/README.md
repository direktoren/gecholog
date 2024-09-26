# Service `healthcheck`

## Short summary

The `healthcheck` service is a service to see if (1) whether a service is healthy and (2) what the current sha256 checksum of the currently deployed configuration file is for that service. `healthcheck` reports information based on the `/app/checksum/*_config.sha256` files. 

## Options

| Option             | Description                                               |
|--------------------|-----------------------------------------------------------|
| -p                 | print checksum in production                              |
| -s                 | service to check                                          |
| --version          | print version                                             |

## Example

This is how run in the docker context

    docker exec gecholog ./healthcheck -s gl -p

Prints

    9ac52f58f2c480abf7254657648acde0709e1b37c468e72e49714d4e2bebf1c7

Healthy

    echo $?                                                                                                     
    0

Not healthy

    echo $?                                                                                                     
    1