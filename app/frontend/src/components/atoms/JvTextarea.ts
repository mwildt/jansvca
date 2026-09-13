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
      font-weight: 500;
      color: var(--jv-text-muted);
      margin-bottom: 6px;
    }
    textarea {
      width: 100%;
      min-height: 96px;
      padding: 11px 13px;
      border-radius: var(--jv-radius-sm);
      border: 1px solid var(--jv-border);
      background: var(--jv-surface-alt);
      color: var(--jv-text);
      font-size: 0.92rem;
      font-family: var(--jv-font-mono);
      resize: vertical;
      box-sizing: border-box;
      transition: border-color 0.14s ease, box-shadow 0.14s ease;
    }
    textarea::placeholder {
      color: #5b6477;
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
