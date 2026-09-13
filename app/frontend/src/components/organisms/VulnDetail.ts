import { LitElement, html, css } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { VulnerabilityView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvConfirmDialog";
import "../atoms/JvButton";
import "../atoms/JvInput";
import "../atoms/JvBadge";
import { cvssTone } from "../atoms/JvBadge";

// Organism: vulnerability detail with affected ranges.
@customElement("jv-vuln-detail")
export class JvVulnDetail extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .top {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: var(--jv-lg);
      margin-bottom: var(--jv-xl);
    }
    .id {
      color: var(--jv-text-muted);
      font-size: 0.9rem;
    }
    .section {
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      padding: var(--jv-lg) var(--jv-xl);
      margin-bottom: var(--jv-xl);
    }
    h3 {
      margin: 0 0 var(--jv-md);
    }
    .desc {
      color: var(--jv-text-muted);
      margin-bottom: var(--jv-md);
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.92rem;
    }
    th, td {
      text-align: left;
      padding: var(--jv-sm) var(--jv-md);
      border-bottom: 1px solid var(--jv-border);
    }
    th {
      color: var(--jv-text-muted);
      font-weight: 600;
      font-size: 0.8rem;
      text-transform: uppercase;
      letter-spacing: 0.04em;
    }
    .row-actions {
      display: flex;
      gap: var(--jv-xs);
    }
    .form {
      display: grid;
      grid-template-columns: 2fr 2fr auto;
      gap: var(--jv-md);
      align-items: end;
    }
    @media (max-width: 760px) {
      .form {
        grid-template-columns: 1fr;
      }
    }
  `;

  @property() vulnId = "";

  @state() private vuln: VulnerabilityView | null = null;
  @state() private loading = true;
  @state() private form = { component: "", version_range: "" };
  @state() private deleteOpen = false;
  @state() private removeComponent: string | null = null;

  connectedCallback(): void {
    super.connectedCallback();
    this.#load();
  }

  willUpdate(changed: Map<string, unknown>): void {
    if (changed.has("vulnId") && this.vulnId) this.#load();
  }

  async #load(): Promise<void> {
    if (!this.vulnId) return;
    this.loading = true;
    try {
      const all = await api.listVulnerabilities();
      this.vuln = all.find((v) => v.id === this.vulnId) ?? null;
    } catch (e) {
      toast((e as Error).message, "error");
    } finally {
      this.loading = false;
    }
  }

  async #addRange(): Promise<void> {
    if (!this.form.component || !this.form.version_range) {
      toast("Komponente und Version-Range erforderlich", "error");
      return;
    }
    try {
      await api.addAffectedRange(this.vulnId, {
        component: this.form.component,
        version_range: this.form.version_range,
      });
      this.form = { component: "", version_range: "" };
      await this.#load();
      toast("Affected-Range hinzugefügt", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  async #confirmRemove(): Promise<void> {
    if (!this.removeComponent) return;
    const c = this.removeComponent;
    this.removeComponent = null;
    try {
      await api.removeAffectedRange(this.vulnId, c);
      await this.#load();
      toast("Affected-Range entfernt", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  async #confirmDelete(): Promise<void> {
    this.deleteOpen = false;
    try {
      await api.deleteVulnerability(this.vulnId);
      toast("Schwachstelle gelöscht", "success");
      navigate("/vulnerabilities");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  render() {
    if (this.loading) return html`<p>Lädt…</p>`;
    if (!this.vuln) return html`<p>Schwachstelle nicht gefunden.</p>`;
    const v = this.vuln;
    return html`
      <div class="top">
        <div>
          <a href="/vulnerabilities" data-link>← Schwachstellen</a>
          <h2 style="margin:4px 0;">${v.title}</h2>
          <div class="id">${v.id} · ${v.identifier}</div>
        </div>
        <div style="display:flex; gap:8px; align-items:center;">
          <jv-badge tone=${cvssTone(v.cvss)}>CVSS ${v.cvss.toFixed(1)}</jv-badge>
          <jv-button variant="danger" @click=${() => (this.deleteOpen = true)}>Löschen</jv-button>
        </div>
      </div>

      <div class="section">
        <h3>Beschreibung</h3>
        <p class="desc">${v.description || "Keine Beschreibung"}</p>
      </div>

      <div class="section">
        <h3>Affected-Ranges (${v.affected.length})</h3>
        <div class="form">
          <jv-input
            label="Komponente"
            placeholder="z.B. pkg:npm/lit"
            .value=${this.form.component}
            @change=${(e: CustomEvent<{ value: string }>) => (this.form.component = e.detail.value)}
          ></jv-input>
          <jv-input
            label="Version-Range"
            placeholder="z.B. >=3.0.0 <3.2.2"
            .value=${this.form.version_range}
            @change=${(e: CustomEvent<{ value: string }>) => (this.form.version_range = e.detail.value)}
          ></jv-input>
          <jv-button variant="primary" @click=${() => this.#addRange()}>Hinzufügen</jv-button>
        </div>
        ${v.affected.length === 0
          ? html`<p style="color:var(--jv-text-muted);">Keine Affected-Ranges.</p>`
          : html`<table>
              <thead>
                <tr><th>Komponente</th><th>Version-Range</th><th></th></tr>
              </thead>
              <tbody>
                ${v.affected.map(
                  (a) => html`<tr>
                    <td>${a.component}</td>
                    <td><code>${a.version_range}</code></td>
                    <td>
                      <div class="row-actions">
                        <jv-button variant="danger" @click=${() => (this.removeComponent = a.component)}>Entfernen</jv-button>
                      </div>
                    </td>
                  </tr>`,
                )}
              </tbody>
            </table>`}
      </div>

      <jv-confirm-dialog
        .open=${this.deleteOpen}
        heading="Schwachstelle löschen?"
        message="Die Schwachstelle wird soft-deleted."
        @confirm=${() => this.#confirmDelete()}
        @cancel=${() => (this.deleteOpen = false)}
      ></jv-confirm-dialog>
      <jv-confirm-dialog
        .open=${this.removeComponent !== null}
        heading="Affected-Range entfernen?"
        message=${`Range für ${this.removeComponent ?? ""} wird entfernt.`}
        @confirm=${() => this.#confirmRemove()}
        @cancel=${() => (this.removeComponent = null)}
      ></jv-confirm-dialog>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-vuln-detail": JvVulnDetail;
  }
}
