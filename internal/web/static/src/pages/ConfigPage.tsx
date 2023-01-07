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
import {
  Autoform,
  FormSpec,
  jsonSchemaToFormSpec,
  ObjectFormField,
} from "../components/Autoform/Autoform";
import { Card } from "../components/lib/Card/Card";
import { Navbar } from "../components/lib/Navbar/Navbar";
import { Table, TableRow } from "../components/lib/Table/Table";

import * as configSchema from "../../../../../logsuck-config.schema.json";
import { LogsuckConfig } from "../api/config";
import { Infobox } from "../components/lib/Infobox/Infobox";

interface ConfigPageProps {
  getConfig: () => Promise<LogsuckConfig>;
  updateConfig: (value: LogsuckConfig) => Promise<any>;

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
  initialValues: LogsuckConfig;
}

interface ConfigPageStateLoadingError extends ConfigPageStateBase {
  type: "loadingerror";
}

type ConfigPageState =
  | ConfigPageStateLoading
  | ConfigPageStateLoaded
  | ConfigPageStateLoadingError;

const CONFIG_SCHEMA_SPEC = jsonSchemaToFormSpec(
  "",
  configSchema
) as ObjectFormField;

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
      topLevelProperty: null,
      initialValues: cfg,
    });
  }

  render() {
    const currentField = CONFIG_SCHEMA_SPEC.fields.filter(
      (f) => f.name === this.state.topLevelProperty
    )[0];
    let currentSpec: FormSpec | undefined;
    if (currentField) {
      currentSpec = {
        type: "OBJECT",
        name: "",
        fields: [
          {
            ...currentField,
            name: this.state.topLevelProperty,
          },
        ],
      } as FormSpec;
    }
    return (
      <div>
        <Navbar />
        <main role="main" className="ls-container">
          <div className="d-flex flex-row gap-6">
            <div className="shrink-1">
              <Card>
                <Table hoverable={true}>
                  <tbody>
                    {CONFIG_SCHEMA_SPEC.fields.map((f) => (
                      <TableRow onClick={() => this.navigate(f.name)}>
                        <td>{f.name}</td>
                      </TableRow>
                    ))}
                  </tbody>
                </Table>
              </Card>
            </div>
            <div className="grow-1 shrink-0" style={{ flexBasis: "80%" }}>
              {this.state.type === "loaded" && (
                <div>
                  {this.state.initialValues.forceStaticConfig && (
                    <div className="mb-3">
                      <Infobox type="info">
                        "forceStaticConfig" is set in the configuration. The
                        configuration is in read only mode. In order to modify
                        it, set "forceStaticConfig" to false in the JSON
                        configuration.
                      </Infobox>
                    </div>
                  )}
                  {!currentSpec && (
                    <p>
                      Choose one of the properties on the left to edit your
                      configuration.
                    </p>
                  )}
                  {currentSpec && (
                    <Autoform
                      key={this.state.topLevelProperty}
                      spec={currentSpec}
                      initialValues={this.state.initialValues}
                      onSubmit={async (v: LogsuckConfig) => {
                        await this.props.updateConfig(v);
                        await this.reload();
                      }}
                      readonly={this.state.initialValues.forceStaticConfig}
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
