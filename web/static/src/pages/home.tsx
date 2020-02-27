import { Component, h } from "preact";
import { SearchResult } from "../api/v1";
import { LogEvent } from "../models/Event";

interface HomeProps {
    searchApi: (searchString: string) => Promise<SearchResult>;
}

export enum HomeState {
    HAVENT_SEARCHED,
    WAITING_FOR_SEARCH,
    SEARCHED_ERROR,
    SEARCHED_SUCCESS
}

interface HomeStateBase {
    state: HomeState;

    searchString: string;
}

export type HomeStateStruct = HaventSearched | WaitingForSearch | SearchedError | SearchedSuccess;

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

interface SearchedSuccess extends HomeStateBase {
    state: HomeState.SEARCHED_SUCCESS;

    searchResult: LogEvent[];
}

export class HomeComponent extends Component<HomeProps, HomeStateStruct> {

    constructor(props: HomeProps) {
        super(props);

        this.state = {
            state: HomeState.HAVENT_SEARCHED,
            searchString: ''
        };
    }

    render() {
        return <div>
            <header>
                <nav class="navbar navbar-dark bg-dark">
                    <a href="/" class="navbar-brand">logsuck</a>
                </nav>
            </header>
            <main role="main" class="container-fluid">
                <div class="search-container">
                    <form onSubmit={(evt) => { this.onSearch(evt); }}>
                        <label for="searchinput">Search</label>
                        <div class="input-group mb-3">
                            <textarea id="searchinput" type="text" class="form-control" onChange={(evt) => this.onSearchChanged(evt)} value={this.state.searchString} />
                            <div class="input-group-append">
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
                    {this.state.state === HomeState.SEARCHED_SUCCESS && <div>
                        {this.state.searchResult.length === 0 && <div class="alert alert-info">
                            No results found. Try a different search?
                            </div>}
                        {this.state.searchResult.length !== 0 && <div class="row">
                            <div class="col-xl-2">
                                <div class="card mb-3 mb-xl-0">
                                    <div class="card-header">
                                        Fields
                                    </div>
                                    <div class="card-body">
                                        Field aggregations will go here!
                                    </div>
                                </div>
                            </div>
                            <div class="col-xl-10">
                                <div class="card">
                                    <table class="table table-hover">
                                        <thead>
                                            <tr>
                                                <th scope="col">Time</th>
                                                <th scope="col">Text</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {this.state.searchResult.map(e => <tr>
                                                <td>{e.timestamp.toLocaleString()}</td>
                                                <td>{e.raw}</td>
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

    private onSearchChanged(evt: any) {
        this.setState({
            searchString: evt.target.value
        });
    }

    private async onSearch(evt: any) {
        evt.preventDefault();
        this.setState({
            state: HomeState.WAITING_FOR_SEARCH
        });
        try {
            const result = await this.props.searchApi(this.state.searchString);
            this.setState({
                state: HomeState.SEARCHED_SUCCESS,
                searchResult: result.events
            });
        } catch (e) {
            console.log(e);
            this.setState({
                state: HomeState.SEARCHED_ERROR,
                searchError: 'Something went wrong.'
            });
        }
    }
}
