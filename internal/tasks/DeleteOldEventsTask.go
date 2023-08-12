// Copyright 2022 Jack Bister
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

package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"github.com/jackbister/logsuck/internal/events"
	"github.com/jackbister/logsuck/internal/search"
)

type DeleteOldEventsTask struct {
	Repo events.Repository

	Logger *slog.Logger
}

func NewDeleteOldEventsTask(repo events.Repository, logger *slog.Logger) Task {
	return &DeleteOldEventsTask{
		Repo:   repo,
		Logger: logger,
	}
}

func (t *DeleteOldEventsTask) Name() string {
	return "@logsuck/DeleteOldEventsTask"
}

func (t *DeleteOldEventsTask) Run(cfg map[string]any, ctx context.Context) {
	minAgeAny, ok := cfg["minAge"]
	if !ok {
		t.Logger.Error("Failed to get minAge. Will not do anything.")
		return
	}
	minAgeStr, ok := minAgeAny.(string)
	if !ok {
		t.Logger.Error("Failed to cast minAge to string. Will not do anything.")
		return
	}
	if minAgeStr == "" {
		t.Logger.Error("minAgeStr=''. Will not do anything.")
		return
	}
	d, err := parseDuration(minAgeStr)
	if err != nil {
		t.Logger.Error("Failed to parse minAgeStr. Will not do anything.",
			slog.String("minAgeStr", minAgeStr))
		return
	}
	endTime := time.Now().Add(-d)
	eventsChan := t.Repo.FilterStream(&search.Search{}, nil, &endTime)
	for events := range eventsChan {
		t.Logger.Info("Got events to delete",
			slog.Int("numEvents", len(events)))
		ids := make([]int64, len(events))
		for i, evt := range events {
			ids[i] = evt.Id
		}
		err := t.Repo.DeleteBatch(ids)
		if err != nil {
			t.Logger.Error("Failed to delete events",
				slog.Int("numEvents", len(ids)),
				slog.Any("error", err))
		}
	}
}

var durationRegexp = regexp.MustCompile("^(\\d+)(s|m|h|d|M|y)$")

// Unfortunately time.ParseDuration does not support strings like "1d".
// And specifying max age in terms of hours is probably not great if you want a max age of a year or something...
// In this function, 1 day = 24 hours, 1 month = 30 days and 1 year=365 days. So there is no consideration for leap years or anything silly like that.
// And yes, it is a bit wack because 12m != 1y. But hopefully no one will notice.
func parseDuration(str string) (time.Duration, error) {
	match := durationRegexp.FindStringSubmatch(str)
	if len(match) < 3 {
		return 0, fmt.Errorf("str='%s' does not match the duration pattern. A duration must be a positive number followed by one of s, m, h, d, M, or y. For example 7d.", str)
	}
	count, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, fmt.Errorf("str='%s' could not be converted to a duration. Failed to convert '%s' to a number.", str, match[1])
	}
	d := time.Duration(count)
	switch match[2] {
	case "s":
		return d * time.Second, nil
	case "m":
		return d * time.Minute, nil
	case "h":
		return d * time.Hour, nil
	case "d":
		return d * 24 * time.Hour, nil
	case "M":
		return d * 30 * 24 * time.Hour, nil
	case "y":
		return d * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("str='%s' could not be converted to a duration. Unknown duration type='%s'", str, match[2])
	}
}
