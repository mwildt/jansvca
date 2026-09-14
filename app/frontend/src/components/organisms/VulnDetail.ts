import { LitElement, html, css, type TemplateResult } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { AffectedRangeView, VulnerabilityView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvBreadcrumb";
import "../molecules/JvConfirmDialog";
import "../molecules/JvSection";
import "../molecules/JvButtonRow";
import "../molecules/JvTable";
import "../atoms/JvButton";
import "../atoms/JvInput";
import "../atoms/JvBadge";
import "../atoms/JvCode";
import "../atoms/JvSpinner";
import { cvssTone } from "../atoms/JvBadge";

// Organism: vulnerability detail with affected ranges.
@customElement("jv-vuln-detail")
export class JvVulnDetail extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .id {
      color: var(--jv-text-muted);
      font-size: 0.9rem;
      margin-top: var(--jv-xs);
    }
    .desc {
      color: var(--jv-text-muted);
      margin-bottom: var(--jv-md);
      line-height: 1.5;
    }
    .form {
      display: grid;
      grid-template-columns: 2fr 2fr auto;
      gap: var(--jv-md);
      align-items: end;
      margin-bottom: var(--jv-lg);
    }
    @media (max-width: 760px) {
      .form {
        grid-template-columns: 1fr;
      }
    }
    .empty {
      color: var(--jv-text-muted);
      padding: var(--jv-lg) 0;
      text-align: center;
    }
  `;

  @property() vulnId = "";

  @state() private vuln: VulnerabilityView | null = null;
  @state() private loading = true;
  @state() private form = { component: "", version_range: "" };
  @state() private deleteOpen = false;
  @state() private removeComponent: string | null = null;

  #rangeColumns = [{ label: "Komponente" }, { label: "Version-Range" }, { label: "" }];

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
      const found = all.find((v) => v.id === this.vulnId) ?? null;
      if (found) found.affected = found.affected ?? [];
      this.vuln = found;
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
      toast("Affected-Range hinzugef\u00fcgt", "success");
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
      toast("Schwachstelle gel\u00f6scht", "success");
      navigate("/vulnerabilities");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  #renderRangeCell = (a: AffectedRangeView, i: number): TemplateResult => {
    if (i === 0) return html`<td>${a.component}</td>`;
    if (i === 1) return html`<td><jv-code>${a.version_range}</jv-code></td>`;
    return html`<td>
      <jv-button-row align="end">
        <jv-button variant="danger" @click=${() => (this.removeComponent = a.component)}>Entfernen</jv-button>
      </jv-button-row>
    </td>`;
  };

  render() {
    if (this.loading) return html`<jv-spinner></jv-spinner>`;
    if (!this.vuln) return html`<p>Schwachstelle nicht gefunden.</p>`;
    const v = this.vuln;
    return html`
      <jv-breadcrumb href="/vulnerabilities">\u2190 Schwachstellen</jv-breadcrumb>

      <div style="display:flex; justify-content:space-between; align-items:baseline; gap:var(--jv-lg); margin-bottom:var(--jv-xl);">
        <div>
          <h2 style="margin:0 0 var(--jv-xs); font-size:1.6rem; letter-spacing:-0.02em;">${v.title}</h2>
          <div class="id">
            <jv-code>${v.id}</jv-code> \u00b7 ${v.identifier}
          </div>
        </div>
        <jv-button-row>
          <jv-badge tone=${cvssTone(v.cvss)}>CVSS ${v.cvss.toFixed(1)}</jv-badge>
          <jv-button variant="danger" @click=${() => (this.deleteOpen = true)}>L\u00f6schen</jv-button>
        </jv-button-row>
      </div>

      <jv-section heading="Beschreibung">
        <p class="desc">${v.description || "Keine Beschreibung"}</p>
      </jv-section>

      <jv-section heading="Affected-Ranges" .count=${v.affected.length}>
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
          <jv-button variant="primary" @click=${() => this.#addRange()}>Hinzuf\u00fcgen</jv-button>
        </div>
        ${v.affected.length === 0
          ? html`<div class="empty">Keine Affected-Ranges.</div>`
          : html`<jv-table
              .columns=${this.#rangeColumns}
              .rows=${v.affected}
              .renderCell=${this.#renderRangeCell}
            ></jv-table>`}
      </jv-section>

      <jv-confirm-dialog
        .open=${this.deleteOpen}
        heading="Schwachstelle l\u00f6schen?"
        message="Die Schwachstelle wird soft-deleted."
        @confirm=${() => this.#confirmDelete()}
        @cancel=${() => (this.deleteOpen = false)}
      ></jv-confirm-dialog>
      <jv-confirm-dialog
        .open=${this.removeComponent !== null}
        heading="Affected-Range entfernen?"
        message=${`Range f\u00fcr ${this.removeComponent ?? ""} wird entfernt.`}
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
