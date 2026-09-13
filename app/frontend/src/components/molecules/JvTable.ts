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
    .wrap {
      overflow-x: auto;
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      background: var(--jv-surface);
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.88rem;
    }
    th,
    td {
      text-align: left;
      padding: var(--jv-md) var(--jv-lg);
      border-bottom: 1px solid var(--jv-border);
    }
    th {
      color: var(--jv-text-muted);
      font-weight: 600;
      font-size: 0.72rem;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      background: var(--jv-surface-alt);
      position: sticky;
      top: 0;
      z-index: 1;
    }
    tbody tr {
      transition: background 0.1s ease;
    }
    tbody tr:last-child td {
      border-bottom: none;
    }
    tbody tr:hover {
      background: var(--jv-surface-alt);
    }
    .empty {
      color: var(--jv-text-muted);
      padding: var(--jv-2xl) 0;
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
    return html`<div class="wrap"><table>
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
    </table></div>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-table": JvTable<unknown>;
  }
}
