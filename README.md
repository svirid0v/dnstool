# Lan Cache dnstool utility

## Notes

This application has been created to replace the bash scripts which are utilised to bootstrap the various containers required to deploy the Lan Cache stack.

Another intention of the application is to simplify bootstrapping for IPv6 configuration generation.


## Usage

An example usage to generate configuration for the `lancache-dns` container would be: `dnstool generate lancache-dns`.

Executing the application with no sub-commands/arguments results in the following output:

```text
A replacement utility for the configuration generator bash script:
https://github.com/lancachenet/lancache-dns/blob/d626a74c02c7a8383eeaaab493fcdffe536aea95/overlay/hooks/entrypoint-pre.d/10_generate_config.sh
utilised to generate configuration for lancache-dns containers

Usage:
  dnstool [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  generate    Generate configuration for lancache container(s)
  help        Help about any command

Flags:
  -h, --help   help for dnstool

Use "dnstool [command] --help" for more information about a command.
```

Similarly, the output for the `generate` sub-command results in the following output:

```text
Generate and manipulate configuration files for lancache container(s)

Usage:
  dnstool generate [command]

Available Commands:
  lancache-dns Generate configuration for lancache-dns container

Flags:
  -h, --help   help for generate

Use "dnstool generate [command] --help" for more information about a command.
```
