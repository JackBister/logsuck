# Tasks

Tasks are maintenance programs which run inside Logsuck to perform actions like removing old events, keeping statistics, etc.

Tasks are configured using the `tasks` key in `logsuck.json`. An example task configuration may look like this:

```json
"tasks": {
    "tasks": [
        {
            "name": "@logsuck/DeleteOldEventsTask",
            "enabled": true,
            "interval": "1m",
            "config": [
                {
                    "key": "minAge",
                    "value": "1d"
                }
            ]
        }
    ]
},
```

Every task must have a `name`, `enabled` and `interval` property. The available task names are listed below. `interval` can only be specified in seconds, minutes or hours.

**If there is no configuration for a task, the task will never run.** Make sure you read through the available tasks below and configure the ones that are relevant to you.

The `config` property is specific to each task. See the description of the tasks below to find out which config properties are available for the task.

# List of available tasks

## @logsuck/DeleteOldEventsTask

DeleteOldEventsTask removes events that are older than a certain age from the Logsuck database. This is useful for a few reasons:

- Preventing database size from growing out of control
- You may not want to keep logs forever for compliance and privacy reasons

If you enable this task you should run it frequently. If you run it too infrequently you may risk a situation where the task needs to delete too many events and causing long lasting locks in the database which slow other parts of Logsuck down.

### Available config properties

#### minAge

Sets the minimum age of the events to delete. Any event older than this will be deleted. If `minAge` is not set or cannot be parsed this task will not do anything.

The format of `minAge` is `<number><unit>` where number is a positive integer and `unit` is one of `s`, `m`, `h`, `d`, `M`, or `y`.

For example, `"minAge": "1d"` means that the task should delete any event older than 24 hours. `"minAge": "2M"` means that the task should delete any event older than 60 days.
