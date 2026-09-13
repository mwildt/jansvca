import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: a button with variants.
@customElement("jv-button")
export class JvButton extends LitElement {
  static styles = css`
    :host {
      display: inline-flex;
    }
    :host([block]) {
      display: flex;
      width: 100%;
    }
    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: var(--jv-sm);
      padding: 11px var(--jv-lg);
      border-radius: var(--jv-sm);
      border: 1px solid var(--jv-border);
      background: var(--jv-surface-alt);
      color: var(--jv-text);
      font-size: 0.9rem;
      font-weight: 600;
      font-family: inherit;
      cursor: pointer;
      transition: background 0.14s ease, border-color 0.14s ease, transform 0.06s ease, box-shadow 0.14s ease;
      white-space: nowrap;
    }
    :host([block]) button {
      width: 100%;
    }
    button:hover {
      background: var(--jv-surface);
      border-color: var(--jv-border-strong);
    }
    button:active {
      transform: translateY(1px);
    }
    :host([variant="primary"]) button {
      background: var(--jv-primary);
      border-color: transparent;
      color: #fff;
      box-shadow: var(--jv-shadow-primary);
    }
    :host([variant="primary"]) button:hover {
      background: var(--jv-primary-hover);
    }
    :host([variant="danger"]) button {
      background: var(--jv-danger-soft);
      border-color: transparent;
      color: var(--jv-danger);
    }
    :host([variant="danger"]) button:hover {
      background: var(--jv-danger);
      color: #fff;
    }
    :host([variant="ghost"]) button {
      background: transparent;
      border-color: transparent;
      color: var(--jv-text-muted);
    }
    :host([variant="ghost"]) button:hover {
      color: var(--jv-text);
      background: var(--jv-surface-alt);
    }
    button:disabled {
      opacity: 0.5;
      cursor: not-allowed;
      transform: none;
      box-shadow: none;
    }
  `;

  @property({ reflect: true }) variant: "" | "primary" | "danger" | "ghost" = "";
  @property({ type: Boolean }) disabled = false;
  @property({ type: Boolean, reflect: true }) block = false;
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
