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

import { Component, createRef, h, JSX, Ref, RenderableProps } from "preact";
import { Button } from "./Button";

export interface DropdownProps {
  isOpen: boolean;
  onOpenStateChanged: (isOpen: boolean) => void;
  triggerText: string;
}

interface DropdownState {}

export class Dropdown extends Component<
  RenderableProps<DropdownProps>,
  DropdownState
> {
  private contentRef: Ref<HTMLDivElement>;

  constructor(props: RenderableProps<DropdownProps>) {
    super(props);

    this.clickHandler = this.clickHandler.bind(this);

    this.contentRef = createRef();
    this.state = {};
  }

  componentDidMount() {
    window.addEventListener("click", this.clickHandler);
  }

  componentWillUnmount() {
    window.removeEventListener("click", this.clickHandler);
  }

  render() {
    return (
      <div style={{ position: "relative", height: "100%" }}>
        <Button
          type="button"
          buttonType="secondary"
          onClick={(evt) => {
            evt.preventDefault();
            evt.stopPropagation();
            this.props.onOpenStateChanged(!this.props.isOpen);
          }}
          aria-haspopup="true"
          aria-expanded={this.props.isOpen}
          style={{
            display: "flex",
            flexDirection: "row",
            alignItems: "center",
            paddingRight: "6px",
          }}
        >
          <span style={{ marginRight: "10px" }}>{this.props.triggerText}</span>
          <span style={{ fontSize: "10px", marginTop: "2px" }}>&#x25bc;</span>
        </Button>
        {this.props.isOpen && (
          <div ref={this.contentRef} className="ls-dropdown-container">
            {this.props.children}
          </div>
        )}
      </div>
    );
  }

  private clickHandler(evt: MouseEvent) {
    if (!this.props.isOpen) {
      return;
    }
    const contentRef = this.contentRef as any;
    if (!contentRef || !contentRef.current) {
      return;
    }
    const contentRefCurrent = contentRef.current as HTMLDivElement;
    if (!evt.target) {
      return;
    }
    const target = evt.target as Node;
    if (!contentRefCurrent.contains(target)) {
      this.props.onOpenStateChanged(false);
    }
  }
}

export interface DropdownItemProps {
  isCurrent: boolean;
}

export const DropdownItem = (
  props: JSX.IntrinsicElements["button"] & DropdownItemProps
) => (
  <button
    {...props}
    className={`ls-dropdown-item ${props.isCurrent ? "ls-current" : ""}`}
  >
    {props.children}
  </button>
);
