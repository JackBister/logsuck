import { Component, h } from "preact";
import { SearchResult, StartJobResult, PollJobResult, startJob, JobState } from "../api/v1";
import { LogEvent } from "../models/Event";
import { Popover } from "../components/popover";
import { TopFieldValueInfo } from "../models/TopFieldValueInfo";
import { TimeSelect } from "../components/TimeSelect";
import { TimeSelection } from "../models/TimeSelection";
import { Pagination } from "../components/Pagination";

const EVENTS_PER_PAGE = 25;
const TOP_FIELDS_COUNT = 15;

interface HomeProps {
    searchApi: (searchString: string, timeSelection: TimeSelection) => Promise<SearchResult>;
    startJob: (searchString: string, timeSelection: TimeSelection) => Promise<StartJobResult>
    pollJob: (jobId: number) => Promise<PollJobResult>;
    getResults: (jobId: number, skip: number, take: number) => Promise<LogEvent[]>;
}

export enum HomeState {
    HAVENT_SEARCHED,
    WAITING_FOR_SEARCH,
    SEARCHED_ERROR,
    SEARCHED_POLLING,
    SEARCHED_SUCCESS
}

interface HomeStateBase {
    state: HomeState;

    searchString: string;
    selectedTime: TimeSelection;
}

export type HomeStateStruct = HaventSearched | WaitingForSearch | SearchedError | SearchedPolling | SearchedSuccess;

interface HaventSearched extends HomeStateBase {
    state: HomeState.HAVENT_SEARCHED;
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

    // allFields: { [key: string]: number };
    // topFields: { [key: string]: number };
    // selectedField: SelectedField | null;
}

interface SelectedField {
    name: string;
    topValues: TopFieldValueInfo[];
}

interface SearchedSuccess extends HomeStateBase {
    state: HomeState.SEARCHED_SUCCESS;

    searchResult: LogEvent[];

    allFields: { [key: string]: number };
    topFields: { [key: string]: number };
    selectedField: SelectedField | null;
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
                            <input id="searchinput" type="text" class="form-control" onChange={(evt) => this.onSearchChanged(evt)} value={this.state.searchString} />
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
                    {(this.state.state === HomeState.HAVENT_SEARCHED || this.state.state === HomeState.SEARCHED_ERROR) &&
                        <div>
                            You haven't searched yet! I haven't put content here yet!
                        </div>}
                    {this.state.state === HomeState.WAITING_FOR_SEARCH &&
                        <div>
                            Loading... There should be a spinner here!
                        </div>}
                    {this.state.state === HomeState.SEARCHED_POLLING && <div>
                        {this.state.searchResult.length === 0 && <div class="alert alert-info">
                            No results found. Try a different search?
                            </div>}
                        {this.state.searchResult.length !== 0 && <div class="row">
                            <div class="col-xl-2">
                                <div class="card mb-3 mb-xl-0">
                                    <div class="card-header">
                                        Fields
                                    </div>
                                    {/*
                                    {Object.keys(this.state.topFields).length === 0 &&
                                        <div>No fields extracted</div>}
                                    {Object.keys(this.state.topFields).length > 0 &&
                                        <table class="table table-sm table-hover">
                                            <tbody>
                                                {Object.keys(this.state.topFields).map(k =>
                                                    <tr key={k} onClick={(evt) => { evt.stopPropagation(); this.onFieldClicked(k); }} class="test field-row">
                                                        <td>{k}</td>
                                                        <td style={{ textAlign: "right" }}>{(this.state as SearchedSuccess).topFields[k]}</td>
                                                    </tr>)}
                                            </tbody>
                                        </table>}
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
                                                */}
                                </div>
                            </div>
                            <div class="col-xl-10">
                                <Pagination
                                    currentPageIndex={this.state.currentPageIndex}
                                    numberOfPages={Math.ceil(this.state.numMatched / EVENTS_PER_PAGE)}
                                    onPageChanged={(n) => this.onPageChanged(n)}>
                                </Pagination>
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
        </div>;
    }

    private onBodyClicked(evt: any) {
        if (this.state.state === HomeState.SEARCHED_SUCCESS && this.state.selectedField) {
            if (!(evt.target as HTMLDivElement).matches('.popover *')) {
                this.setState({
                    ...this.state,
                    selectedField: null
                });
            }
        }
    }

    private onFieldClicked(k: string) {
        if (this.state.state !== HomeState.SEARCHED_SUCCESS) {
            // Really weird state. Maybe throw error?
            return;
        }
        if (this.state.selectedField?.name === k) {
            this.setState({
                ...this.state,
                selectedField: null
            });
        } else {
            const topValues = this.calculateTopFieldValues(this.state.searchResult, k);
            this.setState({
                ...this.state,
                selectedField: {
                    name: k,
                    topValues: topValues
                }
            });
        }
    }

    private onFieldValueClicked(value: string) {
        if (this.state.state !== HomeState.SEARCHED_SUCCESS || this.state.selectedField === null) {
            return;
        }
        this.addFieldQueryAndSearch(this.state.selectedField.name, value);
    }

    private addFieldQueryAndSearch(key: string, value: string) {
        this.setState({
            searchString: `${key}=${value} ` + this.state.searchString
        }, () => this.onSearch());
    }

    private calculateTopFieldValues(searchResult: LogEvent[], fieldName: string): TopFieldValueInfo[] {
        const counts: { [key: string]: number } = {};
        let totalCount = 0;
        for (const event of searchResult) {
            if (!event.fields[fieldName]) {
                continue;
            }
            totalCount++;
            const value = event.fields[fieldName];
            if (counts[value]) {
                counts[value]++;
            } else {
                counts[value] = 1;
            }
        }
        return Object.keys(counts)
            .sort((a, b) => counts[b] - counts[a])
            .slice(0, TOP_FIELDS_COUNT)
            .map(k => ({
                value: k,
                count: counts[k],
                percentage: counts[k] / totalCount
            }));
    }

    private async onPageChanged(newPageIndex: number) {
        if (this.state.state !== HomeState.SEARCHED_POLLING) {
            throw new Error("Weird state");
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

    private onSearchChanged(evt: any) {
        this.setState({
            searchString: evt.target.value
        });
    }

    private async onSearch(evt?: any) {
        if (evt) {
            evt.preventDefault();
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
                poller: window.setInterval(async () => {
                    if (this.state.state !== HomeState.SEARCHED_POLLING) {
                        throw new Error("Really weird state! In poller but state != SEARCHED_POLLING");
                    }
                    try {
                        const pollResult = await this.props.pollJob(startJobResult.id);
                        this.setState({
                            numMatched: pollResult.stats.numMatchedEvents
                        });
                        if (this.state.searchResult.length < EVENTS_PER_PAGE) {
                            this.setState({
                                searchResult: await this.props.getResults(startJobResult.id, 0, EVENTS_PER_PAGE)
                            });
                        }
                        if (pollResult.state == JobState.ABORTED || pollResult.state == JobState.FINISHED) {
                            window.clearInterval(this.state.poller);
                            // TODO: Change state
                        }
                    } catch (e) {
                        console.log(e);
                    }
                }, 500),
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
    }

    private static isExcludedFieldName(str: string): boolean {
        return str === '_time' || str === 'source';
    }
}
