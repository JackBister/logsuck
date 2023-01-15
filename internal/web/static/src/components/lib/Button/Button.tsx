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

import { h, JSX } from "preact";
import {
  lsButton,
  lsButtonPrimary,
  lsButtonSecondary,
  lsButtonText,
} from "./Button.style.scss";

export interface ButtonProps {
  buttonType: "primary" | "secondary" | "text";
}

export const Button = (
  props: JSX.IntrinsicElements["button"] & ButtonProps
) => {
  let className = lsButton;
  if (props.buttonType === "primary") {
    className += " " + lsButtonPrimary;
  } else if (props.buttonType === "secondary") {
    className += " " + lsButtonSecondary;
  } else if (props.buttonType === "text") {
    className += " " + lsButtonText;
  }
  if (props.className) {
    className += " " + props.className;
  }
  return (
    <button type={props.type || "button"} {...props} className={className}>
      {props.children}
    </button>
  );
};
