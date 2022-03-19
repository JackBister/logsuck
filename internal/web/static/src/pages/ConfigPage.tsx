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
import { Card, CardHeader } from "../components/lib/Card/Card";
import { Navbar } from "../components/lib/Navbar/Navbar";

interface ConfigPageProps {
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

  async componentDidMount() {}

  render() {
    return (
      <div>
        <Navbar />
        <main role="main" className="ls-container">
          {this.state.type === "loading" && <div>Loading...</div>}
          {this.state.type === "loadingerror" && <div>Error while loading</div>}
          {this.state.type === "loaded" && (
            <div className="w-100 d-flex flex-row align-start gap-6">
              <Card className="shrink-1">
                <CardHeader>Configuration</CardHeader>
              </Card>
              <div className="grow-1 shrink-0" style={{ flexBasis: "80%" }}>
                <Card>
                  <CardHeader>{this.state.topLevelProperty}</CardHeader>
                </Card>
              </div>
            </div>
          )}
        </main>
      </div>
    );
  }
}
