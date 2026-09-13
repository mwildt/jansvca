import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: labelled textarea.
@customElement("jv-textarea")
export class JvTextarea extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    label {
      display: block;
      font-size: 0.8rem;
      color: var(--jv-text-muted);
      margin-bottom: var(--jv-xs);
    }
    textarea {
      width: 100%;
      min-height: 72px;
      padding: var(--jv-sm) var(--jv-md);
      border-radius: var(--jv-sm);
      border: 1px solid var(--jv-border);
      background: var(--jv-surface);
      color: var(--jv-text);
      font-size: 0.95rem;
      font-family: inherit;
      resize: vertical;
    }
    textarea:focus {
      outline: none;
      border-color: var(--jv-primary);
      box-shadow: 0 0 0 2px rgba(29, 78, 216, 0.15);
    }
  `;

  @property() label = "";
  @property() value = "";
  @property() name = "";
  @property() placeholder = "";

  #onInput(e: InputEvent) {
    const input = e.target as HTMLTextAreaElement;
    this.value = input.value;
    this.dispatchEvent(
      new CustomEvent("change", { detail: { value: this.value, name: this.name } }),
    );
  }

  render() {
    return html`<label>
      ${this.label ? html`<span>${this.label}</span>` : null}
      <textarea
        .value=${this.value}
        name=${this.name}
        placeholder=${this.placeholder}
        @input=${this.#onInput}
      ></textarea>
    </label>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-textarea": JvTextarea;
  }
}
