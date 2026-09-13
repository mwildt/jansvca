import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: a single statistic tile (big number + label). Replaces the ad-hoc
// .stat blocks duplicated across dashboard, project list and project detail.
@customElement("jv-stat")
export class JvStat extends LitElement {
  static styles = css`
    :host {
      display: block;
      position: relative;
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      padding: var(--jv-lg) var(--jv-lg) var(--jv-md);
      min-width: 150px;
      box-shadow: var(--jv-shadow-sm);
      font-family: var(--jv-font-sans);
      overflow: hidden;
    }
    :host::before {
      content: "";
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 3px;
      background: var(--jv-grad-brand);
      opacity: 0.7;
    }
    .num {
      font-size: 1.75rem;
      font-weight: 700;
      color: var(--jv-text);
      line-height: 1.1;
      letter-spacing: -0.02em;
      font-variant-numeric: tabular-nums;
    }
    .label {
      color: var(--jv-text-muted);
      font-size: 0.78rem;
      margin-top: var(--jv-xs);
      letter-spacing: 0.01em;
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
    :host([tone="success"]) .num {
      color: var(--jv-success);
    }
  `;

  @property({ reflect: true }) tone: "" | "danger" | "warn" | "primary" | "success" = "";
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
