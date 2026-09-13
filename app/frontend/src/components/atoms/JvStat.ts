import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: a single statistic tile (big number + label). Replaces the ad-hoc
// .stat blocks duplicated across dashboard, project list and project detail.
@customElement("jv-stat")
export class JvStat extends LitElement {
  static styles = css`
    :host {
      display: block;
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      padding: var(--jv-md) var(--jv-lg);
      min-width: 140px;
      box-shadow: var(--jv-shadow-sm);
      font-family: var(--jv-font-sans);
    }
    .num {
      font-size: 1.5rem;
      font-weight: 700;
      color: var(--jv-text);
      line-height: 1.2;
    }
    .label {
      color: var(--jv-text-muted);
      font-size: 0.78rem;
      margin-top: var(--jv-xs);
    }
    :host([tone="danger"]) .num {
      color: var(--jv-danger);
    }
    :host([tone="warn"]) .num {
      color: var(--jv-warn);
    }
    :host([tone="primary"]) .num {
      color: var(--jv-primary);
    }
  `;

  @property({ reflect: true }) tone: "" | "danger" | "warn" | "primary" = "";
  @property() value: string | number = "";

  render() {
    return html`<div class="num">${this.value}</div>
      <div class="label"><slot></slot></div>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-stat": JvStat;
  }
}
