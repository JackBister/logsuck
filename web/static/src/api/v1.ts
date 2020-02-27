import { LogEvent } from "../models/Event"

export interface SearchResult {
    events: LogEvent[];
}

type RestSearchResult = {
    Raw: string;
    Timestamp: string;
    Source: string;
}[];

export function search(searchString: string): Promise<SearchResult> {
    const queryParams = `?searchString=${searchString}`;
    return fetch('/api/v1/search' + queryParams)
        .then(r => r.json())
        .then((j: RestSearchResult) => {
            const domainEvents = j.map((e) => ({
                raw: e.Raw,
                timestamp: new Date(e.Timestamp),
                source: e.Source
            }));
            return {
                events: domainEvents
            };
        })
}
