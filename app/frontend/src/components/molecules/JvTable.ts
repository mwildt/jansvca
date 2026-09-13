import { LitElement, html, css, type TemplateResult } from "lit";
import { customElement, property } from "lit/decorators.js";

// Molecule: a themed data table. Columns are declared via the `columns`
// property (header labels). Rows are rendered through the `rows` property
// plus a `renderCell` function returning a TemplateResult for each cell,
// keeping the table styling in one place while leaving cell content flexible.
export interface JvTableColumn {
  label: string;
}

@customElement("jv-table")
export class JvTable<T = unknown> extends LitElement {
  static styles = css`
    :host {
      display: block;
      font-family: var(--jv-font-sans);
      color: var(--jv-text);
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.88rem;
    }
    th,
    td {
      text-align: left;
      padding: var(--jv-sm) var(--jv-md);
      border-bottom: 1px solid var(--jv-border);
    }
    th {
      color: var(--jv-text-muted);
      font-weight: 500;
      font-size: 0.74rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }
    tbody tr:last-child td {
      border-bottom: none;
    }
    tbody tr:hover {
      background: var(--jv-surface-alt);
    }
    .empty {
      color: var(--jv-text-muted);
      padding: var(--jv-lg) 0;
      text-align: center;
    }
  `;

  @property({ attribute: false }) columns: JvTableColumn[] = [];
  @property({ attribute: false }) rows: T[] = [];
  @property({ attribute: false }) renderCell: ((row: T, colIndex: number) => TemplateResult) | null = null;
  @property() emptyMessage = "";

  render() {
    if (this.rows.length === 0 && this.emptyMessage) {
      return html`<div class="empty">${this.emptyMessage}</div>`;
    }
    return html`<table>
      <thead>
        <tr>
          ${this.columns.map((c) => html`<th>${c.label}</th>`)}
        </tr>
      </thead>
      <tbody>
        ${this.rows.map(
          (row) => html`<tr>
            ${this.columns.map((_, i) =>
              this.renderCell ? this.renderCell(row, i) : html`<td></td>`,
            )}
          </tr>`,
        )}
      </tbody>
    </table>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-table": JvTable<unknown>;
  }
}
