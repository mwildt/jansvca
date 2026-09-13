import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { VulnerabilityView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvEmpty";
import "../molecules/JvConfirmDialog";
import "../atoms/JvButton";
import "../atoms/JvInput";
import "../atoms/JvBadge";
import { cvssTone } from "../atoms/JvBadge";

// Organism: list of vulnerabilities plus create form.
@customElement("jv-vuln-list")
export class JvVulnList extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: var(--jv-xl);
    }
    .card {
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      padding: var(--jv-lg) var(--jv-xl);
      margin-bottom: var(--jv-md);
    }
    .card a {
      font-weight: 600;
      font-size: 1.05rem;
    }
    .row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: var(--jv-md);
    }
    .meta {
      display: flex;
      gap: var(--jv-lg);
      color: var(--jv-text-muted);
      font-size: 0.85rem;
      margin-top: var(--jv-xs);
      flex-wrap: wrap;
    }
    .form {
      display: grid;
      grid-template-columns: 1fr 1fr 2fr 100px auto;
      gap: var(--jv-md);
      align-items: end;
      background: var(--jv-surface-alt);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      padding: var(--jv-lg) var(--jv-xl);
      margin-bottom: var(--jv-xl);
    }
    @media (max-width: 860px) {
      .form {
        grid-template-columns: 1fr;
      }
    }
  `;

  @state() private vulns: VulnerabilityView[] = [];
  @state() private loading = true;
  @state() private form = { id: "", identifier: "", title: "", cvss: "0" };
  @state() private submitting = false;
  @state() private deleteId: string | null = null;

  connectedCallback(): void {
    super.connectedCallback();
    this.#load();
  }

  async #load(): Promise<void> {
    this.loading = true;
    try {
      this.vulns = await api.listVulnerabilities();
    } catch (e) {
      toast((e as Error).message, "error");
    } finally {
      this.loading = false;
    }
  }

  async #submit(): Promise<void> {
    if (!this.form.id || !this.form.identifier || !this.form.title) {
      toast("ID, Identifier und Titel erforderlich", "error");
      return;
    }
    this.submitting = true;
    try {
      await api.createVulnerability({
        id: this.form.id,
        identifier: this.form.identifier,
        title: this.form.title,
        cvss: Number(this.form.cvss) || 0,
      });
      this.form = { id: "", identifier: "", title: "", cvss: "0" };
      await this.#load();
      toast("Schwachstelle angelegt", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    } finally {
      this.submitting = false;
    }
  }

  async #confirmDelete(): Promise<void> {
    if (!this.deleteId) return;
    const id = this.deleteId;
    this.deleteId = null;
    try {
      await api.deleteVulnerability(id);
      await this.#load();
      toast("Schwachstelle gelöscht", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  render() {
    return html`<div class="header">
        <h2>Schwachstellen</h2>
        <jv-button variant="primary" @click=${() => this.#submit()} ?disabled=${this.submitting}
          >Anlegen</jv-button
        >
      </div>

      <div class="form">
        <jv-input
          label="ID"
          placeholder="z.B. vuln-1"
          .value=${this.form.id}
          @change=${(e: CustomEvent<{ value: string }>) => (this.form.id = e.detail.value)}
        ></jv-input>
        <jv-input
          label="Identifier"
          placeholder="CVE-2024-1234"
          .value=${this.form.identifier}
          @change=${(e: CustomEvent<{ value: string }>) => (this.form.identifier = e.detail.value)}
        ></jv-input>
        <jv-input
          label="Titel"
          .value=${this.form.title}
          @change=${(e: CustomEvent<{ value: string }>) => (this.form.title = e.detail.value)}
        ></jv-input>
        <jv-input
          label="CVSS"
          type="number"
          .value=${this.form.cvss}
          @change=${(e: CustomEvent<{ value: string }>) => (this.form.cvss = e.detail.value)}
        ></jv-input>
        <jv-button variant="primary" @click=${() => this.#submit()} ?disabled=${this.submitting}
          >Anlegen</jv-button
        >
      </div>

      ${this.loading
        ? html`<p>Lädt…</p>`
        : this.vulns.length === 0
          ? html`<jv-empty
              heading="Keine Schwachstellen"
              message="Lege eine Schwachstelle über das Formular an."
            ></jv-empty>`
          : this.vulns.map(
              (v) => html`
                <div class="card">
                  <div class="row">
                    <a href=${`/vulnerabilities/${encodeURIComponent(v.id)}`} data-link>${v.title}</a>
                    <jv-badge tone=${cvssTone(v.cvss)}>${v.cvss.toFixed(1)}</jv-badge>
                  </div>
                  <div class="meta">
                    <span>${v.id}</span>
                    <span>${v.identifier}</span>
                    <span>${v.affected.length} Affected-Ranges</span>
                  </div>
                  <div style="margin-top:8px; display:flex; gap:8px;">
                    <jv-button @click=${() => navigate(`/vulnerabilities/${encodeURIComponent(v.id)}`)}>Öffnen</jv-button>
                    <jv-button variant="danger" @click=${() => (this.deleteId = v.id)}>Löschen</jv-button>
                  </div>
                </div>
              `,
            )}

      <jv-confirm-dialog
        .open=${this.deleteId !== null}
        heading="Schwachstelle löschen?"
        message="Die Schwachstelle wird soft-deleted."
        @confirm=${() => this.#confirmDelete()}
        @cancel=${() => (this.deleteId = null)}
      ></jv-confirm-dialog>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-vuln-list": JvVulnList;
  }
}
