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

import {
  Accordion,
  Button,
  Flex,
  NativeSelect,
  NumberInput,
  Select,
  Space,
  Stack,
  Text,
  TextInput,
} from "@mantine/core";
import {
  Field,
  FieldArray,
  FieldArrayRenderProps,
  Formik,
  FormikProps,
  FormikValues,
} from "formik";
import { Component, h, RenderableProps } from "preact";

export type FormFieldType =
  | "ARRAY"
  | "BOOLEAN"
  | "ENUM"
  | "NUMBER"
  | "OBJECT"
  | "STRING";

export interface ConditionalField {
  key: string;
  value: any;
}

export interface FormFieldBase {
  type: FormFieldType;
  displayName?: string;
  name: string;
  readonly?: boolean;
  conditional?: ConditionalField;
}

export interface ArrayFormField extends FormFieldBase {
  type: "ARRAY";
  headerFieldName?: string;
  itemTypes: FormField;
}

export interface BooleanFormField extends FormFieldBase {
  type: "BOOLEAN";
}

export interface EnumFormField extends FormFieldBase {
  type: "ENUM";
  symbols: string[];
}

export interface NumberFormField extends FormFieldBase {
  type: "NUMBER";
}

export interface ObjectFormField extends FormFieldBase {
  type: "OBJECT";
  fields: FormField[];
}

export interface StringFormField extends FormFieldBase {
  type: "STRING";

  dynamicEnum: string;
}

export type FormField =
  | ArrayFormField
  | BooleanFormField
  | EnumFormField
  | NumberFormField
  | ObjectFormField
  | StringFormField;

export interface FormSpec {
  fields: FormField[];
}

function map<T>(field: FormField, mapper: (f: FormField) => T): T[] {
  const t = field.type;
  switch (t) {
    case "ARRAY":
      return [mapper(field)].concat(map(field.itemTypes, mapper));
    case "BOOLEAN":
      return [mapper(field)];
    case "ENUM":
      return [mapper(field)];
    case "NUMBER":
      return [mapper(field)];
    case "OBJECT":
      return [mapper(field)].concat(
        (field as ObjectFormField).fields.flatMap((f) => map(f, mapper))
      );
    case "STRING":
      return [mapper(field)];
    default:
      return exhaustiveSwitch(t);
  }
}

function notNull<T>(value: T | null): value is T {
  return value !== null;
}

function isIgnoredField(value: FormField) {
  return value.name === "$schema";
}

const FIELD_NAMES_ASSUMED_TO_BE_HEADINGS = ["key", "name", "fileName"];

export function jsonSchemaToFormSpec(
  name: string,
  jsonSchema: any
): FormField | null {
  if (name === "" && jsonSchema.type !== "object") {
    throw new Error(
      "only the root object of a JSON schema can have an empty name"
    );
  }
  const metadata = jsonSchema.autoform || {};
  if (jsonSchema.type === "string") {
    if (jsonSchema.enum && jsonSchema.enum.length > 0) {
      return {
        type: "ENUM",
        name,
        symbols: jsonSchema.enum as string[],
        readonly: metadata.readonly,
        conditional: metadata.conditional,
      } as EnumFormField;
    }
    if (metadata) {
      console.log("md", metadata);
    }
    return {
      type: "STRING",
      name,
      readonly: metadata.readonly,
      conditional: metadata.conditional,
      dynamicEnum: metadata.dynamicEnum,
    } as StringFormField;
  } else if (jsonSchema.type === "boolean") {
    return {
      type: "BOOLEAN",
      name,
      readonly: metadata.readonly,
      conditional: metadata.conditional,
    } as BooleanFormField;
  } else if (jsonSchema.type === "number") {
    return {
      type: "NUMBER",
      name,
      readonly: metadata.readonly,
      conditional: metadata.conditional,
    } as NumberFormField;
  } else if (jsonSchema.type === "array") {
    const itemType = jsonSchemaToFormSpec(name, jsonSchema.items);
    if (itemType === null) {
      return null;
    }
    let headerFieldName = undefined;
    if (itemType.type === "OBJECT") {
      const headerField = itemType.fields.find(
        (f) =>
          f.type === "STRING" &&
          FIELD_NAMES_ASSUMED_TO_BE_HEADINGS.includes(f.name)
      );
      headerFieldName = headerField?.name;
    }
    return {
      type: "ARRAY",
      name,
      headerFieldName,
      itemTypes: itemType,
      readonly: metadata.readonly,
      conditional: metadata.conditional,
    } as ArrayFormField;
  } else if (jsonSchema.type === "object") {
    const properties = jsonSchema.properties;
    if (!properties) {
      return {
        type: "OBJECT",
        name,
        fields: [],
        readonly: metadata.readonly,
        conditional: metadata.conditional,
      } as ObjectFormField;
    }
    const fields = Object.keys(properties)
      .map((k) => {
        const prop = properties[k];
        return jsonSchemaToFormSpec(k, prop);
      })
      .filter(notNull)
      .filter((f) => !isIgnoredField(f));
    return {
      type: "OBJECT",
      name,
      fields,
      readonly: metadata.readonly,
      conditional: metadata.conditional,
    } as ObjectFormField;
  } else {
    return null;
  }
}

