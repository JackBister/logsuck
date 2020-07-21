import { Component, h } from "preact";
import { StartJobResult, PollJobResult, JobState, FieldValueCounts } from "../api/v1";
import { LogEvent } from "../models/Event";
import { Popover } from "../components/popover";
import { TopFieldValueInfo } from "../models/TopFieldValueInfo";
import { TimeSelect } from "../components/TimeSelect";
import { TimeSelection } from "../models/TimeSelection";
import { Pagination } from "../components/Pagination";
import { RecentSearch } from "../services/RecentSearches";

const EVENTS_PER_PAGE = 25;
const TOP_FIELDS_COUNT = 15;

interface HomeProps {
    startJob: (searchString: string, timeSelection: TimeSelection) => Promise<StartJobResult>
    pollJob: (jobId: number) => Promise<PollJobResult>;
    getResults: (jobId: number, skip: number, take: number) => Promise<LogEvent[]>;
    abortJob: (jobId: number) => Promise<{}>;
    getFieldValueCounts: (jobId: number, fieldName: string) => Promise<FieldValueCounts>;

    addRecentSearch: (search: RecentSearch) => Promise<void>;
    getRecentSearches: () => Promise<RecentSearch[]>;
}

export enum HomeState {
    HAVENT_SEARCHED,
    WAITING_FOR_SEARCH,
    SEARCHED_ERROR,
    SEARCHED_POLLING,
    SEARCHED_POLLING_FINISHED
}

interface HomeStateBase {
    state: HomeState;

    searchString: string;
    selectedTime: TimeSelection;
}

export type HomeStateStruct = HaventSearched | WaitingForSearch | SearchedError | SearchedPolling | SearchedPollingFinished;

interface HaventSearched extends HomeStateBase {
    state: HomeState.HAVENT_SEARCHED;

    recentSearches?: RecentSearch[];
}

interface WaitingForSearch extends HomeStateBase {
    state: HomeState.WAITING_FOR_SEARCH;
}

interface SearchedError extends HomeStateBase {
    state: HomeState.SEARCHED_ERROR;

    searchError: string;
}

interface SearchedPolling extends HomeStateBase {
    state: HomeState.SEARCHED_POLLING;

    jobId: number;
    poller: number;

    searchResult: LogEvent[];
    numMatched: number;

    currentPageIndex: number;

    allFields: { [key: string]: number };
    topFields: { [key: string]: number };
    selectedField: SelectedField | null;
}

interface SearchedPollingFinished extends HomeStateBase {
    state: HomeState.SEARCHED_POLLING_FINISHED;

    jobId: number;

    searchResult: LogEvent[];
    numMatched: number;

    currentPageIndex: number;

    allFields: { [key: string]: number };
    topFields: { [key: string]: number };
    selectedField: SelectedField | null;
}

interface SelectedField {
    name: string;
    topValues: TopFieldValueInfo[];
}

export class HomeComponent extends Component<HomeProps, HomeStateStruct> {

    constructor(props: HomeProps) {
        super(props);

        this.state = {
            state: HomeState.HAVENT_SEARCHED,
            searchString: '',
            selectedTime: {}
        };
    }

    async componentDidMount() {
        if (this.state.state === HomeState.HAVENT_SEARCHED) {
            this.setState({
                recentSearches: await this.props.getRecentSearches(),
            });
        }
    }

