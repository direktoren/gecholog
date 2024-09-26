# Service `entrypoint`

## Short summary

The `entrypoint` service is the Gecholog version of a container entrypoint script. It copies configuration files to its destination (unless files already exist there), makes sure to create random values for environment variables and finally kicks off the main service.

## Options

| Option             | Description                                               |
|--------------------|-----------------------------------------------------------|
| --version          | print version                                             |
| -t                 | target folder for files                                   |
| -s                 | source folder for files                                   |
| -f                 | colon separated list of files                             |
| -e                 | env vars to randomize if == `not_set`                     |
| -c                 | child process to spawn `command:arg1:arg2:..`             |
