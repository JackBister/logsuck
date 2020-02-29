import { Component, h } from "preact";

const MARGIN_FROM_PARENT_PX = 16;

interface PopoverProps {
    direction: 'right'; // TODO: Other directions
    heading: string;
    isOpen: boolean;
    widthPx: number;
}

interface PopoverState {
    // This is a bit hacky, used to keep heading while the popover fades out
    // assumes that the heading is set to '' when isOpen is set to false
    latestHeading: string;

    size: 'xl' | 'not-xl';
}

const PARENT_XL_STYLE = {
    position: 'absolute',
    'z-index': 999,
    transition: '.15s linear opacity',
    top: 'unset',
    left: 'unset',
    'max-width': 'unset',
};

const FADE_STYLE = {
    position: 'fixed',
    left: 0,
    right: 0,
    top: 0,
    bottom: 0,
    'background-color': 'rgba(0, 0, 0, 0.3)',
    'z-index': 998,
};

export class Popover extends Component<PopoverProps, PopoverState> {

    private mediaQuery: MediaQueryList;

    constructor(props: PopoverProps) {
        super(props);

        this.mediaQueryCallback = this.mediaQueryCallback.bind(this);
        this.mediaQuery = window.matchMedia('(min-width: 1200px)');

        this.state = {
            latestHeading: props.heading,
            size: this.mediaQuery.matches ? 'xl' : 'not-xl'
        };
    }

    private mediaQueryCallback(evt: MediaQueryListEvent) {
        this.setState({ size: evt.matches ? 'xl' : 'not-xl' });
    }

    componentDidMount() {
        this.mediaQuery.addEventListener('change', this.mediaQueryCallback);
    }

    componentWillUnmount() {
        this.mediaQuery.removeEventListener('change', this.mediaQueryCallback);
    }

    componentWillUpdate(nextProps: PopoverProps) {
        if (nextProps.heading !== '') {
            this.setState({
                latestHeading: nextProps.heading
            });
        }
    }

    render() {
        const style = {
            ...PARENT_XL_STYLE,
            width: this.props.widthPx + 'px',
            right: '-' + (this.props.widthPx + MARGIN_FROM_PARENT_PX) + 'px',
            opacity: this.props.isOpen ? 1 : 0
        };
        if (this.state.size === 'not-xl') {
            style['right'] = '25%';
        }
        return [
            <div class="popover bs-popover-right" style={style}>
                {this.state.size === 'xl' && <div class="arrow" style="top: 40px;"/>}
                <div class="popover-header">
                    {this.state.latestHeading}
                </div>
                {this.props.children}
            </div>,
            <div style={this.props.isOpen ? FADE_STYLE : {}}>
            </div>
        ];
    }
}
