# Relay

## Description

Command Line Interface of Relay for Blockchain Transmission Protocol

## Usage

` relay [flags] `

### Options

| Name,shorthand          | Environment Variable        | Required | Default | Description                                                 |
|-------------------------|-----------------------------|----------|---------|-------------------------------------------------------------|
| --base_dir              | RELAY_BASE_DIR              | false    |         | Base directory for data                                     |
| --src_config            | RELAY_SOURCE_CONFIG         | false    |         | Source network configuration                                |
| --dst_config            | RELAY_DESTINATION_CONFIG    | false    |         | Destination network configuration                           |
| --direction             | RELAY_DIRECTION             | false    |         | Relay network direction (both,front,reverse)                |
| --config, -c            | RELAY_CONFIG                | false    |         | Parsing configuration file                                  |
| --console_level         | RELAY_CONSOLE_LEVEL         | false    | trace   | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --log_forwarder.address | RELAY_LOG_FORWARDER_ADDRESS | false    |         | LogForwarder address                                        |
| --log_forwarder.level   | RELAY_LOG_FORWARDER_LEVEL   | false    | info    | LogForwarder level                                          |
| --log_forwarder.name    | RELAY_LOG_FORWARDER_NAME    | false    |         | LogForwarder name                                           |
| --log_forwarder.options | RELAY_LOG_FORWARDER_OPTIONS | false    | []      | LogForwarder options, comma-separated 'key=value'           |
| --log_forwarder.vendor  | RELAY_LOG_FORWARDER_VENDOR  | false    |         | LogForwarder vendor (fluentd,logstash)                      |
| --log_level             | RELAY_LOG_LEVEL             | false    | debug   | Global log level (trace,debug,info,warn,error,fatal,panic)  |
| --log_writer.compress   | RELAY_LOG_WRITER_COMPRESS   | false    | false   | Use gzip on rotated log file                                |
| --log_writer.filename   | RELAY_LOG_WRITER_FILENAME   | false    |         | Log file name (rotated files resides in same directory)     |
| --log_writer.localtime  | RELAY_LOG_WRITER_LOCALTIME  | false    | false   | Use localtime on rotated log file instead of UTC            |
| --log_writer.maxage     | RELAY_LOG_WRITER_MAXAGE     | false    | 0       | Maximum age of log file in day                              |
| --log_writer.maxbackups | RELAY_LOG_WRITER_MAXBACKUPS | false    | 0       | Maximum number of backups                                   |
| --log_writer.maxsize    | RELAY_LOG_WRITER_MAXSIZE    | false    | 100     | Maximum log file size in MiB                                |

### Child commands

