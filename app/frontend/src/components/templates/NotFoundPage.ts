import { LitElement, html, css } from "lit";
import { customElement } from "lit/decorators.js";

@customElement("jv-not-found")
export class JvNotFound extends LitElement {
  static styles = css`
    :host {
      display: block;
      text-align: center;
      padding: var(--jv-xl);
      color: var(--jv-text-muted);
    }
    h2 {
      color: var(--jv-text);
    }
  `;
  render() {
    return html`<h2>404</h2><p>Seite nicht gefunden.</p><p><a href="/" data-link>Zur Übersicht</a></p>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-not-found": JvNotFound;
  }
}
