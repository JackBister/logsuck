package dependencyinjection

import (
	"github.com/jackbister/logsuck/pkg/logsuck"
	"github.com/jackbister/logsuck/plugins/postgres"
	"github.com/jackbister/logsuck/plugins/sqlite_common"
	"github.com/jackbister/logsuck/plugins/sqlite_events"
	"github.com/jackbister/logsuck/plugins/steps"
	"github.com/jackbister/logsuck/plugins/tasks"
)

var usedPlugins = []logsuck.Plugin{
	postgres.Plugin,
	sqlite_common.Plugin,
	sqlite_events.Plugin,
	steps.Plugin,
	tasks.Plugin,
}
