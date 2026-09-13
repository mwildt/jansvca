import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: labelled text input.
@customElement("jv-input")
export class JvInput extends LitElement {
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
    input {
      width: 100%;
      padding: var(--jv-sm) var(--jv-md);
      border-radius: var(--jv-sm);
      border: 1px solid var(--jv-border);
      background: var(--jv-surface);
      color: var(--jv-text);
      font-size: 0.95rem;
      font-family: inherit;
    }
    input:focus {
      outline: none;
      border-color: var(--jv-primary);
      box-shadow: 0 0 0 2px rgba(29, 78, 216, 0.15);
    }
  `;

  @property() label = "";
  @property() value = "";
  @property() name = "";
  @property() placeholder = "";
  @property() type: "text" | "number" | "password" = "text";

  #onInput(e: InputEvent) {
    const input = e.target as HTMLInputElement;
    this.value = input.value;
    this.dispatchEvent(
      new CustomEvent("change", { detail: { value: this.value, name: this.name } }),
    );
  }

  render() {
    return html`<label>
      ${this.label ? html`<span>${this.label}</span>` : null}
      <input
        .value=${this.value}
        name=${this.name}
        placeholder=${this.placeholder}
        type=${this.type}
        @input=${this.#onInput}
      />
    </label>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-input": JvInput;
  }
}
