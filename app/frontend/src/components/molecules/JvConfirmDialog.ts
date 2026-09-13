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
      background: rgba(15, 23, 42, 0.45);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 100;
    }
    .dialog {
      background: var(--jv-surface);
      border-radius: var(--jv-lg);
      padding: var(--jv-xl);
      max-width: 420px;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.25);
    }
    h3 {
      margin: 0 0 var(--jv-md);
      font-size: 1.1rem;
    }
    p {
      margin: 0 0 var(--jv-xl);
      color: var(--jv-text-muted);
    }
    .actions {
      display: flex;
      justify-content: flex-end;
      gap: var(--jv-sm);
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
