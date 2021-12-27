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

import { h, Component } from "preact";
import { TimeSelection } from "../models/TimeSelection";
import { validateIsoTimestamp } from "../validateIsoTimestamp";
import { Button } from "./lib/Button";
import { Dropdown, DropdownItem } from "./lib/Dropdown";
import { Input, InputGroup } from "./lib/Input";

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
      <Dropdown
        isOpen={this.state.isOpen}
        onOpenStateChanged={(isOpen) => this.setState({ isOpen })}
        triggerText={displayName}
      >
        <div>
          {options.map((o) => (
            <DropdownItem
              type="button"
              isCurrent={selection === o.value}
              onClick={() => this.onSelection(o)}
            >
              {o.name}
            </DropdownItem>
          ))}
          <hr />
          <div className="px-4">
            <h6 className="t-h6">Date and time range</h6>
            <div className="mb-3">
              <div className="d-flex justify-between">
                <label>From</label>
                <Button
                  type="button"
                  buttonType="text"
                  onClick={(evt) => {
                    this.props.onTimeSelected({
                      ...this.props.selection,
                      startTime: undefined,
                    });
                    evt.stopPropagation();
                  }}
                >
                  Clear
                </Button>
              </div>
              <InputGroup>
                <Input
                  id="timeSelectAbsoluteFromDate"
                  name="timeSelectAbsoluteFromDate"
                  type="date"
                  placeholder="yyyy-MM-dd"
                  onInput={(evt) => {
                    evt.preventDefault();
                    this.onDateUpdated("startTime", (evt.target as any).value);
                  }}
                  value={startDate}
                />
                <Input
                  id="timeSelectAbsoluteFromTime"
                  name="timeSelectAbsoluteFromTime"
                  type="time"
                  step="1"
                  placeholder="HH:mm:ss"
                  onInput={(evt) => {
                    evt.preventDefault();
                    this.onTimeUpdated("startTime", (evt.target as any).value);
                  }}
                  value={startTime}
                />
              </InputGroup>
            </div>
            <div className="mb-3">
              <div className="d-flex justify-between">
                <label>To</label>
                <Button
                  type="button"
                  buttonType="text"
                  onClick={(evt) => {
                    this.props.onTimeSelected({
                      ...this.props.selection,
                      endTime: undefined,
                    });
                    evt.stopPropagation();
                  }}
                >
                  Clear
                </Button>
              </div>
              <InputGroup>
                <Input
                  id="timeSelectAbsoluteToDate"
                  name="timeSelectAbsoluteToDate"
                  type="date"
                  placeholder="yyyy-MM-dd"
                  onInput={(evt) => {
                    evt.preventDefault();
                    this.onDateUpdated("endTime", (evt.target as any).value);
                  }}
                  value={endDate}
                />
                <Input
                  id="timeSelectAbsoluteToTime"
                  name="timeSelectAbsoluteToTime"
                  type="time"
                  step="1"
                  placeholder="HH:mm:ss"
                  onInput={(evt) => {
                    evt.preventDefault();
                    this.onTimeUpdated("endTime", (evt.target as any).value);
                  }}
                  value={endTime}
                />
              </InputGroup>
            </div>
          </div>
        </div>
      </Dropdown>
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
    const nextSelection = { ...this.props.selection };
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
    const nextSelection = { ...this.props.selection };
    nextSelection[part] = next;
    this.props.onTimeSelected(nextSelection);
  }
}
