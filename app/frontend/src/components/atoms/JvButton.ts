import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: a button with variants. Consistent height (36px) and radius across all
// variants so rows of buttons line up.
@customElement("jv-button")
export class JvButton extends LitElement {
  static styles = css`
    :host {
      display: inline-flex;
    }
    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: var(--jv-sm);
      height: 36px;
      padding: 0 var(--jv-md);
      border-radius: var(--jv-r-sm);
      border: 1px solid var(--jv-border);
      background: var(--jv-surface-alt);
      color: var(--jv-text);
      font-size: 0.875rem;
      font-weight: 500;
      font-family: inherit;
      cursor: pointer;
      white-space: nowrap;
      transition: background 0.14s ease, border-color 0.14s ease, transform 0.06s ease, box-shadow 0.14s ease;
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
      box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
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
