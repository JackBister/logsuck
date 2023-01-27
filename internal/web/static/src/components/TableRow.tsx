import { h, RenderableProps } from "preact";

export interface TableRowProps {
  onClick?: (evt: Event) => void;
}

export const TableRow = (props: RenderableProps<TableRowProps>) => (
  <tr
    tabIndex={props.onClick && 0}
    onClick={props.onClick}
    style={props.onClick && { cursor: "pointer" }}
    role={props.onClick && "button"}
    onKeyDown={
      props.onClick &&
      ((evt: KeyboardEvent) => {
        if (!props.onClick) {
          return;
        }
        if (evt.key === " " || evt.key === "Enter" || evt.key === "Spacebar") {
          props.onClick(evt);
        }
      })
    }
  >
    {props.children}
  </tr>
);
