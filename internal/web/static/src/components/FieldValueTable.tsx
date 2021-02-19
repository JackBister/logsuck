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
import { TopFieldValueInfo } from "../models/TopFieldValueInfo";

export interface FieldValueTableProps {
  values: TopFieldValueInfo[];
  onFieldValueClicked: (value: string) => void;
}

export const FieldValueTable = ({
  values,
  onFieldValueClicked,
}: FieldValueTableProps) => (
  <table class="table table-sm table-hover">
    <tbody>
      {values.map((f) => (
        <tr
          key={f.value}
          onClick={() => onFieldValueClicked(f.value)}
          style={{ cursor: "pointer" }}
        >
          <td class="field-value">{f.value}</td>
          <td class="field-value-count" style={{ textAlign: "right" }}>
            {f.count}
          </td>
          <td class="field-value-percentage" style={{ textAlign: "right" }}>
            {(f.percentage * 100).toFixed(2)} %
          </td>
        </tr>
      ))}
    </tbody>
  </table>
);
