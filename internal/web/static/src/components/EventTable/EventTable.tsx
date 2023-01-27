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

import { Button, Flex, Table } from "@mantine/core";
import { h } from "preact";
import { LogEvent } from "../../models/Event";
import {
  eventTableRow,
  eventTimestamp,
  eventRaw,
} from "./EventTable.style.scss";

export interface EventTableProps {
  events: LogEvent[];
  onViewContextClicked: (id: number) => void;
}

export const EventTable = ({
  events,
  onViewContextClicked,
}: EventTableProps) => (
  <Table highlightOnHover>
    <thead>
      <tr>
        <th scope="col" style={{ width: "10%" }}>
          Time
        </th>
        <th scope="col">Event</th>
      </tr>
    </thead>
    <tbody>
      {events.map((e) => (
        <tr key={e.raw} className={eventTableRow}>
          <td className={eventTimestamp}>
            <time dateTime={e.timestamp.toISOString()}>
              {e.timestamp.toLocaleString()}
            </time>
          </td>
          <td>
            <div
              style={{
                display: "flex",
                flexDirection: "column",
              }}
            >
              <div className={eventRaw}>{e.raw}</div>
              <hr
                style={{
                  width: "100%",
                  marginTop: "0",
                  marginBottom: "0",
                }}
              />
              <Flex direction="row" align="center" gap="lg" px="md">
                <dl>
                  <dt>source</dt>
                  <dd>{e.source}</dd>
                </dl>
                <div>
                  <Button
                    type="button"
                    variant="subtle"
                    onClick={() => onViewContextClicked(e.id)}
                    style={{ marginTop: "-2px" }}
                  >
                    View context
                  </Button>
                </div>
              </Flex>
            </div>
          </td>
        </tr>
      ))}
    </tbody>
  </Table>
);
