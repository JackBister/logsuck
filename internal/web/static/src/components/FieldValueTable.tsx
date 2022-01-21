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

import { h } from "preact";
import { TopFieldValueInfo } from "../models/TopFieldValueInfo";
import { Table, TableRow } from "./lib/Table/Table";

export interface FieldValueTableProps {
  values: TopFieldValueInfo[];
  onFieldValueClicked: (value: string) => void;
}

export const FieldValueTable = ({
  values,
  onFieldValueClicked,
}: FieldValueTableProps) => (
  <Table hoverable={true}>
    <thead>
      <tr>
        <th>Value</th>
        <th style={{ textAlign: "right" }}>Count</th>
        <th style={{ textAlign: "right" }}>%</th>
      </tr>
    </thead>
    <tbody>
      {values.map((f) => (
        <TableRow key={f.value} onClick={() => onFieldValueClicked(f.value)}>
          <td>
            {/* Using abbr so you can mouseover and see the full value if it ends up being truncated - there is surely a better way to achieve this */}
            <abbr style={{ cursor: "unset" }} title={f.value}>
              {f.value}
            </abbr>
          </td>
          <td style={{ textAlign: "right" }}>{f.count}</td>
          <td style={{ textAlign: "right" }}>
            {(f.percentage * 100).toFixed(2)} %
          </td>
        </TableRow>
      ))}
    </tbody>
  </Table>
);
