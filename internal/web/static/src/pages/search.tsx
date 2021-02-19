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

import { h, render } from "preact";
import { SearchPageComponent } from "./SearchPage";
import {
  startJob,
  pollJob,
  getResults,
  abortJob,
  getFieldValueCounts,
} from "../api/v1";
import { addRecentSearch } from "../services/RecentSearches";

function main() {
  const appRoot = document.getElementById("app");
  if (!appRoot) {
    throw new Error("No element with id 'app' found!");
  }
  render(
    <SearchPageComponent
      startJob={startJob}
      pollJob={pollJob}
      getResults={getResults}
      abortJob={abortJob}
      getFieldValueCounts={getFieldValueCounts}
      addRecentSearch={addRecentSearch}
      getQueryParams={() => new URLSearchParams(window.location.search)}
      setQueryParams={(params) => {
        const url = new URL(window.location.href);
        url.search = params.toString();
        window.history.replaceState(null, document.title, url.toString());
      }}
    />,
    document.body,
    appRoot
  );
}

main();
