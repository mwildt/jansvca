import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Molecule: a back-link breadcrumb. Renders an anchor with data-link so the
// client-side router picks it up. Default slot allows custom labels.
@customElement("jv-breadcrumb")
export class JvBreadcrumb extends LitElement {
  static styles = css`
    :host {
      display: block;
      margin-bottom: var(--jv-md);
      font-family: var(--jv-font-sans);
    }
    a {
      color: var(--jv-text-muted);
      text-decoration: none;
      font-size: 0.85rem;
      transition: color 0.12s ease;
    }
    a:hover {
      color: var(--jv-text);
    }
  `;

  @property() href = "";

  render() {
    return html`<a href=${this.href} data-link><slot></slot></a>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-breadcrumb": JvBreadcrumb;
  }
}
