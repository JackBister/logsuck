import { Component, h } from "preact";
import { FileTypeConfig } from "../../api/v1";
import { Button } from "../lib/Button/Button";
import { SimpleExpansionPanel } from "../lib/ExpansionPanel/ExpansionPanel";
import { Infobox } from "../lib/Infobox/Infobox";
import { Input } from "../lib/Input/Input";
import {
  configPage,
  configGroup,
  groupMargin,
} from "./FileTypeConfig.style.scss";

interface FileTypeConfigsProps {
  getFileTypeConfigs: () => Promise<FileTypeConfig[]>;
  updateFileTypeConfig: (cfg: FileTypeConfig) => Promise<any>;
}

interface FileTypeConfigsState {
  isLoaded: boolean;

  fileTypeConfigs?: FileTypeConfig[];
  messageType?: "info" | "error";
  message?: string;
}

interface FileTypeConfigProps {
  name: string;
  cfg: FileTypeConfig;
  onSave: (name: string, cfg: FileTypeConfig) => void;
}

const SingleFileTypeConfigComponent = ({
  name,
  cfg,
  onSave,
}: FileTypeConfigProps) => {
  const getInputValue = (form: HTMLFormElement, k: string) => {
    const v = form.elements.namedItem(k);
    console.log(v);
    return v instanceof HTMLInputElement ? v.value : null;
  };
  return (
    <SimpleExpansionPanel title={name}>
      <form
        className={configGroup + " " + groupMargin}
        onSubmit={async (evt) => {
          console.log("onsubmit");
          evt.preventDefault();
          const form = evt.target as HTMLFormElement;
          console.log(form.elements);
          const timeLayout = getInputValue(form, "timeLayout");
          const readInterval = getInputValue(form, "readInterval");
          const parserType = getInputValue(form, "parserType");
          const regexEventDelimiter = getInputValue(
            form,
            "regexEventDelimiter"
          );
          const regexFieldExtractors: string[] = [];
          for (const k of form.elements) {
            if (
              k instanceof HTMLInputElement &&
              k.name.startsWith("regexFieldExtractors[")
            ) {
              regexFieldExtractors.push(k.value);
            }
          }
          const fileTypeConfig: FileTypeConfig = {
            name: name,
            timeLayout: timeLayout || undefined,
            readInterval: readInterval || undefined,
            parser: {
              type: "Regex",
              regexConfig: {
                eventDelimiter: regexEventDelimiter || undefined,
                fieldExtractors: regexFieldExtractors,
              },
            },
          };

          onSave(name, fileTypeConfig);
        }}
      >
        <h2>{name}</h2>
        <label htmlFor={name + "-timeLayout"}>Time layout</label>
        <Input
          id={name + "-timeLayout"}
          name="timeLayout"
          placeholder="2006/01/02 15:04:05"
          value={cfg.timeLayout}
        />
        <label htmlFor={name + "-readInterval"}>Read interval</label>
        <Input
          id={name + "-readInterval"}
          name="readInterval"
          placeholder="1s"
          value={cfg.readInterval}
        />
        <label htmlFor={name + "-parserType"}>Parser type</label>
        <select
          id={name + "-parserType"}
          name="parserType"
          value={cfg.parser.type}
        >
          <option value="Regex">Regex</option>
        </select>

        {cfg.parser.type === "Regex" && (
          <div className={configGroup}>
            <label htmlFor={name + "-Regex-eventDelimiter"}>
              Event delimiter
            </label>
            <Input
              id={name + "-Regex-eventDelimiter"}
              name="regexEventDelimiter"
              placeholder="\n"
              value={cfg.parser.regexConfig.eventDelimiter}
            />

            <label>Field extractors</label>
            {cfg.parser.regexConfig.fieldExtractors.map((fd, i) => (
              <Input name={"regexFieldExtractors[" + i + "]"} value={fd} />
            ))}
          </div>
        )}

        <div>
          <Button buttonType="primary" type="submit">
            Save
          </Button>
        </div>
      </form>
    </SimpleExpansionPanel>
  );
};

export class FileTypeConfigsComponent extends Component<
  FileTypeConfigsProps,
  FileTypeConfigsState
> {
  constructor(props: FileTypeConfigsProps) {
    super(props);

    this.state = {
      isLoaded: false,
    };
  }

  async componentDidMount() {
    this.reload();
  }

  render() {
    return (
      <div className={configPage}>
        <h1>File types</h1>
        {this.state.message && this.state.messageType && (
          <div style={{ marginBottom: "2rem" }}>
            <Infobox type={this.state.messageType}>
              {this.state.message}
            </Infobox>
          </div>
        )}
        {this.state.isLoaded && this.state.fileTypeConfigs && (
          <div
            style={{ display: "flex", flexDirection: "column", gap: "2rem" }}
          >
            {this.state.fileTypeConfigs.map((ftc) => (
              <SingleFileTypeConfigComponent
                name={ftc.name}
                cfg={ftc}
                onSave={(name, cfg) => this.onSave(name, cfg)}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  private async reload() {
    const fileTypeConfig = await this.props.getFileTypeConfigs();
    this.setState({
      isLoaded: true,
      fileTypeConfigs: fileTypeConfig,
    });
  }

  private async onSave(name: string, cfg: FileTypeConfig) {
    try {
      await this.props.updateFileTypeConfig(cfg);
      await this.reload();
      this.setState({
        messageType: "info",
        message: `Successfully saved config for file type ${name}.`,
      });
    } catch (e) {
      this.setState({
        messageType: "error",
        message: `Got error when saving config for file type ${name}: ${e}.`,
      });
    }
  }
}
