
export interface LogEvent {
    raw: string;
    timestamp: Date;
    source: string;
    fields: { [key: string]: string }
}
