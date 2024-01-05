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

package postgres

import (
	"log/slog"

	"github.com/jackbister/logsuck/pkg/logsuck"

	"go.uber.org/dig"
)

var Plugin = logsuck.Plugin{
	Name: "@logsuck/postgres",
	Provide: func(c *dig.Container, logger *slog.Logger) error {
		err := c.Provide(NewPostgresConfigRepository)
		if err != nil {
			return err
		}
		err = c.Provide(NewPostgresJobRepository)
		if err != nil {
			return err
		}
		return nil
	},
}
