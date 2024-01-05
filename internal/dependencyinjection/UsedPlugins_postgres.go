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

//go:build postgres
// +build postgres

package dependencyinjection

import (
	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/plugins/postgres"
	"github.com/jackbister/logsuck/plugins/postgres_common"
	"github.com/jackbister/logsuck/plugins/postgres_events"
	"github.com/jackbister/logsuck/plugins/steps"
	"github.com/jackbister/logsuck/plugins/tasks"
)

var usedPlugins = []logsuck.Plugin{
	postgres.Plugin,
	postgres_common.Plugin,
	postgres_events.Plugin,
	steps.Plugin,
	tasks.Plugin,
}
