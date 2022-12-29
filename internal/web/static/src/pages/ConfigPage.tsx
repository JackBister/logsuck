/**
 * Copyright 2022 Jack Bister
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
import { Config } from "../api/v1";
import { Autoform, FormSpec } from "../components/Autoform/Autoform";
import { Card } from "../components/lib/Card/Card";
import { Navbar } from "../components/lib/Navbar/Navbar";
import { Table, TableRow } from "../components/lib/Table/Table";

interface ConfigPageProps {
  getConfig: () => Promise<Config>;
  updateConfig: (value: Config) => Promise<any>;

  getQueryParams: () => URLSearchParams;
  setQueryParams: (params: URLSearchParams) => void;
}

interface ConfigPageStateBase {
  type: string;

  topLevelProperty?: string | null;
}

interface ConfigPageStateLoading extends ConfigPageStateBase {
  type: "loading";
}

interface ConfigPageStateLoaded extends ConfigPageStateBase {
  type: "loaded";
  initialValues: Config;
}

interface ConfigPageStateLoadingError extends ConfigPageStateBase {
  type: "loadingerror";
}

type ConfigPageState =
  | ConfigPageStateLoading
  | ConfigPageStateLoaded
  | ConfigPageStateLoadingError;

const FILE_PAGE_SPEC: FormSpec = {
  fields: [
    {
      name: "files",
      type: "ARRAY",
      headerFieldName: "fileName",
      itemTypes: {
        type: "OBJECT",
        name: "file",
        fields: [
          { type: "STRING", name: "fileName" },
          {
            type: "ARRAY",
            name: "fileTypes",
            itemTypes: {
              type: "STRING",
              name: "fileType",
            },
          },
        ],
      },
    },
  ],
};

export class ConfigPageComponent extends Component<
  ConfigPageProps,
  ConfigPageState
> {
  constructor(props: ConfigPageProps) {
    super(props);

    this.state = {
      type: "loading",
    };
  }

  async componentDidMount() {
    const cfg = await this.props.getConfig();
    this.setState({
      type: "loaded",
      topLevelProperty: "files",
      initialValues: cfg,
    });
  }

  render() {
    return (
      <div>
        <Navbar />
        <main role="main" className="ls-container">
          <div className="d-flex flex-row gap-6">
            <div className="shrink-1">
              <Card>
                <Table hoverable={true}>
                  <tbody>
                    <TableRow onClick={() => this.navigate("files")}>
                      <td>Files</td>
                    </TableRow>
                  </tbody>
                </Table>
              </Card>
            </div>
            <div className="grow-1 shrink-0" style={{ flexBasis: "80%" }}>
              {this.state.type === "loaded" && (
                <div>
                  {this.state.topLevelProperty === "files" && (
                    <Autoform
                      spec={FILE_PAGE_SPEC}
                      initialValues={this.state.initialValues}
                      onSubmit={async (v: Config) => {
                        await this.props.updateConfig(v);
                        await this.reload();
                      }}
                    ></Autoform>
                  )}
                </div>
              )}
            </div>
          </div>
        </main>
      </div>
    );
  }

  private navigate(topLevelProperty: string) {
    if (this.state.type !== "loaded") {
      return;
    }
    this.setState({ type: "loaded", topLevelProperty });
  }

  private async reload() {
    this.setState({
      type: "loading",
    });
    const cfg = await this.props.getConfig();
    this.setState({
      type: "loaded",
      initialValues: cfg,
    });
  }
}
