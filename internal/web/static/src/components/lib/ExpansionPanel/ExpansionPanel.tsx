import { Component, h, RenderableProps } from "preact";

export interface ExpansionPanelProps {
  expanded: boolean;
  onExpansionStateChanged: (newState: boolean) => void;
  title: string;
}

interface ExpansionPanelState {}

export interface SimpleExpansionPanelProps {
  initialExpanded?: boolean;
  title: string;
}

interface SimpleExpansionPanelState {
  expanded: boolean;
}

export class SimpleExpansionPanel extends Component<
  SimpleExpansionPanelProps,
  SimpleExpansionPanelState
> {
  constructor(props: SimpleExpansionPanelProps) {
    super(props);

    this.state = {
      expanded: !!props.initialExpanded,
    };
  }

  render() {
    return (
      <ExpansionPanel
        expanded={this.state.expanded}
        onExpansionStateChanged={(expanded) => this.setState({ expanded })}
        title={this.props.title}
      >
        {this.props.children}
      </ExpansionPanel>
    );
  }
}

export class ExpansionPanel extends Component<
  RenderableProps<ExpansionPanelProps>,
  ExpansionPanelState
> {
  constructor(props: ExpansionPanelProps) {
    super(props);

    this.state = {};
  }

  render() {
    return (
      <div>
        <button
          onClick={() =>
            this.props.onExpansionStateChanged(!this.props.expanded)
          }
        >
          {this.props.title}
        </button>
        {this.props.expanded && <div>{this.props.children}</div>}
      </div>
    );
  }
}
