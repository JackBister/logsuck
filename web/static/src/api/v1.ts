import { LogEvent } from "../models/Event"
import { TimeSelection } from "../models/TimeSelection";

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

export function search(searchString: string, timeSelection: TimeSelection): Promise<SearchResult> {
    let queryParams = `?searchString=${searchString}`;
    if (timeSelection.relativeTime) {
        queryParams += `&relativeTime=${timeSelection.relativeTime}`;
    }
    if (timeSelection.startTime) {
        queryParams += `&startTime=${timeSelection.startTime.toISOString()}`;
    }
    if (timeSelection.endTime) {
        queryParams += `&endTime=${timeSelection.endTime.toISOString()}`;
    }
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
