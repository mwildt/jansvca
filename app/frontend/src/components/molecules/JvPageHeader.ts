import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Molecule: a page-level header with title, subtitle and an actions slot,
// standardizing the heading layout duplicated across organisms.
@customElement("jv-page-header")
export class JvPageHeader extends LitElement {
  static styles = css`
    :host {
      display: flex;
      align-items: flex-end;
      justify-content: space-between;
      gap: var(--jv-lg);
      margin-bottom: var(--jv-xl);
      font-family: var(--jv-font-sans);
    }
    h1 {
      margin: 0;
      font-size: 1.6rem;
      letter-spacing: -0.02em;
      color: var(--jv-text);
      font-weight: 700;
    }
    .sub {
      color: var(--jv-text-muted);
      font-size: 0.9rem;
      margin-top: var(--jv-xs);
      max-width: 70ch;
    }
    .actions {
      display: flex;
      gap: var(--jv-sm);
      align-items: center;
      flex-shrink: 0;
    }
  `;

  @property() heading = "";
  @property() subtitle = "";

  render() {
    return html`
      <div>
        ${this.heading ? html`<h1>${this.heading}</h1>` : null}
        ${this.subtitle ? html`<div class="sub">${this.subtitle}</div>` : null}
      </div>
      <div class="actions"><slot name="actions"></slot></div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-page-header": JvPageHeader;
  }
}
