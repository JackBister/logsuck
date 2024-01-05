// Copyright 2024 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
