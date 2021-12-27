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

import { h, RenderableProps } from "preact";

export interface TableProps {
  hoverable: boolean;
}

export const Table = (props: RenderableProps<TableProps>) => {
  let className = "ls-table ";
  if (props.hoverable) {
    className += "ls-table-hover";
  }
  return <table className={className}>{props.children}</table>;
};

export interface TableRowProps {
  onClick?: (evt: Event) => void;
}

export const TableRow = (props: RenderableProps<TableRowProps>) => (
  <tr
    tabIndex={props.onClick && 0}
    onClick={props.onClick}
    style={props.onClick && { cursor: "pointer" }}
    role={props.onClick && "button"}
    onKeyDown={
      props.onClick &&
      ((evt: KeyboardEvent) => {
        if (!props.onClick) {
          return;
        }
        if (evt.key === " " || evt.key === "Enter" || evt.key === "Spacebar") {
          props.onClick(evt);
        }
      })
    }
  >
    {props.children}
  </tr>
);
