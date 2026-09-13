import { LitElement, html, css } from "lit";
import { customElement, property } from "lit/decorators.js";

// Atom: a small status pill. Tone encodes severity.
@customElement("jv-badge")
export class JvBadge extends LitElement {
  static styles = css`
    :host {
      display: inline-flex;
      align-items: center;
    }
    span {
      display: inline-flex;
      align-items: center;
      padding: 3px var(--jv-sm);
      border-radius: 999px;
      font-size: 0.74rem;
      font-weight: 600;
      line-height: 1.5;
      background: var(--jv-surface-alt);
      color: var(--jv-text-muted);
      border: 1px solid var(--jv-border);
      font-variant-numeric: tabular-nums;
      white-space: nowrap;
    }
    :host([tone="success"]) span {
      background: var(--jv-success-soft);
      color: #4ade80;
      border-color: rgba(34, 197, 94, 0.3);
    }
    :host([tone="warn"]) span {
      background: var(--jv-warn-soft);
      color: #fbbf24;
      border-color: rgba(245, 158, 11, 0.3);
    }
    :host([tone="danger"]) span {
      background: var(--jv-danger-soft);
      color: #f87171;
      border-color: rgba(239, 68, 68, 0.3);
    }
    :host([tone="info"]) span {
      background: var(--jv-primary-soft);
      color: #a5b4fc;
      border-color: rgba(99, 102, 241, 0.3);
    }
  `;

  @property({ reflect: true }) tone: "" | "success" | "warn" | "danger" | "info" = "";

  render() {
    return html`<span><slot></slot></span>`;
  }
}

// cvssTone maps a CVSS score to a badge tone.
export function cvssTone(cvss: number): "" | "success" | "warn" | "danger" | "info" {
  if (cvss >= 9) return "danger";
  if (cvss >= 7) return "warn";
  if (cvss >= 4) return "info";
  if (cvss > 0) return "success";
  return "";
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-badge": JvBadge;
  }
}
