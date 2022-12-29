import {
  ArrayHelpers,
  Field,
  FieldArray,
  FieldArrayRenderProps,
  Formik,
  FormikProps,
  FormikValues,
} from "formik";
import { Component, h, RenderableProps } from "preact";
import { Button } from "../lib/Button/Button";
import { SimpleExpansionPanel } from "../lib/ExpansionPanel/ExpansionPanel";
import { Input } from "../lib/Input/Input";
import { formGroup } from "./Autoform.style.scss";

export type FormFieldType = "ARRAY" | "OBJECT" | "STRING";

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

export interface ObjectFormField extends FormFieldBase {
  type: "OBJECT";
  fields: FormField[];
}

export interface StringFormField extends FormFieldBase {
  type: "STRING";
}

export type FormField = ArrayFormField | ObjectFormField | StringFormField;

export interface FormSpec {
  fields: FormField[];
}

interface AutoformFieldProps {
  level?: number;
  path: string;
  spec: FormField;
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

class AutoformField extends Component<AutoformFieldProps, AutoformFieldState> {
  constructor(props: AutoformFieldProps) {
    super(props);

    this.state = {};
  }

  render() {
    return (
      <div>
        {this.props.spec.type === "ARRAY" && (
          <FieldArray
            name={this.props.path}
            render={(fa: FieldArrayRenderProps) => (
              <div>
                <div className="d-flex flex-row align-end justify-between mb-3">
                  <Heading level={this.props.level}>
                    {this.props.spec.displayName || this.props.spec.name}
                  </Heading>
                  <Button buttonType="text">Add</Button>
                </div>
                {(getPath(fa.form.values, this.props.path) as any[]).map(
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
                            spec={this.props.spec.itemTypes}
                          ></AutoformField>
                        </SimpleExpansionPanel>
                      );
                    }
                    return (
                      <AutoformField
                        key={i}
                        level={(this.props.level || 0) + 1}
                        path={`${this.props.path}[${i}]`}
                        spec={this.props.spec.itemTypes}
                      ></AutoformField>
                    );
                  }
                )}
              </div>
            )}
          ></FieldArray>
        )}
        {this.props.spec.type === "OBJECT" && (
          <div>
            {this.props.spec.fields.map((f, i) => (
              <AutoformField
                key={i}
                level={(this.props.level || 0) + 1}
                path={`${this.props.path}.${f.name}`}
                spec={f}
              ></AutoformField>
            ))}
          </div>
        )}
        {this.props.spec.type === "STRING" && (
          <div className={formGroup}>
            <label htmlFor={this.props.path}>
              {this.props.spec.displayName || this.props.spec.name}
            </label>
            <Field as={Input} name={this.props.path} type="text"></Field>
          </div>
        )}
      </div>
    );
  }
}

export interface AutoformProps<Values> {
  initialValues: Values;
  onSubmit: (v: Values) => void;
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
      <div>
        <Formik
          initialValues={this.props.initialValues}
          onSubmit={(values: Values) => this.props.onSubmit(values)}
        >
          {(p: FormikProps<Values>) => (
            <form onSubmit={p.handleSubmit}>
              {this.props.spec.fields.map((f) => (
                <AutoformField path={f.name} spec={f} level={1}></AutoformField>
              ))}
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
            </form>
          )}
        </Formik>
      </div>
    );
  }
}
