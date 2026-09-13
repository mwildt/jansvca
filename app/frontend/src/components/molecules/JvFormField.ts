import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Molecule: a form field wrapping either jv-input or jv-textarea plus optional
// hint/error text. Slotted for flexibility.
@customElement("jv-form-field")
export class JvFormField extends LitElement {
  static styles = css`
    :host {
      display: block;
      margin-bottom: var(--jv-lg);
      font-family: var(--jv-font-sans);
      color: var(--jv-text);
    }
    .label {
      display: block;
      font-size: 0.78rem;
      font-weight: 500;
      color: var(--jv-text-muted);
      margin-bottom: var(--jv-xs);
      letter-spacing: 0.01em;
    }
    .hint {
      font-size: 0.78rem;
      color: var(--jv-text-muted);
      margin-top: var(--jv-xs);
    }
    .error {
      font-size: 0.78rem;
      color: var(--jv-danger);
      margin-top: var(--jv-xs);
      background: rgba(239, 68, 68, 0.1);
      border: 1px solid rgba(239, 68, 68, 0.28);
      padding: var(--jv-xs) var(--jv-sm);
      border-radius: var(--jv-sm);
    }
  `;

  @property() label = "";
  @property() hint = "";
  @property() error = "";

  render() {
    return html`<div>
      ${this.label ? html`<span class="label">${this.label}</span>` : null}
      <slot></slot>
      ${this.hint ? html`<div class="hint">${this.hint}</div>` : null}
      ${this.error ? html`<div class="error">${this.error}</div>` : null}
    </div>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-form-field": JvFormField;
  }
}
