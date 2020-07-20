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
