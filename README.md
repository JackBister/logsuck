# Logsuck

![CI Status](https://img.shields.io/github/workflow/status/jackbister/logsuck/ci)
![Apache License 2.0](https://img.shields.io/github/license/jackbister/logsuck)

Logsuck is a program that makes it easier for you to deal with your logs. It serves a similar purpose to Splunk or the ELK stack. You can use it to search your logs, correlate logs from different log files with each other and set up alerts and dashboards based on your logs. On top of that, Logsuck is extremely simple to deploy, and is free.

Logsuck does not currently aim to be able to ingest millions of events per second or to have every enterprise feature you can imagine. The target audience for Logsuck is smaller development teams or solo developers who need a powerful tool that is simple to set up and keep running.

Logsuck is currently pre-1.0. This means that there may be changes made to the database schemas used internally by Logsuck which could cause incompatibilities between versions. If that occurs, you will need to delete your `logsuck.db` file before you can use the new version, which will result in your existing search index being deleted. You can work around this by running the new version in parallel with the old version for a while, until you feel like the data that is only in the old database is old enough to be irrelevant. After 1.0, if a schema change is made, a migration will be provided to handle the upgrade gracefully to the extent that it is possible.

![a screenshot of the Logsuck GUI](https://jackbister.com/content/logsuck_v0_gui.png)

1. [Getting started](#getting-started)
   - [Single mode](#single-mode)
   - [Forwarder/Recipient mode](#forwarderrecipient-mode)
2. [Configuration](#configuration)
   - [Command line options](#command-line-options)
   - [JSON configuration](#json-configuration)
3. [Need help?](#need-help)
4. [Contributing](#contributing)
5. [Upcoming features](#upcoming-features)
   - [Before version 1.0](#before-version-10)
   - [After version 1.0](#after-version-10)

## Getting started

Getting started with Logsuck is easy. Logsuck can be ran in two different modes: Single mode or forwarder/recipient mode.

Single mode means you run Logsuck on a server which will both handle reading your log files and serving the Logsuck GUI. This is useful if you are running all of your applications on a single server, or you can even use it to run Logsuck locally to read logs while developing.

Forwarder/recipient mode means you have one recipient instance which will receive events over the network from several forwarder instances. The recipient will also serve the Logsuck GUI. This is useful if you have your applications spread out across multiple servers but want to be able to search them all in one central place.

Note that the different modes of Logsuck do not require different builds of Logsuck. All Logsuck executables are capable of running in single mode, or running as forwarders or recipients.

The guides below assume that you are using a Linux distribution, but Logsuck executables are available for Windows as well.

### Single mode

To run Logsuck in single mode, start off by downloading the Logsuck executable to the server:

```sh
wget https://github.com/JackBister/logsuck/releases/download/v0.0.0/logsuck-v0_0_0-x64-linux.zip
unzip logsuck-v0_0_0-x64-linux.zip
```

Then, create a configuration file. In this case we will mostly use default settings to read Logsuck's own logs:

```sh
echo '{ "files": [ "fileName": "logsuck.txt" ] }' > logsuck.json
```

Finally, start Logsuck:

```sh
./logsuck > logsuck.txt
```

Logsuck is now running. Navigate to http://localhost:8080 and you should see the GUI. Try pressing the search button while leaving the search field empty and you should find some events. Congratulations, you are now running Logsuck in single mode! Continue reading to learn more about configuring Logsuck.

### Forwarder/Recipient mode

Setting up Logsuck in forwarder/recipient mode is slightly more complicated. Lets again start off by downloading the Logsuck executable:

```sh
wget https://github.com/JackBister/logsuck/releases/download/v0.0.0/logsuck-v0_0_0-x64-linux.zip
unzip logsuck-v0_0_0-x64-linux.zip
```

This time, we will create two different directories which will be the working directories for each of the two Logsuck instances we need to run.

```sh
mkdir forwarder
mkdir recipient
```

Now, create the configuration for the recipient instance. This configuration will set Logsuck up to listen for events from Logsuck forwarders on port 9000.

```sh
echo '{ "recipient": { "enabled": true, "address": ":9000" } }' > ./recipient/logsuck.json
```

And create the configuration for the forwarder instance. This will configure the forwarder to read its own log and send its events to port 9000 on localhost, which is where the recipient will be running.

```sh
echo '{ "files": [ "fileName": "logsuck-forwarder.txt" ], "forwarder": { "enabled": true, "recipientAddress": "http://localhost:9000" } }' > ./forwarder/logsuck.json
```

Start the recipient:

```sh
cd recipient && ../logsuck > logsuck-recipient.txt
```

Open a new shell and start the forwarder:

```sh
cd forwarder && ../logsuck > logsuck-forwarder.txt
```

You should now be able to navigate to http://localhost:8080 in the browser and see the GUI served by the recipient instance. If you leave the search field empty and press the search button, you should see events show up. If you look at the "source" field underneath the events, you should see that it is always "logsuck-forwarder.txt", confirming that they were sent by the forwarder. You are now running Logsuck in forwarder/recipient mode! Continue reading to learn more about configuring Logsuck.

## Configuration

### Command line options

There are a few configuration options available on the command line when running Logsuck. It is recommended that you use JSON configuration for production usage of Logsuck, but these options may be useful when running Logsuck locally on a single file.

`-config <filename>`
The name of the file containing the configuration for Logsuck. If a config file exists, all other command line configuration will be ignored. (default "logsuck.json")

`-dbfile <filename>`
The name of the file in which Logsuck will store its data. If the name ':memory:' is used, no file will be created and everything will be stored in memory. If the file does not exist, a new file will be created. (default "logsuck.db")

`-delimiter <regex>`
The delimiter between events in the log. Usually \n. (default "\n")

`-fieldextractor <regex>`
A regular expression which will be used to extract field values from events. Can be given in two variants:

1. An expression containing any number of named capture groups. The names of the capture groups will be used as the field names and the captured strings will be used as the values.
2. An expression with two unnamed capture groups. The first capture group will be used as the field name and the second group as the value.

If a field with the name '\_time' is extracted and matches the given timelayout, it will be used as the timestamp of the event. Otherwise the time the event was read will be used.

Multiple extractors can be specified by using the fieldextractor flag multiple times. (defaults "(\w+)=(\w+)" and "(?P<\_time>\d\d\d\d/\d\d/\d\d \d\d:\d\d:\d\d.\d\d\d\d\d\d)")

`-help`
Print information about command line options and quit.

`-timelayout <string>`
The layout of the timestamp which will be extracted in the \_time field. For more information on how to write a timelayout and examples, see https://golang.org/pkg/time/#Parse and https://golang.org/pkg/time/#pkg-constants. (default "2006/01/02 15:04:05")

`-version`
Print version info and quit.

`-webaddr <string>`
The address on which the search GUI will be exposed. (default ":8080")

### JSON configuration

JSON is the recommended way of configuring Logsuck for more complex usage. By default, Logsuck will look in its working directory for a `logsuck.json` file which will contain the configuration. If the file is found, all command line options will be ignored. There is a JSON schema which documents the configuration file available [here](https://github.com/JackBister/logsuck/blob/master/logsuck-config.schema.json).

## Need help?

If you have any questions about using Logsuck after reading the documentation, please [create an issue](https://github.com/JackBister/logsuck/issues/new) on this repository! There are no stupid questions here. You asking a question will help improve the documentation for everyone, so it is very much appreciated!

## Contributing

Contributions in the form of pull requests and issues are welcome! Here are some tips on running Logsuck locally for developing.

First of all, you will need a version of Go with support for Go modules (1.14 is used when building in CI), as well as a fairly recent Node/npm installation (Node v12 and npm v6 are used in CI). If cross compiling for Windows, you need [gcc-mingw-w64](http://mingw-w64.org/doku.php/download).

Logsuck consists of two parts: A Go application for the backend and a [preact](https://preactjs.com/) GUI for the frontend. When the Go application is built, the frontend is bundled into the executable using [vfsgen](https://github.com/shurcooL/vfsgen). This bundling step allows Logsuck to be deployed as a single executable.

Since the frontend is necessary to build the backend, lets start off by building the frontend. All frontend code is contained in the `./web/static` directory.

```sh
cd web/static
npm install && npm run build
```

This will build the frontend and put it in the `./web/static/dist` directory. Time to build the backend.

It is easier to build the backend on Linux than it is for Windows. If using Windows, I would recommend using Windows Subsystem for Linux and cross compiling to a Windows executable instead of trying to build directly on Windows. The official release builds for Windows are cross compiled from an Ubuntu installation.

Whether you're compiling for Linux or cross compiling for Windows, the first step is to use vfsgen to generate a bundle of the frontend files which will be embedded in the executable.

```sh
go generate ./cmd/logsuck/main.go
```

Then, if compiling for Linux, all you need to do is:

```sh
go build ./cmd/logsuck/main.go
```

to create a `./main` file you can run.

If cross compiling for Windows, try the following instead:

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build ./cmd/logsuck/main.go
```

and you should have a `./main.exe` file.

If you are working on the frontend, you can do the following things to make your life easier:

1. Run webpack in watch mode by running `npm run watch` in `./web/static`
2. Add `{ "web": { "usePackagedFiles": "false" } }` to your `./logsuck.json`. This will make it so that Logsuck will read the files off the filesystem instead of using the files that were bundled into the executable by running go generate.

With those two steps done, you don't need to restart Logsuck to see the changes you've made to the frontend, and your changes will be compiled when you save. You just need to refresh your browser after making a change to see it in action.

## Upcoming features

Logsuck is still heavily in development, so there are many features still being worked on as we race towards version 1.0.

### Before version 1.0

- Glob patterns to find log files
- Compression for the FTS table to reduce storage requirements
- Retention setting to delete old events after a certain period of time
- "Show source" / "Show context" button to view events from the same source that are close in time to the selected event
- Ability to search via time spans that are not relative to the current time, such as "All events between 2020-01-01 and 2020-01-05"
- Ad hoc field extraction using pipes in the search command (equivalent to Splunk's "| rex")
- E-mail alerts

### After version 1.0

- Authentication and authorization
- Creating tables and charts from searches
- Dashboards
- Plugins
- Advanced search commands like dedup, eval, transaction, etc.
- Support for an external database other than logsuck.db
- [Your favorite feature](https://github.com/JackBister/logsuck/issues/new)
- ... And many more great things!
