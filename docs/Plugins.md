# Plugins

Plugins can be used to customize Logsuck by replacing components or adding new behaviors.

Plugins are included into Logsuck at build time. The plugins which are included can be found in the `internal/dependencyinjection/UsedPlugins.go` file.

The following plugins are included in the default Logsuck build:

- `sqlite_common`: Contains SQLite infrastructure used by the other SQLite plugins.
- `sqlite_config`: Contains an SQLite implementation of configuration management.
- `sqlite_events`: Contains the SQLite implementation of event storage and full text search.
- `sqlite_jobs`: Contains an SQLite implementation of jobs management.
- `steps`: Contains core Logsuck search pipeline steps.
- `tasks`: Contains core Logsuck tasks.

In addition, the following plugins are included in the Logsuck repository. These are not meant to be part of the Logsuck core, but are provided as a proof of concept for the plugin system:

- `postgres`: Contains PostgreSQL implementations of job and configuration management.
- `postgres_common`: Contains PostgreSQL infrastructure used by the `postgres` and `postgres_events` plugins.
- `postgres_events`: Contains PostgreSQL implementations of event storage and full text search.

# What can be customized using plugins?

The following aspects of Logsuck can be customized:

- A new pipeline step can be added by providing a constructor returning `pipeline.StepDefinition` and using `dig.Group("tasks")`
- A new task can be added by providing a constructor returning `tasks.Task` and using `dig.Group("tasks")`
- The configuration repository can be replaced by providing a constructor returning `config.ConfigRepository`
- The job repository can be replaced by providing a constructor returning `jobs.Repository`
- The events repository (including the full text search) can be replaced by providing a constructor returning `events.Repository`

# Creating a plugin

As an example of how plugins can extend Logsuck's functionality, here is a step by step guide to implementing a plugin which adds a new pipeline step. The new step will filter events, only matching events containing a string from the plugin's configuration.

## Implement the pipeline step

First off, create a new directory in the `plugins` directory called `my_plugin`. This directory will contain all code related to the plugin.

Create a file called `MyPluginStep.go` in the `my_plugin` directory and add the following code:

```go
package my_plugin

import (
	"context"
	"strings"

	"github.com/jackbister/logsuck/pkg/logsuck/events"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"
)

type MyPluginStep struct {
	filter string
}

func (p *MyPluginStep) Execute(ctx context.Context, pipe pipeline.PipelinePipe, params pipeline.PipelineParameters) {
	defer close(pipe.Output)

	for {
		select {
		case <-ctx.Done():
			return
		case res, ok := <-pipe.Input:
			if !ok {
				return
			}

			output := []events.EventWithExtractedFields{}
			for _, evt := range res.Events {
				if strings.Contains(evt.Raw, p.filter) {
					output = append(output, evt)
				}
			}
			pipe.Output <- pipeline.PipelineStepResult{Events: output}
		}
	}
}

func (p *MyPluginStep) Name() string {
	return "MyPluginStep"
}

func (p *MyPluginStep) InputType() pipeline.PipelinePipeType {
	return pipeline.PipelinePipeTypeEvents
}

func (p *MyPluginStep) OutputType() pipeline.PipelinePipeType {
	return pipeline.PipelinePipeTypeEvents
}
```

The MyPluginStep struct implements the PipelineStep interface which can be found in `pkg/logsuck/pipeline/Pipeline.go`. Plugins should only depend on code contained in the `pkg` directory, never on code contained in `internal`.

## Create the configuration schema

To make the plugin configurable, a schema must be provided for Logsuck to show in its GUI. Create a file called `my_plugin.schema.json` in the `my_plugin` directory containing the following:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "my_plugin",
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "filter": {
      "description": "The string used to filter events in MyPluginStep.",
      "type": "string"
    }
  }
}
```

## Provide the plugin

Providing the plugin to Logsuck is done by creating an instance of the `logsuck.Plugin` struct and adding it to the `UsedPlugins.go` file in the `internal/dependencyinjection` directory.

Create a file called `MyPlugin.go` in the `my_plugin` directory and add the following:

```go
package my_plugin

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/pkg/logsuck/config"
	"github.com/jackbister/logsuck/pkg/logsuck/pipeline"

	"go.uber.org/dig"
)

const pluginName = "my_plugin"

//go:embed my_plugin.schema.json
var schemaString string

type Config struct {
	Filter string
}

var Plugin = logsuck.Plugin{
	Name: pluginName,
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(func(configSource config.ConfigSource) pipeline.StepDefinition {
			return pipeline.StepDefinition{
				StepName: "MyPluginStep",
				Compiler: func(input string, options map[string]string) (pipeline.PipelineStep, error) {
					cfg, err := GetConfig(configSource)
					if err != nil {
						return nil, err
					}
					return &MyPluginStep{
						filter: cfg.Filter,
					}, nil
				},
			}
		}, dig.Group("steps"))
		if err != nil {
			return err
		}
		return nil
	},
	JsonSchema: func() (map[string]any, error) {
		ret := map[string]any{}
		err := json.Unmarshal([]byte(schemaString), &ret)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal my_plugin JSON schema: %w", err)
		}
		return ret, nil
	},
}

func GetConfig(configSource config.ConfigSource) (*Config, error) {
	cfg, err := configSource.Get()
	if err != nil {
		return nil, err
	}
	ret := Config{}

	cfgMap, ok := cfg.Cfg.Plugins[pluginName].(map[string]any)
	if ok {
		if cs, ok := cfgMap["filter"].(string); ok {
			ret.Filter = cs
		}
	}
	if ret.Filter == "" {
		return nil, fmt.Errorf("got empty filter in my_plugin configuration")
	}
	return &ret, nil
}
```

This file provides the JSON schema for the plugin and makes the step available to Logsuck. When the step is used in a search, the filter string is retrieved from the configuration and used by the plugin step.

Next up, add the plugin to `internal/dependencyinjection/UsedPlugins.go`:

```go
package dependencyinjection

import (
	"github.com/jackbister/logsuck/plugins/my_plugin"
    // ...
)

var usedPlugins = []logsuck.Plugin{
	my_plugin.Plugin,
    // ...
}
```

## Test the plugin

Run Logsuck using `go run cmd/logsuck/main.go -json log-logsuck.txt > log-logsuck.txt`. You should see a line like this in the `log-logsuck.txt` file confirming that Logsuck is aware of your plugin:

```
{"time":"2024-01-05T16:18:02.8365042+01:00","level":"INFO","msg":"Loading plugin","pluginName":"my_plugin"}
```

Open your browser and go to `http://localhost:8080/config`. Click on `plugins` in the left hand menu and you should see a list of plugins, including `my_plugin`. `my_plugin` contains a text input field named `filter` because of the JSON schema it provides. Set the `filter` field to `Starting` and click `Save`.

Go back to the search page and run a search like: `| MyPluginStep`. The results page should contain an event with the message "Starting Web GUI", confirming that MyPluginStep successfully filters out events based on the `filter` configuration property. Try changing the `filter` property on the configuration page and running more searches to verify that it's working.
