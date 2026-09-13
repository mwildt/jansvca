import { LitElement, html, css } from "lit";
import { customElement } from "lit/decorators.js";

// Molecule: a horizontal row of actions (buttons), with consistent gap and
// alignment. Slot buttons directly.
@customElement("jv-button-row")
export class JvButtonRow extends LitElement {
  static styles = css`
    :host {
      display: flex;
      gap: var(--jv-sm);
      align-items: center;
      flex-wrap: wrap;
      font-family: var(--jv-font-sans);
    }
    :host([align="end"]) {
      justify-content: flex-end;
    }
    :host([align="center"]) {
      justify-content: center;
    }
    :host([align="between"]) {
      justify-content: space-between;
    }
  `;

  render() {
    return html`<slot></slot>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-button-row": JvButtonRow;
  }
}
