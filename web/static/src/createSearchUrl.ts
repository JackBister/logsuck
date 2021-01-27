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

import { TimeSelection } from "./models/TimeSelection";

export const createSearchUrl = (
  searchString: string,
  timeSelection: TimeSelection
) => {
  const queryEncoded = encodeURIComponent(searchString);
  const tsEncoded = createTSUrl(timeSelection);

  return `/search?query=${queryEncoded}${tsEncoded}`;
};

const createTSUrl = (timeSelection: TimeSelection) => {
  if (timeSelection.relativeTime) {
    return `&relativeTime=${timeSelection.relativeTime}`;
  }
  let url = "";
  if (timeSelection.startTime) {
    url += `&startTime=${timeSelection.startTime.toISOString()}`;
  }
  if (timeSelection.endTime) {
    url += `&endTime=${timeSelection.endTime.toISOString()}`;
  }

  if (
    !timeSelection.relativeTime &&
    !timeSelection.startTime &&
    !timeSelection.endTime
  ) {
    url += "&relativeTime=ALL";
  }

  return url;
};
