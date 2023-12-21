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
	postgres_common.Plugin,
	postgres_events.Plugin,
	postgres.Plugin,
	steps.Plugin,
	tasks.Plugin,
}