    render() {
        return <div onClick={(evt) => this.onBodyClicked(evt)}>
            <header>
                <nav class="navbar navbar-dark bg-dark">
                    <a href="/" class="navbar-brand">logsuck</a>
                </nav>
            </header>
            <main role="main" class="container-fluid">
                <div class="search-container">
                    <form onSubmit={(evt) => { this.onSearch(evt); }}>
                        <label htmlFor="searchinput">Search</label>
                        <div class="input-group mb-3">
                            <input id="searchinput" type="text" class="form-control" onInput={(evt) => this.onSearchChanged(evt)} value={this.state.searchString} />
                            <div class="input-group-append">
                                <TimeSelect onTimeSelected={(ts) => this.setState({ selectedTime: ts })} />
                                <button disabled={this.state.state === HomeState.WAITING_FOR_SEARCH} type="submit" class="btn btn-primary">Search</button>
                            </div>
                        </div>
                    </form>
                </div>
                <div class="result-container">
                    {this.state.state === HomeState.SEARCHED_ERROR &&
                        <div class="alert alert-danger">
                            {this.state.searchError}
                        </div>}
                    {this.state.state === HomeState.HAVENT_SEARCHED &&
                        <div>
                            <div class="card">
                                <div class="card-header">
                                    Recent searches
                                </div>
                                {typeof this.state.recentSearches === 'undefined' ?
                                    <div class="card-body"><p>Loading...</p></div> :
                                    this.state.recentSearches.length === 0 ?
                                        <div class="card-body"><p>No recent searches</p></div> :
                                        <table class="table table-sm table-hover">
                                            <tbody>
                                                {this.state.recentSearches.map((rs) =>
                                                    <tr key={rs.searchTime.valueOf()} onClick={() => this.onRecentSearchClicked(rs)} style={{ cursor: "pointer" }}>
                                                        <td>{rs.searchString}</td>
                                                        <td style={{ textAlign: "right" }}>{
                                                            rs.timeSelection.relativeTime || "All time"
                                                        }</td>
                                                    </tr>)}
                                            </tbody>
                                        </table>}
                            </div>
                        </div>}
                    {this.state.state === HomeState.WAITING_FOR_SEARCH &&
                        <div>
                            Loading... There should be a spinner here!
                        </div>}
                    {(this.state.state === HomeState.SEARCHED_POLLING || this.state.state === HomeState.SEARCHED_POLLING_FINISHED) && <div>
                        {this.state.searchResult.length === 0 && <div class="alert alert-info">
                            No results found. Try a different search?
                            </div>}
                        {this.state.searchResult.length !== 0 && <div class="row">
                            <div class="col-xl-2">
                                <div class="card mb-3 mb-xl-0">
                                    <div class="card-header">
                                        Fields
                                    </div>
                                    {Object.keys(this.state.topFields).length === 0 &&
                                        <div>No fields extracted</div>}
                                    {Object.keys(this.state.topFields).length > 0 &&
                                        <table class="table table-sm table-hover">
                                            <tbody>
                                                {Object.keys(this.state.topFields).map(k =>
                                                    <tr key={k} onClick={(evt) => { evt.stopPropagation(); this.onFieldClicked(k); }} class="test field-row">
                                                        <td>{k}</td>
                                                        {(this.state.state === HomeState.SEARCHED_POLLING || this.state.state === HomeState.SEARCHED_POLLING_FINISHED) &&
                                                            <td style={{ textAlign: "right" }}>{this.state.topFields[k]}</td>
                                                        }
                                                    </tr>)}
                                            </tbody>
                                        </table>}
                                    {
                                        <Popover
                                            direction="right"
                                            isOpen={!!this.state.selectedField}
                                            heading={this.state.selectedField?.name || ''}
                                            widthPx={300}>
                                            <table class="table table-sm table-hover">
                                                <tbody>
                                                    {this.state.selectedField?.topValues.map(f => <tr key={f.value} onClick={() => this.onFieldValueClicked(f.value)} style={{ cursor: "pointer" }}>
                                                        <td class="field-value">{f.value}</td>
                                                        <td class="field-value-count" style={{ textAlign: "right" }}>{f.count}</td>
                                                        <td class="field-value-percentage" style={{ textAlign: "right" }}>{(f.percentage * 100).toFixed(2)} %</td>
                                                    </tr>)}
                                                </tbody>
                                            </table>
                                        </Popover>
                                    }
                                </div>
                            </div>
                            <div class="col-xl-10">
                                <div style={{ display: 'flex', flexDirection: 'row', justifyContent: 'space-between' }}>
                                    <Pagination
                                        currentPageIndex={this.state.currentPageIndex}
                                        numberOfPages={Math.ceil(this.state.numMatched / EVENTS_PER_PAGE)}
                                        onPageChanged={(n) => this.onPageChanged(n)}>
                                    </Pagination>
                                    <div style={{ display: 'flex', flexDirection: 'row', alignItems: 'center' }}>
                                        {
                                            this.state.state === HomeState.SEARCHED_POLLING && <button type="button" class="btn btn-link" onClick={() => this.onCancel()}>
                                                Cancel
                                        </button>
                                        }
                                        <span>
                                            {this.state.numMatched} events matched
                                        </span>
                                    </div>
                                </div>
                                <div class="card">
                                    <table class="table table-hover search-result-table">
                                        <thead>
                                            <tr>
                                                <th scope="col">Time</th>
                                                <th scope="col">Event</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {this.state.searchResult.map(e => <tr key={e.raw}>
                                                <td class="event-timestamp">
                                                    {e.timestamp.toLocaleString()}
                                                </td>
                                                <td>
                                                    <div style={{ display: "flex", flexDirection: "column" }}>
                                                        <div class="event-raw">{e.raw}</div>
                                                        <hr style={{ width: "100%", marginTop: "0.75rem", marginBottom: "0.5rem" }} />
                                                        <div class="event-additional">
                                                            <dl class="row no-gutters" style={{ marginBottom: 0 }}>
                                                                <dt class="col-1">source</dt>
                                                                <dd class="col-1">{e.source}</dd>
                                                            </dl>
                                                        </div>
                                                    </div>
                                                </td>
                                            </tr>)}
                                        </tbody>
                                    </table>
                                </div>
                            </div>
                        </div>}
                    </div>}
                </div>
            </main>
        </div >;
    }

