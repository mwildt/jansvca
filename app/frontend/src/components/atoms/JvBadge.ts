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
      padding: 2px var(--jv-sm);
      border-radius: 999px;
      font-size: 0.75rem;
      font-weight: 600;
      line-height: 1.4;
      background: var(--jv-surface-alt);
      color: var(--jv-text-muted);
      border: 1px solid var(--jv-border);
    }
    :host([tone="success"]) span {
      background: #dcfce7;
      color: #166534;
      border-color: #bbf7d0;
    }
    :host([tone="warn"]) span {
      background: #fef3c7;
      color: #92400e;
      border-color: #fde68a;
    }
    :host([tone="danger"]) span {
      background: #fee2e2;
      color: #991b1b;
      border-color: #fecaca;
    }
    :host([tone="info"]) span {
      background: #dbeafe;
      color: #1e40af;
      border-color: #bfdbfe;
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
