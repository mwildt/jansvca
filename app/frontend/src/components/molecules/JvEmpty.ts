import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Molecule: empty-state placeholder with optional action slot.
@customElement("jv-empty")
export class JvEmpty extends LitElement {
  static styles = css`
    :host {
      display: block;
      text-align: center;
      padding: var(--jv-xl);
      color: var(--jv-text-muted);
      border: 1px dashed var(--jv-border);
      border-radius: var(--jv-md);
      background: var(--jv-surface-alt);
    }
    .title {
      font-weight: 600;
      color: var(--jv-text);
      margin-bottom: var(--jv-xs);
    }
    .actions {
      margin-top: var(--jv-lg);
    }
  `;

  @property() heading = "";
  @property() message = "";

  render() {
    return html`<div class="title">${this.heading}</div>
      <div>${this.message}</div>
      <div class="actions"><slot></slot></div>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-empty": JvEmpty;
  }
}
