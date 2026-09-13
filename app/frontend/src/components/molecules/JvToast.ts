import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Molecule: a transient toast notification. Singleton listener stack so any
// component can push a message via `toast(msg)`.
@customElement("jv-toast")
export class JvToast extends LitElement {
  static styles = css`
    :host {
      position: fixed;
      bottom: var(--jv-xl);
      right: var(--jv-xl);
      z-index: 200;
      display: flex;
      flex-direction: column;
      gap: var(--jv-sm);
      pointer-events: none;
    }
    .toast {
      pointer-events: auto;
      background: var(--jv-surface);
      color: var(--jv-text);
      border: 1px solid var(--jv-border-strong);
      padding: var(--jv-md) var(--jv-lg);
      border-radius: var(--jv-md);
      box-shadow: var(--jv-shadow-lg);
      font-size: 0.88rem;
      max-width: 380px;
      border-left: 3px solid var(--jv-primary);
      animation: jv-toast-in 0.18s ease;
    }
    .toast.error {
      border-left-color: var(--jv-danger);
    }
    .toast.success {
      border-left-color: var(--jv-success);
    }
    @keyframes jv-toast-in {
      from { opacity: 0; transform: translateX(12px); }
    }
  `;

  @property({ attribute: false })
  items: { id: number; message: string; tone: "info" | "error" | "success" }[] = [];

  #seq = 0;

  connectedCallback(): void {
    super.connectedCallback();
    instance = this;
  }

  disconnectedCallback(): void {
    super.disconnectedCallback();
    if (instance === this) instance = null;
  }

  push(message: string, tone: "info" | "error" | "success" = "info", ttl = 3500): void {
    const id = ++this.#seq;
    this.items = [...this.items, { id, message, tone }];
    this.requestUpdate();
    window.setTimeout(() => this.#dismiss(id), ttl);
  }

  #dismiss(id: number): void {
    this.items = this.items.filter((i) => i.id !== id);
  }

  render() {
    return html`${this.items.map(
      (i) => html`<div class="toast ${i.tone}">${i.message}</div>`,
    )}`;
  }
}

let instance: JvToast | null = null;

export function toast(message: string, tone: "info" | "error" | "success" = "info"): void {
  instance?.push(message, tone);
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-toast": JvToast;
  }
}
