import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: a button with variants.
@customElement("jv-button")
export class JvButton extends LitElement {
  static styles = css`
    :host {
      display: inline-flex;
    }
    button {
      display: inline-flex;
      align-items: center;
      gap: var(--jv-sm);
      padding: var(--jv-sm) var(--jv-lg);
      border-radius: var(--jv-sm);
      border: 1px solid var(--jv-border);
      background: var(--jv-surface);
      color: var(--jv-text);
      font-size: 0.9rem;
      font-family: inherit;
      cursor: pointer;
      transition: background 0.12s ease, border-color 0.12s ease;
    }
    button:hover {
      background: var(--jv-surface-alt);
    }
    :host([variant="primary"]) button {
      background: var(--jv-primary);
      border-color: var(--jv-primary);
      color: #fff;
    }
    :host([variant="primary"]) button:hover {
      background: var(--jv-primary-hover);
    }
    :host([variant="danger"]) button {
      background: var(--jv-danger);
      border-color: var(--jv-danger);
      color: #fff;
    }
    :host([variant="danger"]) button:hover {
      background: var(--jv-danger-hover);
    }
    button:disabled {
      opacity: 0.55;
      cursor: not-allowed;
    }
  `;

  @property({ reflect: true }) variant: "" | "primary" | "danger" = "";
  @property({ type: Boolean }) disabled = false;
  @property() type: "button" | "submit" = "button";

  render() {
    return html`<button
      type=${this.type}
      ?disabled=${this.disabled}
      @click=${(e: Event) => {
        if (this.disabled) e.preventDefault();
      }}
    >
      <slot></slot>
    </button>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-button": JvButton;
  }
}
