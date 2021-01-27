/**
 * Copyright 2020 The Logsuck Authors
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

enum Selection {
  LAST_15_MINUTES,
  LAST_60_MINUTES,
  LAST_4_HOURS,
  LAST_24_HOURS,
  LAST_7_DAYS,
  LAST_30_DAYS,
  ALL_TIME,
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

interface TimeSelectState {}

export class TimeSelect extends Component<TimeSelectProps, TimeSelectState> {
  constructor(props: TimeSelectProps) {
    super(props);
    this.state = {};
  }

  render() {
    const selection = toSelection(this.props.selection);
    let selectedOption = options.find((o) => o.value === selection);
    if (!selectedOption) {
      console.error("Did not find time select option with value=" + selection);
      selectedOption = options[0];
    }
    return [
      <button
        class="btn btn-outline-secondary dropdown-toggle"
        type="button"
        data-toggle="dropdown"
        aria-haspopup="true"
        aria-expanded="false"
      >
        {selectedOption.name}
      </button>,
      <div class="dropdown-menu dropdown-menu-right" style="min-width: 276px;">
        {options.map((o) => (
          <button
            type="button"
            class={
              "dropdown-item" +
              (selection === o.value ? " bg-info text-white" : "")
            }
            onClick={() => this.onSelection(o)}
          >
            {o.name}
          </button>
        ))}
      </div>,
    ];
  }

  private onSelection(o: Option) {
    this.setState({
      currentSelection: o.value,
      selectedOptionName: o.name,
    });
    this.props.onTimeSelected(o.ts);
  }
}
