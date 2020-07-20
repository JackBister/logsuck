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
