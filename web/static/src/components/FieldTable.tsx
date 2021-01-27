/**
 * Copyright 2021 The Logsuck Authors
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
        <table class="table table-sm table-hover">
          <tbody>
            {keys.map((k) => (
              <tr
                key={k}
                onClick={(evt) => {
                  evt.stopPropagation();
                  onFieldClicked(k);
                }}
                class="test field-row"
              >
                <td>{k}</td>
                <td style={{ textAlign: "right" }}>{fields[k]}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
};