interface AutoformFieldProps {
  level?: number;
  path: string;
  readonly?: boolean;
  spec: FormField;

  dynamicEnums?: { [key: string]: string[] };
  formikProps: FormikProps<any>;
}

interface AutoformFieldState {}

const getPath = (o: any, s: string): any => {
  s = s.replace(/\[(\w+)\]/g, ".$1"); // convert indexes to properties
  s = s.replace(/^\./, ""); // strip a leading dot
  const a = s.split(".");
  for (let i = 0, n = a.length; i < n; ++i) {
    const k = a[i];
    if (k in o) {
      o = o[k];
    } else {
      return;
    }
  }
  return o;
};

const Heading = (props: RenderableProps<{ level?: number }>) => {
  switch (props.level) {
    case 1:
      return <h1>{props.children}</h1>;
    case 2:
      return <h2>{props.children}</h2>;
    case 3:
      return <h3>{props.children}</h3>;
    case 4:
      return <h4>{props.children}</h4>;
    case 5:
      return <h5>{props.children}</h5>;
    default:
      return <h6>{props.children}</h6>;
  }
};

const escapeStringValue = (s: string) => {
  let escapedStringValue = JSON.stringify(s);
  return escapedStringValue?.substring(1, escapedStringValue.length - 1);
};

const unescapeStringValue = (s: string) => {
  return JSON.parse('"' + s + '"');
};

const getDefaultValue = (fs: FormField) => {
  const t = fs.type;
  switch (t) {
    case "ARRAY":
      return [];
    case "BOOLEAN":
      return false;
    case "ENUM":
      return (fs as EnumFormField).symbols[0];
    case "NUMBER":
      return undefined;
    case "OBJECT":
      const ret: Record<string, any> = {};
      for (const field of (fs as ObjectFormField).fields) {
        ret[field.name] = getDefaultValue(field);
      }
      return ret;
    case "STRING":
      return "";
    default:
      return exhaustiveSwitch(t);
  }
};

function exhaustiveSwitch(v: never): never {
  throw new Error("Unhandled type=" + v);
}

function StringField(props: AutoformFieldProps) {
  if (props.spec.type !== "STRING") {
    throw new Error(
      "Attempted to use StringField on non-STRING field=" + props.spec
    );
  }
  const escapedStringValue = escapeStringValue(
    getPath(props.formikProps.values, props.path)
  );
  const isDynamicEnum = props.spec.type === "STRING" && props.spec.dynamicEnum;
  const isWaitingForDynamicEnum = isDynamicEnum && !props.dynamicEnums;
  if (isWaitingForDynamicEnum) {
    return (
      <div>
        <label htmlFor={props.path}>
          {props.spec.displayName || props.spec.name}
        </label>
        <Field
          as={TextInput}
          name={props.path}
          disabled={true}
          readonly={props.readonly || props.spec.readonly}
          value={escapedStringValue}
        ></Field>
      </div>
    );
  }
  if (isDynamicEnum && props.dynamicEnums) {
    const dynamicEnumValues = props.dynamicEnums[props.spec.dynamicEnum];
    return (
      <div>
        <label htmlFor={props.path}>
          {props.spec.displayName || props.spec.name}
        </label>
        <Field
          as={NativeSelect}
          name={props.path}
          disabled={props.readonly || props.spec.readonly}
          readonly={props.readonly || props.spec.readonly}
          data={dynamicEnumValues.map((s) => ({
            value: s,
            label: s,
          }))}
        ></Field>
      </div>
    );
  }
  return (
    <div>
      <label htmlFor={props.path}>
        {props.spec.displayName || props.spec.name}
      </label>
      <Field
        as={TextInput}
        name={props.path}
        onChange={(evt: InputEvent) => {
          if (!evt.target || !(evt.target as any).value) {
            return;
          }
          props.formikProps.setFieldValue(
            props.path,
            unescapeStringValue((evt.target as any).value)
          );
        }}
        disabled={props.readonly || props.spec.readonly}
        readonly={props.readonly || props.spec.readonly}
        value={escapedStringValue}
      ></Field>
    </div>
  );
}

