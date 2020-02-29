import { LogEvent } from "../models/Event"

export interface SearchResult {
    events: LogEvent[];
    fieldCount: { [key: string]: number }
}

interface RestEvent {
    Raw: string;
    Timestamp: string;
    Source: string;
    Fields: { [key: string]: string }
}

interface RestSearchResult {
    Events: RestEvent[];
    FieldCount: { [key: string]: number }
}

export function search(searchString: string): Promise<SearchResult> {
    const queryParams = `?searchString=${searchString}`;
    return fetch('/api/v1/search' + queryParams)
        .then(r => r.json())
        .then((j: RestSearchResult) => {
            const domainEvents = j.Events.map((e) => ({
                raw: e.Raw,
                timestamp: new Date(e.Timestamp),
                source: e.Source,
                fields: e.Fields
            }));
            return {
                events: domainEvents,
                fieldCount: j.FieldCount,
            };
        })
}
