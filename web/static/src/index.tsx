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
import { HomeComponent } from "./pages/home";
import { startJob, pollJob, getResults, abortJob, getFieldValueCounts } from "./api/v1";
import { addRecentSearch, getRecentSearches } from "./services/RecentSearches";

async function main() {
    const appRoot = document.getElementById("app");
    if (!appRoot) {
        throw new Error("No element with id 'app' found!");
    }
    render(<HomeComponent
        startJob={startJob}
        pollJob={pollJob}
        getResults={getResults}
        abortJob={abortJob}
        getFieldValueCounts={getFieldValueCounts}

        addRecentSearch={addRecentSearch}
        getRecentSearches={getRecentSearches}
    />, appRoot);
}

main();
