package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed "logsuck-config.schema.json"
var schemaString string

func GetBaseSchema() (map[string]any, error) {
	ret := map[string]any{}
	err := json.Unmarshal([]byte(schemaString), &ret)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal logsuck JSON schema: %w", err)
	}
	return ret, nil
}

func MergeSchemas(a, b map[string]any) map[string]any {
	m := map[string]any{}
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}
	return m
}

func CreateSchema(pluginSchemas, taskSchemas map[string]any) (map[string]any, error) {
	configSchema, err := GetBaseSchema()
	if err != nil {
		return nil, err
	}
	ps := map[string]any{
		"type": "object",
		"autoform": map[string]any{
			"displayAsArray": true,
		},
		"properties": map[string]any{},
	}
	ps["properties"] = MergeSchemas(ps["properties"].(map[string]any), pluginSchemas)
	configSchema["properties"] = MergeSchemas(configSchema["properties"].(map[string]any), map[string]any{
		"plugins": ps,
	})

	tc := map[string]any{}
	for k, v := range taskSchemas {
		tc[k] = map[string]any{
			"type": "object",
			"properties": map[string]any{
				"enabled": map[string]any{
					"type": "boolean",
				},
				"interval": map[string]any{
					"type": "string",
				},
				"config": v,
			},
		}
	}

	tcm := map[string]any{}
	tcm["tasks"] = map[string]any{
		"type": "object",
		"autoform": map[string]any{
			"displayAsArray": true,
		},
		"properties": tc,
	}
	configSchema["properties"] = MergeSchemas(configSchema["properties"].(map[string]any), tcm)
	configSchema["tasks"] = tcm

	return configSchema, nil
}
