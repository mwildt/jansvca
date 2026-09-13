import { LitElement, html, css } from "lit";
import { customElement } from "lit/decorators.js";

// Molecule: a responsive grid wrapper for jv-stat tiles. Slots one or more
// <jv-stat> children; columns flow to fill the available width.
@customElement("jv-stat-grid")
export class JvStatGrid extends LitElement {
  static styles = css`
    :host {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
      gap: var(--jv-md);
      margin-bottom: var(--jv-xl);
      font-family: var(--jv-font-sans);
    }
  `;

  render() {
    return html`<slot></slot>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-stat-grid": JvStatGrid;
  }
}