    private onRecentSearchClicked(rs: RecentSearch) {
        this.setState({
            searchString: rs.searchString,
            selectedTime: rs.timeSelection,
        }, () => this.onSearch());
    }

    private onBodyClicked(evt: any) {
        if ((this.state.state === HomeState.SEARCHED_POLLING || this.state.state === HomeState.SEARCHED_POLLING_FINISHED) && this.state.selectedField) {
            if (!(evt.target as HTMLDivElement).matches('.popover *')) {
                this.setState({
                    ...this.state,
                    selectedField: null
                });
            }
        }
    }

    private async onFieldClicked(fieldName: string) {
        if (this.state.state !== HomeState.SEARCHED_POLLING && this.state.state !== HomeState.SEARCHED_POLLING_FINISHED) {
            // Really weird state. Maybe throw error?
            return;
        }
        if (this.state.selectedField?.name === fieldName) {
            this.setState({
                ...this.state,
                selectedField: null
            });
        } else {
            const fieldValues = await this.props.getFieldValueCounts(this.state.jobId, fieldName);
            const keys = Object.keys(fieldValues);
            const totalCount = keys.reduce((acc, k) => acc + fieldValues[k], 0);
            const topValues = keys.sort((a, b) => fieldValues[b] - fieldValues[a])
                .slice(0, TOP_FIELDS_COUNT)
                .map(k => ({
                    value: k,
                    count: fieldValues[k],
                    percentage: fieldValues[k] / totalCount
                }));
            console.log(topValues);
            this.setState({
                ...this.state,
                selectedField: {
                    name: fieldName,
                    topValues: topValues
                }
            });
        }
    }

    private onFieldValueClicked(value: string) {
        if ((this.state.state !== HomeState.SEARCHED_POLLING && this.state.state !== HomeState.SEARCHED_POLLING_FINISHED) || this.state.selectedField === null) {
            return;
        }
        this.addFieldQueryAndSearch(this.state.selectedField.name, value);
    }

    private addFieldQueryAndSearch(key: string, value: string) {
        this.setState({
            searchString: `${key}=${value} ` + this.state.searchString,
            selectedField: null
        }, () => this.onSearch());
    }

