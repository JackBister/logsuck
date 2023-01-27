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

import { h, hydrate } from "preact";
import {
  abortJob,
  getFieldValueCounts,
  getResults,
  pollJob,
  startJob,
} from "../api/v1";
import { getQueryParams, setQueryParams } from "../queryParams";
import { addRecentSearch } from "../services/RecentSearches";
import { SearchPageComponent } from "./SearchPage";

function main() {
  hydrate(
    <SearchPageComponent
      startJob={startJob}
      pollJob={pollJob}
      getResults={getResults}
      abortJob={abortJob}
      getFieldValueCounts={getFieldValueCounts}
      addRecentSearch={addRecentSearch}
      getQueryParams={getQueryParams}
      setQueryParams={setQueryParams}
    />,
    document.body
  );
}

main();
