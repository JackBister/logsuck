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

	"github.com/jackbister/logsuck/pkg/logsuck/config"
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
	configSchemaProperties := configSchema["properties"].(map[string]any)
	ps := map[string]any{
		"type": "object",
		"autoform": map[string]any{
			"displayAsArray": true,
		},
		"properties": map[string]any{},
	}
	pluginSchemasToDelete := []string{}
	for k, v := range pluginSchemas {
		if k2, ok := config.CorePlugins[k]; ok {
			pluginSchemasToDelete = append(pluginSchemasToDelete, k)
			if v2, ok := configSchemaProperties[k2]; ok {
				existingSchema := v2.(map[string]any)
				existingSchemaProperties := existingSchema["properties"].(map[string]any)
				existingSchema["properties"] = MergeSchemas(existingSchemaProperties, v.(map[string]any)["properties"].(map[string]any))
				configSchemaProperties[k2] = existingSchema
			} else {
				configSchemaProperties[k2] = v
			}
		}
	}
	for _, v := range pluginSchemasToDelete {
		delete(pluginSchemas, v)
	}
	if len(pluginSchemas) > 0 {
		ps["properties"] = MergeSchemas(ps["properties"].(map[string]any), pluginSchemas)
		configSchemaProperties = MergeSchemas(configSchemaProperties, map[string]any{
			"plugins": ps,
		})
	}

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
	configSchema["properties"] = MergeSchemas(configSchemaProperties, tcm)
	configSchema["tasks"] = tcm

	return configSchema, nil
}
