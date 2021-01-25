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

import { LogEvent } from "../models/Event";
import { TimeSelection } from "../models/TimeSelection";

interface RestEvent {
  Raw: string;
  Timestamp: string;
  Source: string;
  Fields: { [key: string]: string };
}

export interface StartJobResult {
  id: number;
}

export function startJob(
  searchString: string,
  timeSelection: TimeSelection
): Promise<StartJobResult> {
  let queryParams = `?searchString=${encodeURIComponent(searchString)}`;
  if (timeSelection.relativeTime) {
    queryParams += `&relativeTime=${timeSelection.relativeTime}`;
  }
  if (timeSelection.startTime) {
    queryParams += `&startTime=${timeSelection.startTime.toISOString()}`;
  }
  if (timeSelection.endTime) {
    queryParams += `&endTime=${timeSelection.endTime.toISOString()}`;
  }
  return fetch("/api/v1/startJob" + queryParams, { method: "POST" })
    .then((r) => r.json())
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
  return fetch("/api/v1/jobStats" + queryParams)
    .then((r) => r.json())
    .then((r: RestPollJobResult) => ({
      state: r.State,
      stats: {
        //estimatedProgress: r.Stats.EstimatedProgress,
        numMatchedEvents: r.NumMatchedEvents,
        fieldCount: r.FieldCount,
      },
    }));
}

export function getResults(
  jobId: number,
  skip: number,
  take: number
): Promise<LogEvent[]> {
  const queryParams = `?jobId=${jobId}&skip=${skip}&take=${take}`;
  return fetch("/api/v1/jobResults" + queryParams)
    .then((r) => r.json())
    .then((r: RestEvent[]) =>
      r.map((e) => ({
        raw: e.Raw,
        timestamp: new Date(e.Timestamp),
        source: e.Source,
        fields: e.Fields,
      }))
    );
}

export function abortJob(jobId: number): Promise<{}> {
  const queryParams = `?jobId=${jobId}`;
  return fetch("/api/v1/abortJob" + queryParams, { method: "POST" });
}

export type FieldValueCounts = { [key: string]: number };

export function getFieldValueCounts(
  jobId: number,
  fieldName: string
): Promise<FieldValueCounts> {
  const queryParams = `?jobId=${jobId}&fieldName=${fieldName}`;
  return fetch("/api/v1/jobFieldStats" + queryParams)
    .then((r) => r.json())
    .then((f: FieldValueCounts) => f);
}
