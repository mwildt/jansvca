import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";
import "../atoms/JvButton";

// Molecule: a modal confirmation dialog. Emits "confirm" and "cancel".
@customElement("jv-confirm-dialog")
export class JvConfirmDialog extends LitElement {
  static styles = css`
    :host {
      display: contents;
    }
    .overlay {
      position: fixed;
      inset: 0;
      background: rgba(2, 6, 23, 0.7);
      backdrop-filter: blur(6px);
      -webkit-backdrop-filter: blur(6px);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 100;
      animation: jv-fade 0.14s ease;
    }
    .dialog {
      background: var(--jv-surface);
      border: 1px solid var(--jv-border-strong);
      border-radius: var(--jv-lg);
      padding: var(--jv-xl);
      max-width: 420px;
      width: calc(100% - 48px);
      box-shadow: var(--jv-shadow-lg);
      animation: jv-pop 0.16s ease;
    }
    h3 {
      margin: 0 0 var(--jv-md);
      font-size: 1.15rem;
      font-weight: 700;
      letter-spacing: -0.01em;
    }
    p {
      margin: 0 0 var(--jv-xl);
      color: var(--jv-text-muted);
      line-height: 1.5;
    }
    .actions {
      display: flex;
      justify-content: flex-end;
      gap: var(--jv-sm);
    }
    @keyframes jv-fade {
      from { opacity: 0; }
    }
    @keyframes jv-pop {
      from { opacity: 0; transform: translateY(8px) scale(0.98); }
    }
  `;

  @property({ type: Boolean }) open = false;
  @property() heading = "Bestätigen";
  @property() message = "";

  render() {
    if (!this.open) return null;
    return html`<div class="overlay" @click=${(e: MouseEvent) => {
      if (e.target === e.currentTarget) this.#cancel();
    }}>
      <div class="dialog">
        <h3>${this.heading}</h3>
        <p>${this.message}</p>
        <div class="actions">
          <jv-button @click=${this.#cancel}>Abbrechen</jv-button>
          <jv-button variant="danger" @click=${this.#confirm}>Bestätigen</jv-button>
        </div>
      </div>
    </div>`;
  }

  #cancel() {
    this.open = false;
    this.dispatchEvent(new CustomEvent("cancel"));
  }

  #confirm() {
    this.open = false;
    this.dispatchEvent(new CustomEvent("confirm"));
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-confirm-dialog": JvConfirmDialog;
  }
}
