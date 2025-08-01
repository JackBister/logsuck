/**
 * Copyright 2021 Jack Bister
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { Button, Flex, Menu, Space, Text, TextInput } from "@mantine/core";
import { h, Component } from "preact";
import { TimeSelection } from "../models/TimeSelection";
import { validateIsoTimestamp } from "../validateIsoTimestamp";

enum Selection {
  LAST_15_MINUTES,
  LAST_60_MINUTES,
  LAST_4_HOURS,
  LAST_24_HOURS,
  LAST_7_DAYS,
  LAST_30_DAYS,
  ALL_TIME,
  ABSOLUTE,
}

const toSelection = (ts: TimeSelection) => {
  if (ts.relativeTime === "-15m") {
    return Selection.LAST_15_MINUTES;
  } else if (ts.relativeTime === "-60m") {
    return Selection.LAST_60_MINUTES;
  } else if (ts.relativeTime === "-4h") {
    return Selection.LAST_4_HOURS;
  } else if (ts.relativeTime === "-24h") {
    return Selection.LAST_24_HOURS;
  } else if (ts.relativeTime === "-168h") {
    return Selection.LAST_7_DAYS;
  } else if (ts.relativeTime === "-720h") {
    return Selection.LAST_30_DAYS;
  } else if (
    validateIsoTimestamp(ts.startTime) ||
    validateIsoTimestamp(ts.endTime)
  ) {
    return Selection.ABSOLUTE;
  } else {
    return Selection.ALL_TIME;
  }
};

interface Option {
  value: Selection;
  name: string;
  ts: TimeSelection;
}

const options: Option[] = [
  {
    value: Selection.LAST_15_MINUTES,
    name: "Last 15 minutes",
    ts: { relativeTime: "-15m" },
  },
  {
    value: Selection.LAST_60_MINUTES,
    name: "Last 60 minutes",
    ts: { relativeTime: "-60m" },
  },
  {
    value: Selection.LAST_4_HOURS,
    name: "Last 4 hours",
    ts: { relativeTime: "-4h" },
  },
  {
    value: Selection.LAST_24_HOURS,
    name: "Last 24 hours",
    ts: { relativeTime: "-24h" },
  },
  {
    value: Selection.LAST_7_DAYS,
    name: "Last 7 days",
    ts: { relativeTime: "-168h" },
  },
  {
    value: Selection.LAST_30_DAYS,
    name: "Last 30 days",
    ts: { relativeTime: "-720h" },
  },
  { value: Selection.ALL_TIME, name: "All time", ts: {} },
];

interface TimeSelectProps {
  selection: TimeSelection;
  onTimeSelected: (newTime: TimeSelection) => void;
}

interface TimeSelectState {
  isOpen: boolean;
}

export class TimeSelect extends Component<TimeSelectProps, TimeSelectState> {
  constructor(props: TimeSelectProps) {
    super(props);
    this.state = {
      isOpen: false,
    };
  }

  render() {
    const selection = toSelection(this.props.selection);
    let displayName: string;
    if (selection === Selection.ABSOLUTE) {
      if (
        validateIsoTimestamp(this.props.selection.startTime) &&
        validateIsoTimestamp(this.props.selection.endTime)
      ) {
        displayName =
          this.props.selection.startTime +
          " to " +
          this.props.selection.endTime;
      } else if (validateIsoTimestamp(this.props.selection.startTime)) {
        displayName = "After " + this.props.selection.startTime;
      } else {
        displayName = "Before " + this.props.selection.endTime;
      }
    } else {
      let selectedOption = options.find((o) => o.value === selection);
      if (!selectedOption) {
        console.error(
          "Did not find time select option with value=" + selection
        );
        displayName = options[0].name;
      } else {
        displayName = selectedOption.name;
      }
    }
    let startDate: string = "";
    let startTime: string = "";
    let endDate: string = "";
    let endTime: string = "";
    if (this.props.selection.startTime) {
      const split = this.props.selection.startTime.split("T");
      if (split.length === 2) {
        startDate = split[0];
        startTime = split[1];
      }
    }
    if (this.props.selection.endTime) {
      const split = this.props.selection.endTime.split("T");
      if (split.length === 2) {
        endDate = split[0];
        endTime = split[1];
      }
    }
    return (
      <Menu width={300}>
        <Menu.Target>
          <Button variant="outline">{displayName}</Button>
        </Menu.Target>
        <Menu.Dropdown>
          {options.map((o) => (
            <Menu.Item
              type="button"
              color={o.value === selection ? "blue" : undefined}
              onClick={() => this.onSelection(o)}
            >
              {o.name}
            </Menu.Item>
          ))}
          <Menu.Divider />
          <Menu.Label>Date and time range</Menu.Label>
          <div style="padding: 0 12px 10px">
            <Flex direction="row" justify="space-between">
              <Text fz="sm">From</Text>
              <Button
                type="button"
                variant="subtle"
                onClick={(evt: any) => {
                  this.props.onTimeSelected({
                    ...this.props.selection,
                    startTime: undefined,
                  });
                  evt.stopPropagation();
                }}
              >
                Clear
              </Button>
            </Flex>
            <Flex direction="row" gap="sm">
              <TextInput
                id="timeSelectAbsoluteFromDate"
                name="timeSelectAbsoluteFromDate"
                aria-label="From date"
                type="date"
                placeholder="yyyy-MM-dd"
                onChange={(evt: any) => {
                  evt.preventDefault();
                  this.onDateUpdated(
                    "startTime",
                    (evt.currentTarget as any).value
                  );
                }}
                onKeyDown={(evt: any) => {
                  evt.stopPropagation();
                }}
                value={startDate}
              />
              <TextInput
                id="timeSelectAbsoluteFromTime"
                name="timeSelectAbsoluteFromTime"
                aria-label="From time"
                type="time"
                step="1"
                placeholder="HH:mm:ss"
                style={{ flexGrow: 1 }}
                onChange={(evt: any) => {
                  evt.preventDefault();
                  this.onTimeUpdated(
                    "startTime",
                    (evt.currentTarget as any).value
                  );
                }}
                onKeyDown={(evt: any) => {
                  evt.stopPropagation();
                }}
                value={startTime}
              />
            </Flex>
            <Space h="sm"></Space>
            <Flex direction="row" justify="space-between">
              <Text fz="sm">To</Text>
              <Button
                type="button"
                variant="subtle"
                onClick={(evt: any) => {
                  this.props.onTimeSelected({
                    ...this.props.selection,
                    endTime: undefined,
                  });
                  evt.stopPropagation();
                }}
              >
                Clear
              </Button>
            </Flex>
            <Flex direction="row" gap="sm">
              <TextInput
                id="timeSelectAbsoluteToDate"
                name="timeSelectAbsoluteToDate"
                aria-label="To date"
                type="date"
                placeholder="yyyy-MM-dd"
                onChange={(evt: any) => {
                  evt.preventDefault();
                  this.onDateUpdated(
                    "endTime",
                    (evt.currentTarget as any).value
                  );
                }}
                onKeyDown={(evt: any) => {
                  evt.stopPropagation();
                }}
                value={endDate}
              />
              <TextInput
                id="timeSelectAbsoluteToTime"
                name="timeSelectAbsoluteToTime"
                aria-label="To time"
                type="time"
                step="1"
                placeholder="HH:mm:ss"
                style={{ flexGrow: 1 }}
                onChange={(evt: any) => {
                  evt.preventDefault();
                  this.onTimeUpdated(
                    "endTime",
                    (evt.currentTarget as any).value
                  );
                }}
                onKeyDown={(evt: any) => {
                  evt.stopPropagation();
                }}
                value={endTime}
              />
            </Flex>
          </div>
        </Menu.Dropdown>
      </Menu>
    );
  }

  private onSelection(o: Option) {
    this.props.onTimeSelected(o.ts);
    this.setState({ isOpen: false });
  }

  private onDateUpdated(part: "startTime" | "endTime", value: string) {
    let previous = this.props.selection[part];
    let next: string;
    if (!previous) {
      next = value + "T00:00:00";
    } else {
      const split = previous.split("T");
      next = value + "T" + split[1];
    }
    const nextSelection: TimeSelection = {
      ...this.props.selection,
      relativeTime: undefined,
    };
    nextSelection[part] = next;
    this.props.onTimeSelected(nextSelection);
  }

  private onTimeUpdated(part: "startTime" | "endTime", value: string) {
    let previous = this.props.selection[part];
    let next: string;
    if (!previous) {
      next = "T" + value;
    } else {
      const split = previous.split("T");
      next = split[0] + "T" + value;
    }
    const nextSelection: TimeSelection = {
      ...this.props.selection,
      relativeTime: undefined,
    };
    nextSelection[part] = next;
    this.props.onTimeSelected(nextSelection);
  }
}