class AutoformField extends Component<AutoformFieldProps, AutoformFieldState> {
  constructor(props: AutoformFieldProps) {
    super(props);

    this.state = {};
  }

  private pushArrayItem(fa: FieldArrayRenderProps) {
    if (this.props.spec.type !== "ARRAY") {
      return;
    }
    fa.push(getDefaultValue(this.props.spec.itemTypes));
  }

  render() {
    return (
      <div>
        {this.props.spec.type === "ARRAY" && (
          <FieldArray
            name={this.props.path}
            render={(fa: FieldArrayRenderProps) => (
              <div>
                <Flex direction="row" align="center">
                  <Heading level={this.props.level}>
                    {this.props.spec.displayName || this.props.spec.name}
                  </Heading>
                  {!this.props.readonly && !this.props.spec.readonly && (
                    <Button
                      variant="subtle"
                      onClick={() => this.pushArrayItem(fa)}
                    >
                      Add
                    </Button>
                  )}
                </Flex>
                <Accordion variant="contained" multiple={true}>
                  {(getPath(fa.form.values, this.props.path) as any[])?.map(
                    (a, i) => {
                      if (this.props.spec.type !== "ARRAY") {
                        return null;
                      }
                      if (this.props.spec.itemTypes.type === "OBJECT") {
                        return (
                          <Accordion.Item value={i.toString()}>
                            <Accordion.Control>
                              <Flex direction="row" gap="md">
                                <Text>
                                  {this.props.spec.headerFieldName &&
                                    getPath(a, this.props.spec.headerFieldName)}
                                </Text>
                                {!this.props.readonly &&
                                  !this.props.spec.readonly && (
                                    <Button
                                      variant="subtle"
                                      compact
                                      onClick={() => fa.remove(i)}
                                    >
                                      Remove
                                    </Button>
                                  )}
                              </Flex>
                            </Accordion.Control>
                            <Accordion.Panel>
                              <AutoformField
                                key={i}
                                level={(this.props.level || 0) + 1}
                                path={`${this.props.path}[${i}]`}
                                readonly={
                                  this.props.readonly ||
                                  this.props.spec.readonly
                                }
                                spec={this.props.spec.itemTypes}
                                dynamicEnums={this.props.dynamicEnums}
                                formikProps={this.props.formikProps}
                              ></AutoformField>
                            </Accordion.Panel>
                          </Accordion.Item>
                        );
                      }
                      return (
                        <Flex direction="row" align="center">
                          <AutoformField
                            key={i}
                            level={(this.props.level || 0) + 1}
                            path={`${this.props.path}[${i}]`}
                            readonly={
                              this.props.readonly || this.props.spec.readonly
                            }
                            spec={this.props.spec.itemTypes}
                            dynamicEnums={this.props.dynamicEnums}
                            formikProps={this.props.formikProps}
                          ></AutoformField>
                          {!this.props.readonly &&
                            !this.props.spec.readonly && (
                              <Button
                                variant="subtle"
                                compact
                                style={{ marginTop: "25px" }}
                                onClick={() => fa.remove(i)}
                              >
                                Remove
                              </Button>
                            )}
                        </Flex>
                      );
                    }
                  )}
                </Accordion>
              </div>
            )}
          ></FieldArray>
        )}
        {this.props.spec.type === "OBJECT" && (
          <Stack>
            {this.props.spec.fields.map((f, i) => {
              if (f.conditional) {
                const currentValue = getPath(
                  this.props.formikProps.values,
                  this.props.path + "." + f.conditional.key
                );
                if (currentValue !== f.conditional.value) {
                  return null;
                }
              }
              return (
                <AutoformField
                  key={i}
                  level={(this.props.level || 0) + 1}
                  path={`${this.props.path}.${f.name}`}
                  readonly={this.props.readonly || this.props.spec.readonly}
                  spec={f}
                  dynamicEnums={this.props.dynamicEnums}
                  formikProps={this.props.formikProps}
                ></AutoformField>
              );
            })}
          </Stack>
        )}
        {this.props.spec.type === "BOOLEAN" && (
          <div>
            <label htmlFor={this.props.path}>
              {this.props.spec.displayName || this.props.spec.name}
            </label>
            <Field
              as={Select}
              name={this.props.path}
              disabled={this.props.readonly || this.props.spec.readonly}
              readonly={this.props.readonly || this.props.spec.readonly}
              onChange={(v: boolean) => {
                this.props.formikProps.setFieldValue(this.props.path, v);
              }}
              value={getPath(this.props.formikProps.values, this.props.path)}
              data={[
                { value: false, label: "false" },
                { value: true, label: "true" },
              ]}
            ></Field>
          </div>
        )}
        {this.props.spec.type === "ENUM" && (
          <div>
            <label htmlFor={this.props.path}>
              {this.props.spec.displayName || this.props.spec.name}
            </label>
            <Field
              as={NativeSelect}
              name={this.props.path}
              disabled={this.props.readonly || this.props.spec.readonly}
              readonly={this.props.readonly || this.props.spec.readonly}
              data={this.props.spec.symbols.map((s) => ({
                value: s,
                label: s,
              }))}
            ></Field>
          </div>
        )}
        {this.props.spec.type === "NUMBER" && (
          <div>
            <label htmlFor={this.props.path}>
              {this.props.spec.displayName || this.props.spec.name}
            </label>
            <Field
              as={NumberInput}
              name={this.props.path}
              disabled={this.props.readonly || this.props.spec.readonly}
              readonly={this.props.readonly || this.props.spec.readonly}
            ></Field>
          </div>
        )}
        {this.props.spec.type === "STRING" && (
          <StringField
            level={this.props.level}
            path={this.props.path}
            readonly={this.props.readonly}
            spec={this.props.spec}
            dynamicEnums={this.props.dynamicEnums}
            formikProps={this.props.formikProps}
          />
        )}
      </div>
    );
  }
}