    private async onPageChanged(newPageIndex: number) {
        if (this.state.state !== HomeState.SEARCHED_POLLING && this.state.state !== HomeState.SEARCHED_POLLING_FINISHED) {
            throw new Error("Weird state, state=" + this.state.state + ", but attempted to change page");
        }
        try {
            const newEvents = await this.props.getResults(this.state.jobId, newPageIndex * EVENTS_PER_PAGE, EVENTS_PER_PAGE);
            this.setState({
                searchResult: newEvents,
                currentPageIndex: newPageIndex
            });
        } catch (e) {
            console.log(e);
        }
    }

    private async onCancel() {
        if (this.state.state === HomeState.SEARCHED_POLLING_FINISHED) {
            // Polling already finished so there is nothing to cancel, but it's not an error
            return;
        }
        if (this.state.state !== HomeState.SEARCHED_POLLING) {
            throw new Error("Weird state");
        }
        await this.props.abortJob(this.state.jobId);
        window.clearTimeout(this.state.poller);
        this.setState({
            ...this.state,
            state: HomeState.SEARCHED_POLLING_FINISHED
        });
    }

    private onSearchChanged(evt: any) {
        this.setState({
            searchString: evt.target.value
        });
    }

    private async onSearch(evt?: any) {
        if (evt) {
            evt.preventDefault();
        }
        if (this.state.state === HomeState.SEARCHED_POLLING) {
            try {
                window.clearTimeout(this.state.poller);
                await this.props.abortJob(this.state.jobId);
            } catch (e) {
                console.warn(`failed to abort previous jobId=${this.state.jobId}, will continue with new search`)
            }
        }
        this.setState({
            state: HomeState.WAITING_FOR_SEARCH
        });
        try {
            const startJobResult = await this.props.startJob(this.state.searchString, {
                relativeTime: this.state.selectedTime.relativeTime
            });
            this.setState({
                ...this.state,
                state: HomeState.SEARCHED_POLLING,
                jobId: startJobResult.id,
                poller: window.setTimeout(async () => this.poll(startJobResult.id), 500),
                searchResult: [],
                numMatched: 0,
                currentPageIndex: 0
            });
        } catch (e) {
            console.log(e);
            this.setState({
                ...this.state,
                state: HomeState.SEARCHED_ERROR,
                searchError: 'Something went wrong.'
            });
        }
        this.props.addRecentSearch({
            searchString: this.state.searchString,
            timeSelection: this.state.selectedTime,
            searchTime: new Date(),
        });
    }

    private async poll(id: number) {
        if (this.state.state !== HomeState.SEARCHED_POLLING) {
            throw new Error("Really weird state! In poller but state != SEARCHED_POLLING");
        }
        try {
            const pollResult = await this.props.pollJob(id);
            const topFields = Object.keys(pollResult.stats.fieldCount)
                .sort((a, b) => pollResult.stats.fieldCount[b] - pollResult.stats.fieldCount[a])
                .slice(0, TOP_FIELDS_COUNT)
                .reduce((prev, k) => {
                    prev[k] = pollResult.stats.fieldCount[k];
                    return prev;
                }, {} as any);
            const nextState: any = {
                ...this.state,

                numMatched: pollResult.stats.numMatchedEvents,
                allFields: pollResult.stats.fieldCount,
                topFields: topFields
            }
            if (pollResult.state == JobState.ABORTED || pollResult.state == JobState.FINISHED) {
                window.clearTimeout(this.state.poller);
                nextState.state = HomeState.SEARCHED_POLLING_FINISHED;
            } else {
                nextState.poller = window.setTimeout(() => this.poll(id), 500);
            }
            if (this.state.searchResult.length < EVENTS_PER_PAGE && pollResult.stats.numMatchedEvents > this.state.searchResult.length) {
                nextState.searchResult = await this.props.getResults(id, 0, EVENTS_PER_PAGE);
            }
            this.setState(nextState);
        } catch (e) {
            console.log(e);
        }
    }
}
