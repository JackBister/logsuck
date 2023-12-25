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

import { LogsuckAppShell } from "../components/LogsuckAppShell";
import { Alert, Card, Table } from "@mantine/core";
import { TableRow } from "../components/TableRow";

interface ConfigPageProps {
  getConfig: () => Promise<any>;
  updateConfig: (value: any) => Promise<any>;
  getConfigSchema: () => Promise<any>;
  getDynamicEnum: (enumName: string) => Promise<string[]>;

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
  initialValues: any;
  form: ObjectFormField;
}

interface ConfigPageStateLoadingError extends ConfigPageStateBase {
  type: "loadingerror";
}

type ConfigPageState =
  | ConfigPageStateLoading
  | ConfigPageStateLoaded
  | ConfigPageStateLoadingError;

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
    const schema = await this.props.getConfigSchema();
    const spec = jsonSchemaToFormSpec("", schema) as ObjectFormField;
    const cfg = await this.props.getConfig();
    const queryParams = this.props.getQueryParams();
    let topLevelProperty = null;
    if (queryParams.has("topLevelProperty")) {
      topLevelProperty = queryParams.get("topLevelProperty");
    }
    this.setState({
      type: "loaded",
      topLevelProperty,
      initialValues: cfg,
      form: spec,
    });
  }

  render() {
    if (this.state.type === "loading") {
      return (
        <LogsuckAppShell>
          <p>Loading...</p>
        </LogsuckAppShell>
      );
    }
    if (this.state.type === "loadingerror") {
      return (
        <LogsuckAppShell>
          <Alert title="Error" color="red">
            Something went wrong. Try reloading the page.
          </Alert>
        </LogsuckAppShell>
      );
    }
    const currentField = this.state.form.fields.filter(
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
      <LogsuckAppShell>
        <div className="d-flex flex-row gap-6">
          <div className="shrink-1">
            <Card>
              <Table highlightOnHover>
                <tbody>
                  {this.state.form.fields.map((f) => (
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
              <Card>
                {this.state.initialValues.forceStaticConfig && (
                  <div className="mb-3">
                    <Alert>
                      "forceStaticConfig" is set in the configuration. The
                      configuration is in read only mode. In order to modify it,
                      set "forceStaticConfig" to false in the JSON
                      configuration.
                    </Alert>
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
                    onSubmit={async (v: any) => {
                      await this.props.updateConfig(v);
                      await this.reload();
                    }}
                    getDynamicEnum={this.props.getDynamicEnum}
                    readonly={this.state.initialValues.forceStaticConfig}
                  ></Autoform>
                )}
              </Card>
            )}
          </div>
        </div>
      </LogsuckAppShell>
    );
  }

  private navigate(topLevelProperty: string) {
    if (this.state.type !== "loaded") {
      return;
    }
    this.props.setQueryParams(
      new URLSearchParams({
        topLevelProperty,
      })
    );
    this.setState({ type: "loaded", topLevelProperty });
  }

  private async reload() {
    this.setState({
      type: "loading",
    });
    const schema = await this.props.getConfigSchema();
    const spec = jsonSchemaToFormSpec("", schema) as ObjectFormField;
    const cfg = await this.props.getConfig();
    this.setState({
      type: "loaded",
      initialValues: cfg,
      form: spec,
    });
  }
}