export interface AutoformProps<Values> {
  initialValues: Values;
  onSubmit: (v: Values) => void;
  readonly?: boolean;
  spec: FormSpec;

  getDynamicEnum: (enumName: string) => Promise<string[]>;
}

interface AutoformState {
  dynamicEnums?: { [key: string]: string[] };
}

export class Autoform<Values extends FormikValues> extends Component<
  AutoformProps<Values>,
  AutoformState
> {
  constructor(props: AutoformProps<Values>) {
    super(props);

    this.state = {};
  }

  async componentDidMount() {
    const allDynamicEnums = this.props.spec.fields
      .flatMap((f) =>
        map(f, (f2) => (f2.type === "STRING" ? f2.dynamicEnum : undefined))
      )
      .filter((f) => !!f) as string[];
    const dynamicEnums = {} as { [key: string]: string[] };
    await Promise.all(
      allDynamicEnums.map((s) =>
        this.props
          .getDynamicEnum(s)
          .then((values) => (dynamicEnums[s] = values))
      )
    );
    this.setState({ dynamicEnums });
  }

  render() {
    return (
      <div>
        <Formik
          initialValues={this.props.initialValues}
          onSubmit={(values: Values) => this.props.onSubmit(values)}
        >
          {(p: FormikProps<Values>) => (
            <form onSubmit={p.handleSubmit}>
              {this.props.spec.fields.map((f) => (
                <AutoformField
                  path={f.name}
                  spec={f}
                  level={1}
                  readonly={this.props.readonly}
                  dynamicEnums={this.state.dynamicEnums}
                  formikProps={p}
                ></AutoformField>
              ))}
              <Space h="md" />
              {!this.props.readonly &&
                this.props.spec.fields.filter((f) => !f.readonly).length !==
                  0 && (
                  <Flex gap="md">
                    <Button type="submit">Save</Button>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => p.resetForm()}
                    >
                      Reset
                    </Button>
                  </Flex>
                )}
            </form>
          )}
        </Formik>
      </div>
    );
  }
}
