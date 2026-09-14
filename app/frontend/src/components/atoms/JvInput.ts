import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: labelled text input. Matches jv-button height (36px) for aligned rows.
@customElement("jv-input")
export class JvInput extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    label {
      display: block;
      font-size: 0.78rem;
      font-weight: 500;
      color: var(--jv-text-muted);
      margin-bottom: var(--jv-xs);
      letter-spacing: 0.01em;
    }
    input {
      width: 100%;
      height: 36px;
      padding: 0 var(--jv-md);
      border-radius: var(--jv-r-sm);
      border: 1px solid var(--jv-border);
      background: var(--jv-surface-2);
      color: var(--jv-text);
      font-size: 0.875rem;
      font-family: inherit;
      box-sizing: border-box;
      transition: border-color 0.14s ease, box-shadow 0.14s ease;
    }
    input::placeholder {
      color: #475569;
    }
    input:hover {
      border-color: var(--jv-border-strong);
    }
    input:focus {
      outline: none;
      border-color: var(--jv-primary);
      box-shadow: var(--jv-ring);
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
