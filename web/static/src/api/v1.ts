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

export interface StartJobResult {
    id: number;
}

export function startJob(searchString: string, timeSelection: TimeSelection): Promise<StartJobResult> {
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
    return fetch('/api/v1/startJob' + queryParams)
        .then(r => r.json())
        .then((r: number) => ({ id: r }));
}

export enum JobState {
    RUNNING = 1,
    FINISHED = 2,
    ABORTED = 3,
}

interface RestPollJobResult {
    State: JobState;
    NumMatchedEvents: number;
    FieldCount: { [key: string]: number };
}

export interface JobStats {
    // estimatedProgress: number;
    numMatchedEvents: number;
    fieldCount: { [key: string]: number };
}

export interface PollJobResult {
    state: JobState;
    stats: JobStats;
}

export function pollJob(jobId: number): Promise<PollJobResult> {
    const queryParams = `?jobId=${jobId}`;
    return fetch('/api/v1/jobStats' + queryParams)
        .then(r => r.json())
        .then((r: RestPollJobResult) => ({
            state: r.State,
            stats: {
                //estimatedProgress: r.Stats.EstimatedProgress,
                numMatchedEvents: r.NumMatchedEvents,
                fieldCount: r.FieldCount
            }
        }));
}

export function getResults(jobId: number, skip: number, take: number): Promise<LogEvent[]> {
    const queryParams = `?jobId=${jobId}&skip=${skip}&take=${take}`;
    return fetch('/api/v1/jobResults' + queryParams)
        .then(r => r.json())
        .then((r: RestEvent[]) => r.map(e => ({
            raw: e.Raw,
            timestamp: new Date(e.Timestamp),
            source: e.Source,
            fields: e.Fields
        })));
}

export function abortJob(jobId: number): Promise<{}> {
    const queryParams = `?jobId=${jobId}`;
    return fetch('/api/v1/abortJob' + queryParams);
}
