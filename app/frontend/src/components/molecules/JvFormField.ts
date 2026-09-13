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
