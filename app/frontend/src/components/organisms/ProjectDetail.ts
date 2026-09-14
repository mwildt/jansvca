import { LitElement, html, css, nothing, type TemplateResult } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { Match, ProjectView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvBreadcrumb";
import "../molecules/JvConfirmDialog";
import "../molecules/JvSection";
import "../molecules/JvStatGrid";
import "../molecules/JvButtonRow";
import "../atoms/JvButton";
import "../atoms/JvInput";
import "../atoms/JvTextarea";
import "../atoms/JvBadge";
import "../atoms/JvCard";
import "../atoms/JvStat";
import "../atoms/JvCode";
import "../atoms/JvSpinner";
import { cvssTone } from "../atoms/JvBadge";

// Organism: project detail with components, matches, SBOM import and edit.
@customElement("jv-project-detail")
export class JvProjectDetail extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .desc {
      color: var(--jv-text-muted);
      max-width: 70ch;
      margin-bottom: var(--jv-xl);
      line-height: 1.5;
    }
    .filter-row {
      display: flex;
      align-items: end;
      gap: var(--jv-md);
      margin-bottom: var(--jv-lg);
      max-width: 420px;
    }
    .filter-row jv-input {
      flex: 1;
    }
    .add-row {
      display: grid;
      grid-template-columns: 1fr 160px auto;
      gap: var(--jv-md);
      align-items: end;
      margin-bottom: var(--jv-lg);
    }
    @media (max-width: 640px) {
      .add-row {
        grid-template-columns: 1fr;
      }
    }
    .edit-grid {
      display: grid;
      gap: var(--jv-lg);
      max-width: 560px;
    }
    .sbom {
      margin-top: var(--jv-lg);
      padding: var(--jv-lg);
      border: 1px dashed var(--jv-border-strong);
      border-radius: var(--jv-r-md);
      background: var(--jv-surface-2);
      display: grid;
      gap: var(--jv-md);
    }
    .sbom .hint {
      margin: 0;
      color: var(--jv-text-muted);
      font-size: 0.82rem;
    }
    .sbom .file {
      font-size: 0.85rem;
      color: var(--jv-text-muted);
    }
    .ver-edit {
      display: flex;
      align-items: center;
      gap: var(--jv-xs);
    }
    .ver-input {
      padding: 6px var(--jv-sm);
      border-radius: var(--jv-r-sm);
      border: 1px solid var(--jv-border-strong);
      background: var(--jv-surface-2);
      color: var(--jv-text);
      font-family: var(--jv-font-mono);
      font-size: 0.85rem;
      max-width: 9rem;
    }
    .ver-input:focus {
      outline: none;
      border-color: var(--jv-primary);
      box-shadow: var(--jv-ring);
    }
    .empty {
      color: var(--jv-text-muted);
      padding: var(--jv-lg) 0;
      text-align: center;
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
  `;

  @property() projectId = "";

  @state() private project: ProjectView | null = null;
  @state() private matchResults: Match[] = [];
  @state() private loading = true;
  @state() private compForm = { component: "", version: "" };
  @state() private editing = false;
  @state() private edit = { name: "", description: "" };
  @state() private deleteOpen = false;
  @state() private removeComponent: string | null = null;
  @state() private editingComponent: string | null = null;
  @state() private editVersion = "";
  @state() private savingVersion = false;
  @state() private sbomOpen = false;
  @state() private sbomText = "";
  @state() private importing = false;
  @state() private compFilter = "";

  #matchColumns = [
    { label: "Komponente" },
    { label: "Version" },
    { label: "Schwachstelle" },
    { label: "CVSS" },
  ];

  connectedCallback(): void {
    super.connectedCallback();
    this.#load();
  }

  willUpdate(changed: Map<string, unknown>): void {
    if (changed.has("projectId") && this.projectId) {
      this.#load();
    }
  }

  async #load(): Promise<void> {
    if (!this.projectId) return;
    this.loading = true;
    try {
      const [p, m] = await Promise.all([
        api.getProject(this.projectId),
        api.matches(this.projectId).catch(() => [] as Match[]),
      ]);
      this.project = p;
      this.matchResults = m ?? [];
      this.edit = { name: p.name, description: p.description };
    } catch (e) {
      toast((e as Error).message, "error");
    } finally {
      this.loading = false;
    }
  }

  async #saveEdit(): Promise<void> {
    if (!this.project) return;
    try {
      const patch: { name?: string; description?: string } = {};
      if (this.edit.name !== this.project.name) patch.name = this.edit.name;
      if (this.edit.description !== this.project.description) patch.description = this.edit.description;
      if (patch.name || "description" in patch) {
        await api.updateProject(this.projectId, patch);
        await this.#load();
        toast("Projekt aktualisiert", "success");
      }
      this.editing = false;
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  async #addComponent(): Promise<void> {
    if (!this.compForm.component || !this.compForm.version) {
      toast("Komponente und Version erforderlich", "error");
      return;
    }
    try {
      await api.addComponent(this.projectId, {
        component: this.compForm.component,
        version: this.compForm.version,
      });
      this.compForm = { component: "", version: "" };
      await this.#load();
      toast("Komponente hinzugef\u00fcgt", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  async #confirmRemoveComponent(): Promise<void> {
    if (!this.removeComponent) return;
    const c = this.removeComponent;
    this.removeComponent = null;
    try {
      await api.removeComponent(this.projectId, c);
      await this.#load();
      toast("Komponente entfernt", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  #startEditVersion(component: string, version: string): void {
    this.editingComponent = component;
    this.editVersion = version;
  }

  #cancelEditVersion(): void {
    this.editingComponent = null;
    this.editVersion = "";
  }

  async #saveVersion(component: string): Promise<void> {
    if (!this.editVersion.trim()) {
      toast("Version erforderlich", "error");
      return;
    }
    this.savingVersion = true;
    try {
      await api.updateComponent(this.projectId, component, this.editVersion.trim());
      this.editingComponent = null;
      this.editVersion = "";
      await this.#load();
      toast("Version aktualisiert", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    } finally {
      this.savingVersion = false;
    }
  }

  async #pickSbomFile(e: Event): Promise<void> {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const text = await file.text();
    this.sbomText = text;
    input.value = "";
  }

  async #importSbom(): Promise<void> {
    if (!this.sbomText.trim()) {
      toast("SBOM ist leer", "error");
      return;
    }
    this.importing = true;
    try {
      const res = await api.importSbom(this.projectId, this.sbomText);
      await this.#load();
      this.sbomOpen = false;
      this.sbomText = "";
      toast(`${res.imported} Komponente(n) importiert (${res.components} im SBOM)`, "success");
    } catch (e) {
      toast((e as Error).message, "error");
    } finally {
      this.importing = false;
    }
  }

  async #confirmDelete(): Promise<void> {
    this.deleteOpen = false;
    try {
      await api.deleteProject(this.projectId);
      toast("Projekt gel\u00f6scht", "success");
      navigate("/projects");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  #renderMatchCell = (m: Match, i: number): TemplateResult => {
    if (i === 0) return html`<td>${m.component}</td>`;
    if (i === 1) return html`<td><jv-code>${m.version}</jv-code></td>`;
    if (i === 2) return html`<td>${m.vulnerability_identifier}</td>`;
    return html`<td><jv-badge tone=${cvssTone(m.cvss)}>${m.cvss.toFixed(1)}</jv-badge></td>`;
  };

  render() {
    if (this.loading) return html`<jv-spinner></jv-spinner>`;
    if (!this.project) return html`<p>Projekt nicht gefunden.</p>`;
    const p = this.project;
    const comps = p.components ?? [];
    const f = this.compFilter.trim().toLowerCase();
    const filteredComps = f
      ? comps.filter((c) => c.component.toLowerCase().includes(f) || c.version.toLowerCase().includes(f))
      : comps;
    const matches = this.matchResults ?? [];
    const highCount = matches.filter((m) => m.cvss >= 7).length;
    const maxCvss = matches.reduce((mx, m) => Math.max(mx, m.cvss), 0);

    return html`
      <jv-breadcrumb href="/projects">\u2190 Projekte</jv-breadcrumb>

      <div class="top-actions" style="display:flex; justify-content:space-between; align-items:center; gap:var(--jv-lg); margin-bottom:var(--jv-xl);">
        <div>
          <h1 style="margin:0; font-size:1.6rem; letter-spacing:-0.02em;">${this.editing ? this.edit.name : p.name}</h1>
          <jv-code>${p.id}</jv-code>
        </div>
        <jv-button-row>
          ${this.editing
            ? html`<jv-button variant="primary" @click=${() => this.#saveEdit()}>Speichern</jv-button>
                <jv-button @click=${() => {
                  this.editing = false;
                  this.edit = { name: p.name, description: p.description };
                }}>Abbrechen</jv-button>`
            : html`<jv-button @click=${() => (this.editing = true)}>Bearbeiten</jv-button>
                <jv-button variant="danger" @click=${() => (this.deleteOpen = true)}>L\u00f6schen</jv-button>`}
        </jv-button-row>
      </div>

      ${this.editing
        ? html`<jv-section heading="Projektdaten">
            <div class="edit-grid">
              <jv-input
                label="Name"
                .value=${this.edit.name}
                @change=${(e: CustomEvent<{ value: string }>) => (this.edit.name = e.detail.value)}
              ></jv-input>
              <jv-textarea
                label="Beschreibung"
                .value=${this.edit.description}
                @change=${(e: CustomEvent<{ value: string }>) => (this.edit.description = e.detail.value)}
              ></jv-textarea>
            </div>
          </jv-section>`
        : html`<p class="desc">${p.description || "Keine Beschreibung"}</p>`}

      <jv-stat-grid>
        <jv-stat value=${comps.length}>Komponenten</jv-stat>
        <jv-stat value=${matches.length}>Matches</jv-stat>
        <jv-stat tone=${highCount > 0 ? "danger" : ""} value=${highCount}>Kritisch/Hoch</jv-stat>
        <jv-stat tone=${maxCvss >= 7 ? "warn" : ""} value=${maxCvss.toFixed(1)}>Max. CVSS</jv-stat>
      </jv-stat-grid>

      <jv-section heading="Komponenten" .count=${comps.length}>
        <jv-button slot="actions" variant="ghost" @click=${() => (this.sbomOpen = !this.sbomOpen)}>
          ${this.sbomOpen ? "SBOM schlie\u00dfen" : "SBOM hochladen"}
        </jv-button>

        ${comps.length > 0
          ? html`<div class="filter-row">
              <jv-input
                label="Filter"
                placeholder="Komponente oder Version suchen"
                .value=${this.compFilter}
                @change=${(e: CustomEvent<{ value: string }>) => (this.compFilter = e.detail.value)}
              ></jv-input>
              ${this.compFilter
                ? html`<jv-button variant="ghost" @click=${() => (this.compFilter = "")}>Zur\u00fccksetzen</jv-button>`
                : nothing}
            </div>`
          : nothing}

        <div class="add-row">
          <jv-input
            label="Komponente"
            placeholder="z.B. pkg:npm/lit"
            .value=${this.compForm.component}
            @change=${(e: CustomEvent<{ value: string }>) => (this.compForm.component = e.detail.value)}
          ></jv-input>
          <jv-input
            label="Version"
            placeholder="z.B. 3.2.1"
            .value=${this.compForm.version}
            @change=${(e: CustomEvent<{ value: string }>) => (this.compForm.version = e.detail.value)}
          ></jv-input>
          <jv-button variant="primary" @click=${() => this.#addComponent()}>Hinzuf\u00fcgen</jv-button>
        </div>

        ${this.sbomOpen
          ? html`<div class="sbom">
              <p class="hint">CycloneDX-JSON als Datei ausw\u00e4hlen oder als Text einf\u00fcgen.</p>
              <input class="file" type="file" accept="application/json,.json" @change=${(e: Event) => this.#pickSbomFile(e)} />
              <jv-textarea
                label="CycloneDX JSON"
                placeholder='{ "bomFormat": "CycloneDX", ... }'
                .value=${this.sbomText}
                @change=${(e: CustomEvent<{ value: string }>) => (this.sbomText = e.detail.value)}
              ></jv-textarea>
              <jv-button-row>
                <jv-button variant="primary" ?disabled=${this.importing} @click=${() => this.#importSbom()}>
                  ${this.importing ? "Importiere\u2026" : "Importieren"}
                </jv-button>
                <jv-button @click=${() => { this.sbomOpen = false; this.sbomText = ""; }}>Abbrechen</jv-button>
              </jv-button-row>
            </div>`
          : nothing}

        ${comps.length === 0
          ? html`<div class="empty">Noch keine Komponenten. F\u00fcge eine hinzu oder importiere ein SBOM.</div>`
          : filteredComps.length === 0
            ? html`<div class="empty">Keine Komponenten passen auf den Filter „${this.compFilter}".</div>`
            : html`<table>
                <thead>
                  <tr><th>Komponente</th><th>Version</th><th></th></tr>
                </thead>
                <tbody>
                  ${filteredComps.map(
                    (c) => html`<tr>
                    <td>${c.component}</td>
                    <td>
                      ${this.editingComponent === c.component
                        ? html`<div class="ver-edit">
                            <input
                              class="ver-input"
                              .value=${this.editVersion}
                              @input=${(e: InputEvent) => (this.editVersion = (e.target as HTMLInputElement).value)}
                              @keydown=${(e: KeyboardEvent) => {
                                if (e.key === "Enter") this.#saveVersion(c.component);
                                if (e.key === "Escape") this.#cancelEditVersion();
                              }}
                            />
                            <jv-button variant="primary" ?disabled=${this.savingVersion} @click=${() => this.#saveVersion(c.component)}>OK</jv-button>
                            <jv-button @click=${() => this.#cancelEditVersion()}>\u2715</jv-button>
                          </div>`
                        : html`<jv-code>${c.version}</jv-code>`}
                    </td>
                    <td>
                      <jv-button-row align="end">
                        ${this.editingComponent === c.component
                          ? nothing
                          : html`<jv-button variant="ghost" @click=${() => this.#startEditVersion(c.component, c.version)}>Bearbeiten</jv-button>`}
                        <jv-button variant="danger" @click=${() => (this.removeComponent = c.component)}>Entfernen</jv-button>
                      </jv-button-row>
                    </td>
                  </tr>`,
                  )}
                </tbody>
              </table>`}
      </jv-section>

      <jv-section heading="Matches" .count=${matches.length}>
        ${matches.length === 0
          ? html`<div class="empty">Keine Treffer f\u00fcr die eingesetzten Komponenten.</div>`
          : html`<jv-table
              .columns=${this.#matchColumns}
              .rows=${matches}
              .renderCell=${this.#renderMatchCell}
            ></jv-table>`}
      </jv-section>

      <jv-confirm-dialog
        .open=${this.deleteOpen}
        heading="Projekt l\u00f6schen?"
        message="Das Projekt wird soft-deleted."
        @confirm=${() => this.#confirmDelete()}
        @cancel=${() => (this.deleteOpen = false)}
      ></jv-confirm-dialog>
      <jv-confirm-dialog
        .open=${this.removeComponent !== null}
        heading="Komponente entfernen?"
        message=${`Komponente ${this.removeComponent ?? ""} wird vom Projekt entfernt.`}
        @confirm=${() => this.#confirmRemoveComponent()}
        @cancel=${() => (this.removeComponent = null)}
      ></jv-confirm-dialog>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-project-detail": JvProjectDetail;
  }
}
