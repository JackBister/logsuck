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

import { Component, h } from "preact";
import { Button } from "./lib/Button";

const DISABLED_CLASS = "ls-page-item ls-disabled";
const ENABLED_CLASS = "ls-page-item";

export interface PaginationProps {
  numberOfPages: number;
  currentPageIndex: number;
  onPageChanged: (newPageIndex: number) => void;
}

export interface PaginationState {}

export class Pagination extends Component<PaginationProps, PaginationState> {
  constructor(props: PaginationProps) {
    super(props);

    this.state = {};
  }

  render() {
    return (
      <ul className="ls-pagination mb-3">
        <li
          className={
            this.props.currentPageIndex === 0 ? DISABLED_CLASS : ENABLED_CLASS
          }
        >
          <Button
            type="button"
            buttonType="text"
            disabled={this.props.currentPageIndex === 0}
            onClick={() => this.props.onPageChanged(0)}
          >
            First
          </Button>
        </li>
        <li
          className={
            this.props.currentPageIndex === 0 ? DISABLED_CLASS : ENABLED_CLASS
          }
        >
          <Button
            type="button"
            buttonType="text"
            disabled={this.props.currentPageIndex === 0}
            onClick={() =>
              this.props.onPageChanged(this.props.currentPageIndex - 1)
            }
          >
            Previous
          </Button>
        </li>
        <li className="ls-page-item ls-active">
          <span>{this.props.currentPageIndex + 1}</span>
        </li>
        <li
          className={
            this.props.currentPageIndex === this.props.numberOfPages - 1
              ? DISABLED_CLASS
              : ENABLED_CLASS
          }
        >
          <Button
            type="button"
            buttonType="text"
            disabled={
              this.props.currentPageIndex === this.props.numberOfPages - 1
            }
            onClick={() =>
              this.props.onPageChanged(this.props.currentPageIndex + 1)
            }
          >
            Next
          </Button>
        </li>
        <li
          className={
            this.props.currentPageIndex === this.props.numberOfPages - 1
              ? DISABLED_CLASS
              : ENABLED_CLASS
          }
        >
          <Button
            type="button"
            buttonType="text"
            disabled={
              this.props.currentPageIndex === this.props.numberOfPages - 1
            }
            onClick={() =>
              this.props.onPageChanged(this.props.numberOfPages - 1)
            }
          >
            Last ({this.props.numberOfPages})
          </Button>
        </li>
      </ul>
    );
  }
}
