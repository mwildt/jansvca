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
      font-size: 0.78rem;
      font-weight: 500;
      color: var(--jv-text-muted);
      margin-bottom: var(--jv-xs);
      letter-spacing: 0.01em;
    }
    textarea {
      width: 100%;
      min-height: 96px;
      padding: 9px var(--jv-md);
      border-radius: var(--jv-sm);
      border: 1px solid var(--jv-border);
      background: var(--jv-surface-2);
      color: var(--jv-text);
      font-size: 0.88rem;
      font-family: var(--jv-font-mono);
      resize: vertical;
      box-sizing: border-box;
      transition: border-color 0.14s ease, box-shadow 0.14s ease;
    }
    textarea::placeholder {
      color: #475569;
    }
    textarea:hover {
      border-color: var(--jv-border-strong);
    }
    textarea:focus {
      outline: none;
      border-color: var(--jv-primary);
      box-shadow: var(--jv-ring);
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
