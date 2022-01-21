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
import { TimeSelect } from "./TimeSelect";
import { TimeSelection } from "../models/TimeSelection";
import { createSearchUrl } from "../createSearchUrl";
import { Input, InputGroup } from "./lib/Input/Input";
import { Button } from "./lib/Button/Button";

export interface SearchInputProps {
  isButtonDisabled: boolean;
  searchString: string;
  setSearchString: (str: string) => void;
  selectedTime: TimeSelection;
  setSelectedTime: (ts: TimeSelection) => void;

  onSearch: () => void;
}

export const SearchInput = (props: SearchInputProps) => (
  <form
    className="mb-3"
    onSubmit={(evt) => {
      evt.preventDefault();
      props.onSearch();
    }}
  >
    <label htmlFor="searchinput">Search</label>
    <InputGroup>
      <Input
        id="searchinput"
        name="searchinput"
        type="text"
        onInput={(evt) => props.setSearchString((evt.target as any).value)}
        value={props.searchString}
      />
      <TimeSelect
        selection={props.selectedTime}
        onTimeSelected={(ts) => props.setSelectedTime(ts)}
      />
      <Button
        disabled={props.isButtonDisabled}
        type="submit"
        buttonType="primary"
      >
        Search
      </Button>
    </InputGroup>
  </form>
);

export interface RedirectSearchInputProps {
  navigateTo: (url: string) => void;
}

interface RedirectSearchInputState {
  searchString: string;
  timeSelection: TimeSelection;
}

/**
 * RedirectSearchInput is an easier to use version of SearchInput which can be used on pages which don't need to do anything special with the input.
 * RedirectSearchInput will navigate to the resulting search URL when the search button is clicked.
 */
export class RedirectSearchInput extends Component<
  RedirectSearchInputProps,
  RedirectSearchInputState
> {
  constructor(props: RedirectSearchInputProps) {
    super(props);
    this.state = {
      searchString: "",
      timeSelection: {
        relativeTime: "-15m",
      },
    };
  }

  render() {
    return (
      <SearchInput
        isButtonDisabled={false}
        searchString={this.state.searchString}
        setSearchString={(str) => this.setState({ searchString: str })}
        selectedTime={this.state.timeSelection}
        setSelectedTime={(ts) => this.setState({ timeSelection: ts })}
        onSearch={() =>
          this.props.navigateTo(
            createSearchUrl(this.state.searchString, this.state.timeSelection)
          )
        }
      />
    );
  }
}
