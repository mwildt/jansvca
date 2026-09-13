import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Molecule: empty-state placeholder with optional action slot.
@customElement("jv-empty")
export class JvEmpty extends LitElement {
  static styles = css`
    :host {
      display: block;
      text-align: center;
      padding: var(--jv-2xl);
      color: var(--jv-text-muted);
      border: 1px dashed var(--jv-border-strong);
      border-radius: var(--jv-lg);
      background: var(--jv-surface-2);
    }
    .title {
      font-weight: 700;
      color: var(--jv-text);
      margin-bottom: var(--jv-sm);
      font-size: 1.05rem;
      letter-spacing: -0.01em;
    }
    .actions {
      margin-top: var(--jv-xl);
      display: flex;
      gap: var(--jv-sm);
      justify-content: center;
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
