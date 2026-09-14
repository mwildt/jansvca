import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: a surface container with border, radius and elevation. Centralizes the
// card styling that was duplicated across organisms and the page shell.
@customElement("jv-card")
export class JvCard extends LitElement {
  static styles = css`
    :host {
      display: block;
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-r-lg);
      padding: var(--jv-xl);
      box-shadow: var(--jv-shadow-md);
      color: var(--jv-text);
      font-family: var(--jv-font-sans);
    }
    :host([tone="flat"]) {
      box-shadow: var(--jv-shadow-sm);
      border-radius: var(--jv-r-md);
      padding: var(--jv-lg) var(--jv-xl);
    }
    :host([interactive]) {
      cursor: pointer;
      transition: border-color 0.14s ease, transform 0.1s ease, box-shadow 0.14s ease;
    }
    :host([interactive]:hover) {
      border-color: var(--jv-border-strong);
      transform: translateY(-2px);
      box-shadow: var(--jv-shadow-md);
    }
    :host([interactive]:active) {
      transform: translateY(0);
    }
  `;

  @property({ reflect: true }) tone: "" | "flat" = "";
  @property({ type: Boolean, reflect: true }) interactive = false;

  render() {
    return html`<slot></slot>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-card": JvCard;
  }
}
