import { LitElement, html, css } from "lit";
import { customElement } from "lit/decorators.js";

// Atom: inline monospace code snippet, used in tables and detail views to
// render versions and identifiers consistently.
@customElement("jv-code")
export class JvCode extends LitElement {
  static styles = css`
    :host {
      display: inline-block;
      font-family: var(--jv-font-mono);
      font-size: 0.85rem;
      color: var(--jv-text-muted);
      background: var(--jv-surface-2);
      padding: 1px var(--jv-xs);
      border-radius: var(--jv-r-xs);
    }
  `;

  render() {
    return html`<slot></slot>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-code": JvCode;
  }
}
