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
import { RedirectSearchInput } from "../components/SearchInput";
import { createSearchUrl } from "../createSearchUrl";
import { Card, Table, Title } from "@mantine/core";
import { LogsuckAppShell } from "../components/LogsuckAppShell";
import { TableRow } from "../components/TableRow";

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
      <LogsuckAppShell>
        <RedirectSearchInput navigateTo={this.props.navigateTo} />
        <div>
          <Card>
            <Title order={2}>Recent searches</Title>
            {typeof this.state.recentSearches === "undefined" ? (
              <div className="card-body">
                <p>Loading...</p>
              </div>
            ) : this.state.recentSearches.length === 0 ? (
              <div className="card-body">
                <p>No recent searches</p>
              </div>
            ) : (
              <Table highlightOnHover>
                <tbody>
                  {this.state.recentSearches.map((rs) => (
                    <TableRow
                      key={rs.searchTime.valueOf()}
                      onClick={() => this.onRecentSearchClicked(rs)}
                    >
                      <td>{rs.searchString}</td>
                      <td style={{ textAlign: "right" }}>
                        {rs.timeSelection.relativeTime || "All time"}
                      </td>
                    </TableRow>
                  ))}
                </tbody>
              </Table>
            )}
          </Card>
        </div>
      </LogsuckAppShell>
    );
  }

  private onRecentSearchClicked(rs: RecentSearch) {
    const url = createSearchUrl(rs.searchString, rs.timeSelection);
    this.props.navigateTo(url);
  }
}
