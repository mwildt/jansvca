import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { VulnerabilityView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvPageHeader";
import "../molecules/JvEmpty";
import "../molecules/JvConfirmDialog";
import "../molecules/JvButtonRow";
import "../atoms/JvButton";
import "../atoms/JvInput";
import "../atoms/JvBadge";
import "../atoms/JvCode";
import "../atoms/JvSpinner";
import { cvssTone } from "../atoms/JvBadge";

// Organism: list of vulnerabilities plus create form.
@customElement("jv-vuln-list")
export class JvVulnList extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .form {
      display: grid;
      grid-template-columns: 1fr 1fr 2fr 100px auto;
      gap: var(--jv-md);
      align-items: end;
      background: var(--jv-surface-alt);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-r-md);
      padding: var(--jv-lg) var(--jv-xl);
      margin-bottom: var(--jv-xl);
    }
    @media (max-width: 860px) {
      .form {
        grid-template-columns: 1fr;
      }
    }
    .item {
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-r-md);
      padding: var(--jv-lg) var(--jv-xl);
      margin-bottom: var(--jv-md);
      box-shadow: var(--jv-shadow-sm);
      transition: border-color 0.14s ease, box-shadow 0.14s ease;
    }
    .item:hover {
      border-color: var(--jv-border-strong);
      box-shadow: var(--jv-shadow-md);
    }
    .row {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: var(--jv-md);
    }
    .row a {
      font-weight: 600;
      font-size: 1.05rem;
      color: var(--jv-text);
      text-decoration: none;
    }
    .row a:hover {
      color: var(--jv-primary);
    }
    .meta {
      display: flex;
      gap: var(--jv-lg);
      color: var(--jv-text-muted);
      font-size: 0.85rem;
      margin-top: var(--jv-xs);
      flex-wrap: wrap;
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
      toast("Schwachstelle gel\u00f6scht", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  render() {
    return html`
      <jv-page-header heading="Schwachstellen"></jv-page-header>

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
        <jv-button variant="primary" @click=${() => this.#submit()} ?disabled=${this.submitting}>
          Anlegen
        </jv-button>
      </div>

      ${this.loading
        ? html`<jv-spinner></jv-spinner>`
        : this.vulns.length === 0
          ? html`<jv-empty
              heading="Keine Schwachstellen"
              message="Lege eine Schwachstelle \u00fcber das Formular an."
            ></jv-empty>`
          : this.vulns.map(
              (v) => html`
                <div class="item">
                  <div class="row">
                    <a href=${`/vulnerabilities/${encodeURIComponent(v.id)}`} data-link>${v.title}</a>
                    <jv-badge tone=${cvssTone(v.cvss)}>${v.cvss.toFixed(1)}</jv-badge>
                  </div>
                  <div class="meta">
                    <jv-code>${v.id}</jv-code>
                    <span>${v.identifier}</span>
                    <span>${(v.affected ?? []).length} Affected-Ranges</span>
                  </div>
                  <jv-button-row style="margin-top:var(--jv-md);">
                    <jv-button @click=${() => navigate(`/vulnerabilities/${encodeURIComponent(v.id)}`)}>\u00d6ffnen</jv-button>
                    <jv-button variant="danger" @click=${() => (this.deleteId = v.id)}>L\u00f6schen</jv-button>
                  </jv-button-row>
                </div>
              `,
            )}

      <jv-confirm-dialog
        .open=${this.deleteId !== null}
        heading="Schwachstelle l\u00f6schen?"
        message="Die Schwachstelle wird soft-deleted."
        @confirm=${() => this.#confirmDelete()}
        @cancel=${() => (this.deleteId = null)}
      ></jv-confirm-dialog>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-vuln-list": JvVulnList;
  }
}
