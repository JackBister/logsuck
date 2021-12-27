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

import { Component, createRef, h, Ref, RenderableProps } from "preact";

export interface PopoverProps {
  isOpen: boolean;
  onOpenStateChanged: (isOpen: boolean) => void;
}

export interface PopoverState {}

export class Popover extends Component<
  RenderableProps<PopoverProps>,
  PopoverState
> {
  private contentRef: Ref<HTMLDivElement>;

  constructor(props: RenderableProps<PopoverProps>) {
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
      <div style={{ position: "relative" }}>
        {this.props.isOpen && (
          <div ref={this.contentRef} className="ls-popover-container">
            {this.props.children}
          </div>
        )}
      </div>
    );
  }

  private clickHandler(evt: MouseEvent) {
    if (!this.props.isOpen) {
      console.log("props");
      return;
    }
    const contentRef = this.contentRef as any;
    if (!contentRef || !contentRef.current) {
      console.log("cr");
      return;
    }
    const contentRefCurrent = contentRef.current as HTMLDivElement;
    if (!evt.target) {
      console.log("target");
      return;
    }
    const target = evt.target as Node;
    console.log(target);
    if (!contentRefCurrent.contains(target)) {
      this.props.onOpenStateChanged(false);
    }
  }
}
