/**
 * Copyright 2023 Jack Bister
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { h, Fragment } from "preact";
import { useState } from "preact/hooks";
import { Button } from "@mantine/core";

const Indent = (props: { level: number }) => {
  return <Fragment>{" ".repeat(props.level * 2)}</Fragment>;
};

const JsonArrayView = (props: {
  arr: any[];
  indentInitial: boolean;
  indentLevel: number;
}) => {
  const [isCollapsed, setCollapsed] = useState(false);
  const arr = props.arr as any[];
  return (
    <Fragment>
      {props.indentInitial && <Indent level={props.indentLevel} />}
      {"["}
      <Button
        variant="light"
        compact
        size="sm"
        onClick={() => setCollapsed(!isCollapsed)}
      >
        {isCollapsed ? "+" : "-"}
      </Button>
      {isCollapsed ? (
        "..."
      ) : (
        <Fragment>
          <br />
          {arr.map((v, i) => (
            <Fragment>
              <JsonAnyView
                indentInitial={true}
                indentLevel={props.indentLevel + 1}
                value={v}
              />
              {i !== arr.length - 1 && ","}
              <br />
            </Fragment>
          ))}
          <Indent level={props.indentLevel} />
        </Fragment>
      )}
      {"]"}
    </Fragment>
  );
};

const JsonObjectView = (props: {
  obj: any;
  indentInitial: boolean;
  indentLevel: number;
}) => {
  const [isCollapsed, setCollapsed] = useState(false);
  return (
    <Fragment>
      {props.indentInitial && <Indent level={props.indentLevel} />}
      {"{"}
      <Button
        variant="light"
        compact
        size="sm"
        onClick={() => setCollapsed(!isCollapsed)}
      >
        {isCollapsed ? "+" : "-"}
      </Button>
      {isCollapsed ? (
        "..."
      ) : (
        <Fragment>
          <br />
          {Object.keys(props.obj).map((k, i) => (
            <Fragment>
              <Indent level={props.indentLevel + 1} />"{k}":&nbsp;
              <JsonAnyView
                indentInitial={false}
                indentLevel={props.indentLevel + 1}
                value={props.obj[k]}
              />
              {i !== Object.keys(props.obj).length - 1 && ","}
              <br />
            </Fragment>
          ))}
          <Indent level={props.indentLevel} />
        </Fragment>
      )}
      {"}"}
    </Fragment>
  );
};

const JsonAnyView = (props: {
  value: any;
  indentInitial: boolean;
  indentLevel: number;
}) => {
  if (Array.isArray(props.value)) {
    return (
      <JsonArrayView
        indentInitial={props.indentInitial}
        indentLevel={props.indentLevel}
        arr={props.value}
      />
    );
  }
  if (props.value instanceof Object) {
    return (
      <JsonObjectView
        indentInitial={props.indentInitial}
        indentLevel={props.indentLevel}
        obj={props.value}
      />
    );
  }
  if (typeof props.value === "string") {
    return (
      <Fragment>
        {props.indentInitial && <Indent level={props.indentLevel} />}"
        {props.value}"
      </Fragment>
    );
  }
  return (
    <Fragment>
      {props.indentInitial && <Indent level={props.indentLevel} />}
      {props.value}
    </Fragment>
  );
};

export const JsonView = (props: { value: any }) => {
  return (
    <pre>
      <JsonAnyView indentInitial={true} indentLevel={0} value={props.value} />
    </pre>
  );
};
