import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Molecule: a titled content section (a jv-card with a header row). The
// heading slot allows arbitrary content (title + count badge); the default
// slot holds the section body.
@customElement("jv-section")
export class JvSection extends LitElement {
  static styles = css`
    :host {
      display: block;
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-lg);
      padding: var(--jv-xl) var(--jv-xl) var(--jv-lg);
      margin-bottom: var(--jv-xl);
      box-shadow: var(--jv-shadow-sm);
      font-family: var(--jv-font-sans);
      color: var(--jv-text);
    }
    .head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: var(--jv-lg);
      padding-bottom: var(--jv-lg);
      border-bottom: 1px solid var(--jv-border);
      gap: var(--jv-md);
    }
    .title {
      display: flex;
      align-items: center;
      gap: var(--jv-sm);
    }
    h2 {
      margin: 0;
      font-size: 1.1rem;
      letter-spacing: -0.01em;
      font-weight: 600;
    }
    .count {
      color: var(--jv-text-muted);
      font-weight: 600;
      font-size: 0.8rem;
      background: var(--jv-surface-alt);
      border: 1px solid var(--jv-border);
      padding: 1px var(--jv-sm);
      border-radius: 999px;
      font-variant-numeric: tabular-nums;
    }
    .actions {
      display: flex;
      gap: var(--jv-sm);
      align-items: center;
      flex-shrink: 0;
    }
    .body {
      display: block;
    }
  `;

  @property() heading = "";
  @property({ type: Number }) count: number | null = null;

  render() {
    return html`
      <div class="head">
        <div class="title">
          ${this.heading ? html`<h2>${this.heading}</h2>` : null}
          ${this.count !== null ? html`<span class="count">${this.count}</span>` : null}
          <slot name="title"></slot>
        </div>
        <div class="actions"><slot name="actions"></slot></div>
      </div>
      <div class="body"><slot></slot></div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-section": JvSection;
  }
}
