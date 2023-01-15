import {
  ArrayHelpers,
  Field,
  FieldArray,
  FieldArrayRenderProps,
  Formik,
  FormikProps,
  FormikValues,
} from "formik";
import { Component, createRef, h, Ref, RenderableProps } from "preact";
import { Button } from "../lib/Button/Button";
import { SimpleExpansionPanel } from "../lib/ExpansionPanel/ExpansionPanel";
import { Input } from "../lib/Input/Input";
import { autoform, formGroup, level, level1 } from "./Autoform.style.scss";

export type FormFieldType = "ARRAY" | "ENUM" | "OBJECT" | "STRING";

export interface FormFieldBase {
  type: FormFieldType;
  displayName?: string;
  name: string;
}

export interface ArrayFormField extends FormFieldBase {
  type: "ARRAY";
  headerFieldName?: string;
  itemTypes: FormField;
}

export interface EnumFormField extends FormFieldBase {
  type: "ENUM";
  symbols: string[];
}

export interface ObjectFormField extends FormFieldBase {
  type: "OBJECT";
  fields: FormField[];
}

export interface StringFormField extends FormFieldBase {
  type: "STRING";
}

export type FormField =
  | ArrayFormField
  | EnumFormField
  | ObjectFormField
  | StringFormField;

export interface FormSpec {
  fields: FormField[];
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
  if (jsonSchema.type === "string") {
    if (jsonSchema.enum && jsonSchema.enum.length > 0) {
      return {
        type: "ENUM",
        name,
        symbols: jsonSchema.enum as string[],
      } as EnumFormField;
    }
    return {
      type: "STRING",
      name,
    } as StringFormField;
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
    } as ArrayFormField;
  } else if (jsonSchema.type === "object") {
    const properties = jsonSchema.properties;
    if (!properties) {
      return {
        type: "OBJECT",
        name,
        fields: [],
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

class AutoformField extends Component<AutoformFieldProps, AutoformFieldState> {
  constructor(props: AutoformFieldProps) {
    super(props);

    this.state = {};
  }

  private pushArrayItem(fa: FieldArrayRenderProps) {
    if (this.props.spec.type !== "ARRAY") {
      return;
    }
    if (this.props.spec.itemTypes.type === "STRING") {
      fa.push("");
    } else {
      fa.push({});
    }
  }

  render() {
    let escapedStringValue = "";
    if (this.props.spec.type === "STRING") {
      escapedStringValue = escapeStringValue(
        getPath(this.props.formikProps.values, this.props.path)
      );
    }
    return (
      <div>
        {this.props.spec.type === "ARRAY" && (
          <FieldArray
            name={this.props.path}
            render={(fa: FieldArrayRenderProps) => (
              <div
                className={level + " " + (this.props.level === 1 ? level1 : "")}
              >
                <div className="d-flex flex-row align-end justify-between mb-3">
                  <Heading level={this.props.level}>
                    {this.props.spec.displayName || this.props.spec.name}
                  </Heading>
                  {!this.props.readonly && (
                    <Button
                      buttonType="text"
                      onClick={() => this.pushArrayItem(fa)}
                    >
                      Add
                    </Button>
                  )}
                </div>
                {(getPath(fa.form.values, this.props.path) as any[])?.map(
                  (a, i) => {
                    if (this.props.spec.type !== "ARRAY") {
                      return null;
                    }
                    if (this.props.spec.itemTypes.type === "OBJECT") {
                      return (
                        <SimpleExpansionPanel
                          title={
                            (this.props.spec.headerFieldName &&
                              getPath(a, this.props.spec.headerFieldName)) ||
                            i
                          }
                        >
                          <AutoformField
                            key={i}
                            level={(this.props.level || 0) + 1}
                            path={`${this.props.path}[${i}]`}
                            readonly={this.props.readonly}
                            spec={this.props.spec.itemTypes}
                            formikProps={this.props.formikProps}
                          ></AutoformField>
                        </SimpleExpansionPanel>
                      );
                    }
                    return (
                      <div className="d-flex flex-row justify-start align-center">
                        <AutoformField
                          key={i}
                          level={(this.props.level || 0) + 1}
                          path={`${this.props.path}[${i}]`}
                          readonly={this.props.readonly}
                          spec={this.props.spec.itemTypes}
                          formikProps={this.props.formikProps}
                        ></AutoformField>
                        {!this.props.readonly && (
                          <Button
                            buttonType="text"
                            onClick={() => fa.remove(i)}
                          >
                            X
                          </Button>
                        )}
                      </div>
                    );
                  }
                )}
              </div>
            )}
          ></FieldArray>
        )}
        {this.props.spec.type === "OBJECT" && (
          <div className={level + " " + (this.props.level === 1 ? level1 : "")}>
            {this.props.spec.fields.map((f, i) => (
              <AutoformField
                key={i}
                level={(this.props.level || 0) + 1}
                path={`${this.props.path}.${f.name}`}
                readonly={this.props.readonly}
                spec={f}
                formikProps={this.props.formikProps}
              ></AutoformField>
            ))}
          </div>
        )}
        {this.props.spec.type === "ENUM" && (
          <div className={formGroup}>
            <label htmlFor={this.props.path}>
              {this.props.spec.displayName || this.props.spec.name}
            </label>
            <Field
              as="select"
              name={this.props.path}
              disabled={this.props.readonly}
              readonly={this.props.readonly}
            >
              {this.props.spec.symbols.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </Field>
          </div>
        )}
        {this.props.spec.type === "STRING" && (
          <div className={formGroup}>
            <label htmlFor={this.props.path}>
              {this.props.spec.displayName || this.props.spec.name}
            </label>
            <Field
              as={Input}
              name={this.props.path}
              type="text"
              onChange={(evt: InputEvent) => {
                if (!evt.target || !(evt.target as any).value) {
                  return;
                }
                this.props.formikProps.setFieldValue(
                  this.props.path,
                  unescapeStringValue((evt.target as any).value)
                );
              }}
              disabled={this.props.readonly}
              readonly={this.props.readonly}
              value={escapedStringValue}
            ></Field>
          </div>
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
}

interface AutoformState {}

export class Autoform<Values extends FormikValues> extends Component<
  AutoformProps<Values>,
  AutoformState
> {
  constructor(props: AutoformProps<Values>) {
    super(props);

    this.state = {};
  }

  render() {
    return (
      <div className={autoform}>
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
                  formikProps={p}
                ></AutoformField>
              ))}
              {!this.props.readonly && (
                <div>
                  <Button type="submit" buttonType="primary">
                    Save
                  </Button>
                  <Button
                    type="button"
                    buttonType="secondary"
                    onClick={() => p.resetForm()}
                  >
                    Reset
                  </Button>
                </div>
              )}
            </form>
          )}
        </Formik>
      </div>
    );
  }
}
