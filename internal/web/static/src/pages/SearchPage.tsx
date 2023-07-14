/**
 * Copyright 2021 Jack Bister
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

import { Component, h } from "preact";
import {
  FieldValueCounts,
  JobResultResponse,
  JobState,
  PollJobResult,
  StartJobResult,
} from "../api/v1";
import { EventTable } from "../components/EventTable/EventTable";
import { FieldTable } from "../components/FieldTable";
import { FieldValueTable } from "../components/FieldValueTable";
import { SearchInput } from "../components/SearchInput";
import { createSearchQueryParams } from "../createSearchUrl";
import { LogEvent } from "../models/Event";
import { TimeSelection } from "../models/TimeSelection";
import { TopFieldValueInfo } from "../models/TopFieldValueInfo";
import { RecentSearch } from "../services/RecentSearches";
import { validateIsoTimestamp } from "../validateIsoTimestamp";
import { LogsuckAppShell } from "../components/LogsuckAppShell";
import { Alert, Button, Card, Pagination, Popover, Table } from "@mantine/core";
import { JsonView } from "../components/JsonView/JsonView";

const EVENTS_PER_PAGE = 25;
const TOP_FIELDS_COUNT = 15;

interface SearchProps {
  startJob: (
    searchString: string,
    timeSelection: TimeSelection
  ) => Promise<StartJobResult>;
  pollJob: (jobId: number) => Promise<PollJobResult>;
  getResults: (
    jobId: number,
    skip: number,
    take: number
  ) => Promise<JobResultResponse>;
  abortJob: (jobId: number) => Promise<{}>;
  getFieldValueCounts: (
    jobId: number,
    fieldName: string
  ) => Promise<FieldValueCounts>;

  addRecentSearch: (search: RecentSearch) => Promise<void>;

  getQueryParams: () => URLSearchParams;
  setQueryParams: (params: URLSearchParams) => void;
}

export enum SearchState {
  HAVENT_SEARCHED,
  WAITING_FOR_SEARCH,
  SEARCHED_ERROR,
  SEARCHED_POLLING,
  SEARCHED_POLLING_FINISHED,
}

interface SearchStateBase {
  state: SearchState;

  searchString: string;
  selectedTime: TimeSelection;
}

export type SearchStateStruct =
  | HaventSearched
  | WaitingForSearch
  | SearchedError
  | SearchedPolling
  | SearchedPollingFinished;

interface HaventSearched extends SearchStateBase {
  state: SearchState.HAVENT_SEARCHED;
}

interface WaitingForSearch extends SearchStateBase {
  state: SearchState.WAITING_FOR_SEARCH;
}

interface SearchedError extends SearchStateBase {
  state: SearchState.SEARCHED_ERROR;

  searchError: string;
}

interface SearchedPolling extends SearchStateBase {
  state: SearchState.SEARCHED_POLLING;

  jobId: number;
  poller: number;

  searchResult: JobResultResponse;
  numMatched: number;

  currentPageIndex: number;

  allFields: { [key: string]: number };
  topFields: { [key: string]: number };
  selectedField: SelectedField | null;
  showSelectedField: boolean;
}

interface SearchedPollingFinished extends SearchStateBase {
  state: SearchState.SEARCHED_POLLING_FINISHED;

  jobId: number;

  searchResult: JobResultResponse;
  numMatched: number;

  currentPageIndex: number;

  allFields: { [key: string]: number };
  topFields: { [key: string]: number };
  selectedField: SelectedField | null;
  showSelectedField: boolean;
}

interface SelectedField {
  name: string;
  topValues: TopFieldValueInfo[];
}

export class SearchPageComponent extends Component<
  SearchProps,
  SearchStateStruct
> {
  constructor(props: SearchProps) {
    super(props);

    this.state = {
      state: SearchState.HAVENT_SEARCHED,
      searchString: "",
      selectedTime: {
        relativeTime: "-15m",
      },
    };
  }

  componentDidMount() {
    const queryParams = this.props.getQueryParams();
    let doSearch = false;
    let newState: Partial<SearchStateStruct> = {};
    if (queryParams.has("query")) {
      newState.searchString = decodeURIComponent(
        queryParams.get("query") || ""
      );
      doSearch = true;
    }
    if (queryParams.has("relativeTime")) {
      const relativeTime = queryParams.get("relativeTime");
      if (relativeTime === "ALL") {
        newState.selectedTime = {};
      } else {
        newState.selectedTime = {
          relativeTime: relativeTime || undefined,
        };
      }
      doSearch = true;
    }
    const hasStartTime = queryParams.has("startTime");
    const hasEndTime = queryParams.has("endTime");
    if (hasStartTime || hasEndTime) {
      newState.selectedTime = {};
      doSearch = true;
    }
    if (hasStartTime) {
      const startTimeStr = queryParams.get("startTime") as string;
      if (validateIsoTimestamp(startTimeStr)) {
        newState.selectedTime!.startTime = startTimeStr;
      }
    }
    if (hasEndTime) {
      const endTimeStr = queryParams.get("endTime") as string;
      if (validateIsoTimestamp(endTimeStr)) {
        newState.selectedTime!.endTime = endTimeStr;
      }
    }
    if (queryParams.has("jobId")) {
      const jobIdString = queryParams.get("jobId") as string;
      const jobId = parseInt(jobIdString, 10);
      let currentPageIndex = 0;
      if (queryParams.has("page")) {
        const pageIndex = parseInt(queryParams.get("page") as string, 10);
        if (!isNaN(pageIndex)) {
          currentPageIndex = pageIndex;
        }
      }
      if (!isNaN(jobId)) {
        newState = {
          ...newState,
          state: SearchState.SEARCHED_POLLING,
          jobId: jobId,
          poller: window.setTimeout(async () => this.poll(jobId), 0),
          searchResult: {
            resultType: "EVENTS",
            events: [],
          },
          numMatched: 0,
          currentPageIndex: currentPageIndex,
        };
        doSearch = false;
      }
    }
    this.setState(newState, () => {
      if (doSearch) {
        this.onSearch();
      }
    });
  }

  render() {
    const resultLength = this.getResultLength();
    return (
      <LogsuckAppShell>
        <SearchInput
          isButtonDisabled={this.state.state === SearchState.WAITING_FOR_SEARCH}
          searchString={this.state.searchString}
          setSearchString={(str) => this.setState({ searchString: str })}
          selectedTime={this.state.selectedTime}
          setSelectedTime={(ts) => this.setState({ selectedTime: ts })}
          onSearch={() => this.onSearch()}
        />
        <div>
          {this.state.state === SearchState.SEARCHED_ERROR && (
            <Alert title="Error" color="red">
              {this.state.searchError}
            </Alert>
          )}
          {(this.state.state === SearchState.WAITING_FOR_SEARCH ||
            (this.state.state === SearchState.SEARCHED_POLLING &&
              resultLength === 0)) && (
            <div>Loading... There should be a spinner here!</div>
          )}
          {((this.state.state === SearchState.SEARCHED_POLLING &&
            resultLength > 0) ||
            this.state.state === SearchState.SEARCHED_POLLING_FINISHED) && (
            <div>
              {resultLength === 0 && (
                <Alert>No results found. Try a different search?</Alert>
              )}
              {resultLength !== 0 && (
                <div className="w-100 d-flex flex-row align-start gap-6">
                  <Popover
                    position="right"
                    opened={this.state.showSelectedField}
                    onChange={(isOpen) => {
                      if (!isOpen) {
                        this.setState({ showSelectedField: false });
                      }
                    }}
                  >
                    <Popover.Target>
                      <Card className="shrink-1">
                        <FieldTable
                          fields={this.state.topFields}
                          onFieldClicked={(str) => this.onFieldClicked(str)}
                        />
                      </Card>
                    </Popover.Target>
                    <Popover.Dropdown>
                      <FieldValueTable
                        values={this.state.selectedField?.topValues || []}
                        onFieldValueClicked={(str) =>
                          this.onFieldValueClicked(str)
                        }
                      />
                    </Popover.Dropdown>
                  </Popover>
                  <div className="grow-1 shrink-0" style={{ flexBasis: "80%" }}>
                    <div className="d-flex flex-row justify-between">
                      <Pagination
                        value={this.state.currentPageIndex + 1}
                        total={Math.ceil(
                          this.state.numMatched / EVENTS_PER_PAGE
                        )}
                        onChange={(n: number) => this.onPageChanged(n)}
                      ></Pagination>
                      <div className="mb-3 d-flex flex-row align-center">
                        {this.state.state === SearchState.SEARCHED_POLLING && (
                          <Button
                            type="button"
                            variant="subtle"
                            onClick={() => this.onCancel()}
                          >
                            Cancel
                          </Button>
                        )}
                        <span>{this.state.numMatched} events matched</span>
                      </div>
                    </div>
                    <Card>
                      {this.state.searchResult.resultType === "EVENTS" && (
                        <EventTable
                          events={this.state.searchResult.events}
                          onViewContextClicked={(id) =>
                            this.onViewContextClicked(id)
                          }
                        />
                      )}
                      {this.state.searchResult.resultType === "TABLE" && (
                        <div>
                          <Table>
                            <thead>
                              <tr>
                                {this.getColumnOrder().map((k) => (
                                  <th style={{ paddingLeft: "28px" }}>{k}</th>
                                ))}
                              </tr>
                            </thead>
                            <tbody>
                              {this.state.searchResult.tableRows.map((tr) => {
                                return (
                                  <tr key={tr.rowNumber}>
                                    {this.getColumnOrder().map((k) => (
                                      <td key={k}>
                                        <Button
                                          variant="subtle"
                                          onClick={() =>
                                            this.addFieldQueryAndSearch(
                                              k,
                                              tr.values[k]
                                            )
                                          }
                                        >
                                          {tr.values[k]}
                                        </Button>
                                      </td>
                                    ))}
                                  </tr>
                                );
                              })}
                            </tbody>
                          </Table>
                        </div>
                      )}
                    </Card>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </LogsuckAppShell>
    );
  }

  private async onFieldClicked(fieldName: string) {
    if (
      this.state.state !== SearchState.SEARCHED_POLLING &&
      this.state.state !== SearchState.SEARCHED_POLLING_FINISHED
    ) {
      // Really weird state. Maybe throw error?
      return;
    }
    if (this.state.selectedField?.name === fieldName) {
      this.setState({
        ...this.state,
        showSelectedField: !this.state.showSelectedField,
      });
    } else {
      const fieldValues = await this.props.getFieldValueCounts(
        this.state.jobId,
        fieldName
      );
      const keys = Object.keys(fieldValues);
      const totalCount = keys.reduce((acc, k) => acc + fieldValues[k], 0);
      const topValues = keys
        .sort((a, b) => fieldValues[b] - fieldValues[a])
        .slice(0, TOP_FIELDS_COUNT)
        .map((k) => ({
          value: k,
          count: fieldValues[k],
          percentage: fieldValues[k] / totalCount,
        }));
      this.setState({
        ...this.state,
        selectedField: {
          name: fieldName,
          topValues: topValues,
        },
        showSelectedField: true,
      });
    }
  }

  private onFieldValueClicked(value: string) {
    if (
      (this.state.state !== SearchState.SEARCHED_POLLING &&
        this.state.state !== SearchState.SEARCHED_POLLING_FINISHED) ||
      this.state.selectedField === null
    ) {
      return;
    }
    this.addFieldQueryAndSearch(this.state.selectedField.name, value);
  }

  private addFieldQueryAndSearch(key: string, value: string) {
    this.setState(
      {
        searchString: `${key}=${value} ` + this.state.searchString,
        selectedField: null,
        showSelectedField: false,
      },
      () => this.onSearch()
    );
  }

  private onViewContextClicked(id: number) {
    this.setState(
      {
        searchString: `| surrounding eventId=${id}`,
      },
      () => this.onSearch()
    );
  }

  private async onPageChanged(newPageNumber: number) {
    const newPageIndex = newPageNumber - 1;
    if (
      this.state.state !== SearchState.SEARCHED_POLLING &&
      this.state.state !== SearchState.SEARCHED_POLLING_FINISHED
    ) {
      throw new Error(
        "Weird state, state=" +
          this.state.state +
          ", but attempted to change page"
      );
    }
    try {
      const result = await this.props.getResults(
        this.state.jobId,
        newPageIndex * EVENTS_PER_PAGE,
        EVENTS_PER_PAGE
      );
      this.setState({
        searchResult: result,
        currentPageIndex: newPageIndex,
      });
      this.setQueryParams({
        page: newPageIndex.toString(),
      });
    } catch (e) {
      console.log(e);
    }
  }

  private async onCancel() {
    if (this.state.state === SearchState.SEARCHED_POLLING_FINISHED) {
      // Polling already finished so there is nothing to cancel, but it's not an error
      return;
    }
    if (this.state.state !== SearchState.SEARCHED_POLLING) {
      throw new Error("Weird state");
    }
    await this.props.abortJob(this.state.jobId);
    window.clearTimeout(this.state.poller);
    this.setState({
      ...this.state,
      state: SearchState.SEARCHED_POLLING_FINISHED,
    });
  }

  private async onSearch() {
    if (this.state.state === SearchState.SEARCHED_POLLING) {
      try {
        window.clearTimeout(this.state.poller);
        await this.props.abortJob(this.state.jobId);
      } catch (e) {
        console.warn(
          `failed to abort previous jobId=${this.state.jobId}, will continue with new search`
        );
      }
    }
    this.setState({
      state: SearchState.WAITING_FOR_SEARCH,
    });
    try {
      const qp = createSearchQueryParams(
        this.state.searchString,
        this.state.selectedTime
      );
      this.clearQueryParams();
      this.setQueryParams(qp);
    } catch (e) {
      console.warn("failed to set new query params when starting search", e);
    }
    try {
      const startJobResult = await this.props.startJob(
        this.state.searchString,
        this.state.selectedTime
      );
      this.setState({
        ...this.state,
        state: SearchState.SEARCHED_POLLING,
        jobId: startJobResult.id,
        poller: window.setTimeout(
          async () => this.poll(startJobResult.id),
          500
        ),
        searchResult: {
          resultType: "EVENTS",
          events: [],
        },
        numMatched: 0,
        currentPageIndex: 0,
      });
      this.setQueryParams({ jobId: startJobResult.id.toString() });
    } catch (e) {
      console.log(e);
      this.setState({
        ...this.state,
        state: SearchState.SEARCHED_ERROR,
        searchError: "Something went wrong.",
      });
    }
    this.props.addRecentSearch({
      searchString: this.state.searchString,
      timeSelection: this.state.selectedTime,
      searchTime: new Date(),
    });
  }

  private clearQueryParams() {
    this.props.setQueryParams(new URLSearchParams());
  }

  private setQueryParams(qp: { [key: string]: string }) {
    const queryParams = this.props.getQueryParams();
    for (const k of Object.keys(qp)) {
      queryParams.set(k, qp[k]);
    }
    this.props.setQueryParams(queryParams);
  }

  private async poll(id: number) {
    if (this.state.state !== SearchState.SEARCHED_POLLING) {
      throw new Error(
        "Really weird state! In poller but state != SEARCHED_POLLING"
      );
    }
    if (id !== this.state.jobId) {
      return;
    }
    try {
      const pollResult = await this.props.pollJob(id);
      if (id !== this.state.jobId) {
        return;
      }
      const topFields = Object.keys(pollResult.stats.fieldCount)
        .sort(
          (a, b) =>
            pollResult.stats.fieldCount[b] - pollResult.stats.fieldCount[a]
        )
        .slice(0, TOP_FIELDS_COUNT)
        .reduce((prev, k) => {
          prev[k] = pollResult.stats.fieldCount[k];
          return prev;
        }, {} as any);
      const nextState: any = {
        ...this.state,

        numMatched: pollResult.stats.numMatchedEvents,
        allFields: pollResult.stats.fieldCount,
        topFields: topFields,
      };
      if (
        pollResult.state == JobState.ABORTED ||
        pollResult.state == JobState.FINISHED
      ) {
        window.clearTimeout(this.state.poller);
        nextState.state = SearchState.SEARCHED_POLLING_FINISHED;
      } else {
        nextState.poller = window.setTimeout(() => this.poll(id), 500);
      }
      const resultLength = this.getResultLength();
      if (
        resultLength < EVENTS_PER_PAGE &&
        pollResult.stats.numMatchedEvents > resultLength
      ) {
        const result = await this.props.getResults(
          id,
          this.state.currentPageIndex * EVENTS_PER_PAGE,
          EVENTS_PER_PAGE
        );
        nextState.searchResult = result;
        if (id !== this.state.jobId) {
          return;
        }
      }
      this.setState(nextState);
    } catch (e) {
      console.log(e);
    }
  }

  private getColumnOrder(): string[] {
    if (
      this.state.state !== SearchState.SEARCHED_POLLING &&
      this.state.state !== SearchState.SEARCHED_POLLING_FINISHED
    ) {
      throw new Error(
        "Unexpected state, inside TABLE tbody but state is not SEARCHED_POLLING or SEARCHED_POLLING_FINISHED."
      );
    }
    if (this.state.searchResult.resultType !== "TABLE") {
      throw new Error(
        "Unexpected state, inside TABLE tbody but resultType is not TABLE."
      );
    }
    if (
      !this.state.searchResult.columnOrder ||
      this.state.searchResult.columnOrder.length === 0
    ) {
      if (this.state.searchResult.tableRows.length > 0) {
        return Object.keys(this.state.searchResult.tableRows[0].values);
      } else {
        return [];
      }
    }
    return this.state.searchResult.columnOrder;
  }

  private getResultLength(): number {
    let resultLength = 0;
    if (
      this.state.state === SearchState.SEARCHED_POLLING ||
      this.state.state === SearchState.SEARCHED_POLLING_FINISHED
    ) {
      if (this.state.searchResult.resultType === "EVENTS") {
        resultLength = this.state.searchResult.events.length;
      } else {
        resultLength = this.state.searchResult.tableRows.length;
      }
    }
    return resultLength;
  }
}
