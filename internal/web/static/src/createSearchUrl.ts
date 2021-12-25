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

import { TimeSelection } from "./models/TimeSelection";
import { validateIsoTimestamp } from "./validateIsoTimestamp";

export const createSearchUrl = (
  searchString: string,
  timeSelection: TimeSelection
) => {
  const qp = createSearchQueryParams(searchString, timeSelection);
  const qpString = Object.keys(qp)
    .map((k) => ({ key: k, value: qp[k] }))
    .reduce((prev, { key, value }) => (prev += `&${key}=${value}`), "");
  return `search?${qpString}`;
};

export const createSearchQueryParams = (
  searchString: string,
  timeSelection: TimeSelection
) => {
  const ret: { [key: string]: string } = {};
  ret["query"] = encodeURIComponent(searchString);
  if (timeSelection.relativeTime) {
    ret["relativeTime"] = timeSelection.relativeTime;
    return ret;
  }
  if (validateIsoTimestamp(timeSelection.startTime)) {
    ret["startTime"] = timeSelection.startTime;
  }
  if (validateIsoTimestamp(timeSelection.endTime)) {
    ret["endTime"] = timeSelection.endTime;
  }

  if (
    !timeSelection.relativeTime &&
    !timeSelection.startTime &&
    !timeSelection.endTime
  ) {
    ret["relativeTime"] = "ALL";
  }

  return ret;
};
