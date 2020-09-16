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

import { TimeSelection } from "../models/TimeSelection";

export interface RecentSearch {
    searchString: string;
    timeSelection: TimeSelection;
    searchTime: Date;
}

export async function addRecentSearch(search: RecentSearch): Promise<void> {
    const before = await getRecentSearches();
    const after = [search, ...before].slice(0, 10);
    window.localStorage.setItem("recentSearches", JSON.stringify(after));
}

export async function getRecentSearches(): Promise<RecentSearch[]> {
    let recentSearchesString = window.localStorage.getItem("recentSearches");
    let recentSearches: RecentSearch[] = recentSearchesString == null ? [] : window.JSON.parse(recentSearchesString);
    if (!(recentSearches instanceof Array)) {
        console.warn("Weirdness when getting recent searches: recentSearches key in localStorage contained a non-array. Will overwrite with an empty array. content:", recentSearches)
        recentSearches = [];
    }
    return recentSearches;
}
