import { LitElement, html, css } from "lit";
import { customElement } from "lit/decorators.js";

// Atom: a small inline loading indicator, replacing the bare "<p>Lädt…</p>"
// strings scattered across the organisms.
@customElement("jv-spinner")
export class JvSpinner extends LitElement {
  static styles = css`
    :host {
      display: inline-flex;
      align-items: center;
      gap: var(--jv-sm);
      color: var(--jv-text-muted);
      font-size: 0.9rem;
      font-family: var(--jv-font-sans);
    }
    .ring {
      width: 14px;
      height: 14px;
      border-radius: 50%;
      border: 2px solid var(--jv-border-strong);
      border-top-color: var(--jv-primary);
      animation: jv-spin 0.7s linear infinite;
    }
    @keyframes jv-spin {
      to {
        transform: rotate(360deg);
      }
    }
  `;

  render() {
    return html`<span class="ring"></span><slot>Lädt…</slot>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-spinner": JvSpinner;
  }
}