| Command                         | Description         |
|---------------------------------|---------------------|
| [relay save](#RELAY-save)       | Save configuration  |
| [relay start](#RELAY-start)     | Start server        |
| [relay version](#RELAY-version) | Print relay version |

## Relay save

### Description

Save configuration

### Usage

` relay save [file] [flags] `

### Inherited Options

| Name,shorthand          | Environment Variable        | Required | Default | Description                                                 |
|-------------------------|-----------------------------|----------|---------|-------------------------------------------------------------|
| --base_dir              | RELAY_BASE_DIR              | false    |         | Base directory for data                                     |
| --src_config            | RELAY_SOURCE_CONFIG         | false    |         | Source network configuration                                |
| --dst_config            | RELAY_DESTINATION_CONFIG    | false    |         | Destination network configuration                           |
| --direction             | RELAY_DIRECTION             | false    |         | Relay network direction (both,front,reverse)                |
| --config, -c            | RELAY_CONFIG                | false    |         | Parsing configuration file                                  |
| --console_level         | RELAY_CONSOLE_LEVEL         | false    | trace   | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --log_forwarder.address | RELAY_LOG_FORWARDER_ADDRESS | false    |         | LogForwarder address                                        |
| --log_forwarder.level   | RELAY_LOG_FORWARDER_LEVEL   | false    | info    | LogForwarder level                                          |
| --log_forwarder.name    | RELAY_LOG_FORWARDER_NAME    | false    |         | LogForwarder name                                           |
| --log_forwarder.options | RELAY_LOG_FORWARDER_OPTIONS | false    | []      | LogForwarder options, comma-separated 'key=value'           |
| --log_forwarder.vendor  | RELAY_LOG_FORWARDER_VENDOR  | false    |         | LogForwarder vendor (fluentd,logstash)                      |
| --log_level             | RELAY_LOG_LEVEL             | false    | debug   | Global log level (trace,debug,info,warn,error,fatal,panic)  |
| --log_writer.compress   | RELAY_LOG_WRITER_COMPRESS   | false    | false   | Use gzip on rotated log file                                |
| --log_writer.filename   | RELAY_LOG_WRITER_FILENAME   | false    |         | Log file name (rotated files resides in same directory)     |
| --log_writer.localtime  | RELAY_LOG_WRITER_LOCALTIME  | false    | false   | Use localtime on rotated log file instead of UTC            |
| --log_writer.maxage     | RELAY_LOG_WRITER_MAXAGE     | false    | 0       | Maximum age of log file in day                              |
| --log_writer.maxbackups | RELAY_LOG_WRITER_MAXBACKUPS | false    | 0       | Maximum number of backups                                   |
| --log_writer.maxsize    | RELAY_LOG_WRITER_MAXSIZE    | false    | 100     | Maximum log file size in MiB                                |

### Parent command

| Command         | Description   |
|-----------------|---------------|
| [relay](#RELAY) | BTP Relay CLI |

### Related commands

| Command                         | Description         |
|---------------------------------|---------------------|
| [relay save](#relay-save)       | Save configuration  |
| [relay start](#relay-start)     | Start server        |
| [relay version](#relay-version) | Print relay version |

## Relay start

### Description

Start server

### Usage

` RELAY start [flags] `

### Inherited Options

| Name,shorthand          | Environment Variable        | Required | Default | Description                                                 |
|-------------------------|-----------------------------|----------|---------|-------------------------------------------------------------|
| --base_dir              | RELAY_BASE_DIR              | false    |         | Base directory for data                                     |
| --src_config            | RELAY_SOURCE_CONFIG         | false    |         | Source network configuration                                |
| --dst_config            | RELAY_DESTINATION_CONFIG    | false    |         | Destination network configuration                           |
| --direction             | RELAY_DIRECTION             | false    |         | Relay network direction (both,front,reverse)                |
| --config, -c            | RELAY_CONFIG                | false    |         | Parsing configuration file                                  |
| --console_level         | RELAY_CONSOLE_LEVEL         | false    | trace   | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --log_forwarder.address | RELAY_LOG_FORWARDER_ADDRESS | false    |         | LogForwarder address                                        |
| --log_forwarder.level   | RELAY_LOG_FORWARDER_LEVEL   | false    | info    | LogForwarder level                                          |
| --log_forwarder.name    | RELAY_LOG_FORWARDER_NAME    | false    |         | LogForwarder name                                           |
| --log_forwarder.options | RELAY_LOG_FORWARDER_OPTIONS | false    | []      | LogForwarder options, comma-separated 'key=value'           |
| --log_forwarder.vendor  | RELAY_LOG_FORWARDER_VENDOR  | false    |         | LogForwarder vendor (fluentd,logstash)                      |
| --log_level             | RELAY_LOG_LEVEL             | false    | debug   | Global log level (trace,debug,info,warn,error,fatal,panic)  |
| --log_writer.compress   | RELAY_LOG_WRITER_COMPRESS   | false    | false   | Use gzip on rotated log file                                |
| --log_writer.filename   | RELAY_LOG_WRITER_FILENAME   | false    |         | Log file name (rotated files resides in same directory)     |
| --log_writer.localtime  | RELAY_LOG_WRITER_LOCALTIME  | false    | false   | Use localtime on rotated log file instead of UTC            |
| --log_writer.maxage     | RELAY_LOG_WRITER_MAXAGE     | false    | 0       | Maximum age of log file in day                              |
| --log_writer.maxbackups | RELAY_LOG_WRITER_MAXBACKUPS | false    | 0       | Maximum number of backups                                   |
| --log_writer.maxsize    | RELAY_LOG_WRITER_MAXSIZE    | false    | 100     | Maximum log file size in MiB                                |

### Parent command

| Command         | Description   |
|-----------------|---------------|
| [relay](#RELAY) | BTP Relay CLI |

### Related commands

| Command                         | Description         |
|---------------------------------|---------------------|
| [relay save](#relay-save)       | Save configuration  |
| [relay start](#relay-start)     | Start server        |
| [relay version](#relay-version) | Print relay version |

## Relay version

### Description

Print relay version

### Usage

` relay version `

### Inherited Options

| Name,shorthand          | Environment Variable        | Required | Default | Description                                                 |
|-------------------------|-----------------------------|----------|---------|-------------------------------------------------------------|
| --base_dir              | RELAY_BASE_DIR              | false    |         | Base directory for data                                     |
| --src_config            | RELAY_SOURCE_CONFIG         | false    |         | Source network configuration                                |
| --dst_config            | RELAY_DESTINATION_CONFIG    | false    |         | Destination network configuration                           |
| --direction             | RELAY_DIRECTION             | false    |         | Relay network direction (both,front,reverse)                |
| --config, -c            | RELAY_CONFIG                | false    |         | Parsing configuration file                                  |
| --console_level         | RELAY_CONSOLE_LEVEL         | false    | trace   | Console log level (trace,debug,info,warn,error,fatal,panic) |
| --log_forwarder.address | RELAY_LOG_FORWARDER_ADDRESS | false    |         | LogForwarder address                                        |
| --log_forwarder.level   | RELAY_LOG_FORWARDER_LEVEL   | false    | info    | LogForwarder level                                          |
| --log_forwarder.name    | RELAY_LOG_FORWARDER_NAME    | false    |         | LogForwarder name                                           |
| --log_forwarder.options | RELAY_LOG_FORWARDER_OPTIONS | false    | []      | LogForwarder options, comma-separated 'key=value'           |
| --log_forwarder.vendor  | RELAY_LOG_FORWARDER_VENDOR  | false    |         | LogForwarder vendor (fluentd,logstash)                      |
| --log_level             | RELAY_LOG_LEVEL             | false    | debug   | Global log level (trace,debug,info,warn,error,fatal,panic)  |
| --log_writer.compress   | RELAY_LOG_WRITER_COMPRESS   | false    | false   | Use gzip on rotated log file                                |
| --log_writer.filename   | RELAY_LOG_WRITER_FILENAME   | false    |         | Log file name (rotated files resides in same directory)     |
| --log_writer.localtime  | RELAY_LOG_WRITER_LOCALTIME  | false    | false   | Use localtime on rotated log file instead of UTC            |
| --log_writer.maxage     | RELAY_LOG_WRITER_MAXAGE     | false    | 0       | Maximum age of log file in day                              |
| --log_writer.maxbackups | RELAY_LOG_WRITER_MAXBACKUPS | false    | 0       | Maximum number of backups                                   |
| --log_writer.maxsize    | RELAY_LOG_WRITER_MAXSIZE    | false    | 100     | Maximum log file size in MiB                                |

### Parent command

| Command         | Description   |
|-----------------|---------------|
| [relay](#RELAY) | BTP Relay CLI |

### Related commands

| Command                         | Description         |
|---------------------------------|---------------------|
| [relay save](#RELAY-save)       | Save configuration  |
| [relay start](#RELAY-start)     | Start server        |
| [relay version](#RELAY-version) | Print RELAY version |

