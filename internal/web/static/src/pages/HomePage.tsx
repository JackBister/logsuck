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

import { h, Component } from "preact";
import { RecentSearch } from "../services/RecentSearches";
import { Navbar } from "../components/Navbar";
import { RedirectSearchInput } from "../components/SearchInput";
import { createSearchUrl } from "../createSearchUrl";

interface HomeProps {
  getRecentSearches: () => Promise<RecentSearch[]>;
  navigateTo: (url: string) => void;
}

interface HomeState {
  recentSearches?: RecentSearch[];
}

export class HomeComponent extends Component<HomeProps, HomeState> {
  constructor(props: HomeProps) {
    super(props);

    this.state = {};
  }

  async componentDidMount() {
    this.setState({
      recentSearches: await this.props.getRecentSearches(),
    });
  }

  render() {
    return (
      <div>
        <Navbar />
        <main role="main" class="container-fluid">
          <RedirectSearchInput navigateTo={this.props.navigateTo} />
          <div class="result-container">
            <div>
              <div class="card">
                <div class="card-header">Recent searches</div>
                {typeof this.state.recentSearches === "undefined" ? (
                  <div class="card-body">
                    <p>Loading...</p>
                  </div>
                ) : this.state.recentSearches.length === 0 ? (
                  <div class="card-body">
                    <p>No recent searches</p>
                  </div>
                ) : (
                  <table class="table table-sm table-hover">
                    <tbody>
                      {this.state.recentSearches.map((rs) => (
                        <tr
                          key={rs.searchTime.valueOf()}
                          onClick={() => this.onRecentSearchClicked(rs)}
                          style={{ cursor: "pointer" }}
                        >
                          <td>{rs.searchString}</td>
                          <td style={{ textAlign: "right" }}>
                            {rs.timeSelection.relativeTime || "All time"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </div>
          </div>
        </main>
      </div>
    );
  }

  private onRecentSearchClicked(rs: RecentSearch) {
    const url = createSearchUrl(rs.searchString, rs.timeSelection);
    this.props.navigateTo(url);
  }
}
