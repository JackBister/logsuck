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
import { Table, TableRow } from "./lib/Table";

export interface FieldTableProps {
  fields: { [key: string]: number };
  onFieldClicked: (str: string) => void;
}

export const FieldTable = ({ fields, onFieldClicked }: FieldTableProps) => {
  const keys = Object.keys(fields);
  return (
    <div>
      {keys.length === 0 && <div>No fields extracted</div>}
      {keys.length > 0 && (
        <Table hoverable={true}>
          <thead>
            <tr>
              <th>Value</th>
              <th style={{ textAlign: "right" }}>Count</th>
            </tr>
          </thead>
          <tbody>
            {keys.map((k) => (
              <TableRow
                key={k}
                onClick={(evt) => {
                  evt.stopPropagation();
                  onFieldClicked(k);
                }}
              >
                <td>
                  {/* Using abbr so you can mouseover and see the full value if it ends up being truncated - there is surely a better way to achieve this */}
                  <abbr style={{ cursor: "unset" }} title={k}>
                    {k}
                  </abbr>
                </td>
                <td style={{ textAlign: "right" }}>{fields[k]}</td>
              </TableRow>
            ))}
          </tbody>
        </Table>
      )}
    </div>
  );
};
