# Configuration

## Static configuration and dynamic configuration

There are two types of configuration in Logsuck: static configuration and dynamic configuration. Static configuration is configuration which is given when the program starts using the command line or a configuration file. Dynamic configuration is configuration which can be changed using the GUI while Logsuck is running. When running in forwarder/recipient mode, the dynamic configuration on the forwaders will be retrieved from the recipient, meaning that you only need to update the dynamic configuration on the recipient.

### Static-only configuration

All configuration properties can be modified from the GUI, but there are some configuration properties which are _only_ read from the static configuration, meaning that the value displayed in the GUI may not reflect the actual configuration. In most cases this limitation is applied because using dynamic configuration would not work well in forwarder/recipient setups. The following properties can only be set through static configuration:

- `forceStaticConfig`
- `forwarder`
- `host`
- `recipient`
- `web`

## Configuration schema

A schema for the JSON configuration file is available [here](../logsuck-config.schema.json). This schema is valid for the core Logsuck build. If you customize Logsuck using [plugins](./Plugins.md) you can generate a schema matching your setup by running Logsuck with the `-schema` command line parameter. Using `-schema` will dump the current JSON schema to stdout.

## Initial configuration

When Logsuck is run for the first time, it will create a database file containing its data. Included in this data is the dynamic configuration. When running for the first time, the static configuration will be written into the database file as dynamic configuration, and from then on the dynamic configuration will be used for everything except the static-only configuration properties mentioned above.

What this means is that after running Logsuck for the first time, all configuration (except for static-only configuration) must be done through the GUI, since the configuration stored in the database file will override any configuration in the command line or configuration file.

If you want to avoid this behavior and exclusively configure Logsuck using the command line or configuration file, you can use the `-forceStaticConfig true` command line option or set `forceStaticConfig` to true in the configuration file. When this option is set, the static configuration will not be copied into the database, and configuration will never be read from the database. The config page in the GUI will show the static configuration and will not allow you to edit anything.
