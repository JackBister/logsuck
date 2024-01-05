//go:build !postgres
// +build !postgres

package dependencyinjection

import (
	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/plugins/sqlite_common"
	"github.com/jackbister/logsuck/plugins/sqlite_config"
	"github.com/jackbister/logsuck/plugins/sqlite_events"
	"github.com/jackbister/logsuck/plugins/sqlite_jobs"
	"github.com/jackbister/logsuck/plugins/steps"
	"github.com/jackbister/logsuck/plugins/tasks"
)

var usedPlugins = []logsuck.Plugin{
	sqlite_common.Plugin,
	sqlite_config.Plugin,
	sqlite_events.Plugin,
	sqlite_jobs.Plugin,
	steps.Plugin,
	tasks.Plugin,
}
