// Copyright 2021 Jack Bister
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

type Config struct {
	HostName          string
	HostType          string
	ForceStaticConfig bool

	Forwarder *ForwarderConfig
	Recipient *RecipientConfig

	SQLite *SqliteConfig

	Web *WebConfig

	Files     map[string]FileConfig
	FileTypes map[string]FileTypeConfig
	HostTypes map[string]HostTypeConfig
	Tasks     TasksConfig
}
