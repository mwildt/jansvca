import { LitElement, html, css } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { Match, ProjectView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvConfirmDialog";
import "../atoms/JvButton";
import "../atoms/JvInput";
import "../atoms/JvBadge";
import { cvssTone } from "../atoms/JvBadge";

// Organism: project detail with components, matches and edit/delete actions.
@customElement("jv-project-detail")
export class JvProjectDetail extends LitElement {
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
    .desc {
      color: var(--jv-text-muted);
      margin-bottom: var(--jv-md);
    }
    .form {
      display: grid;
      grid-template-columns: 2fr 1fr auto;
      gap: var(--jv-md);
      align-items: end;
    }
    @media (max-width: 760px) {
      .form {
        grid-template-columns: 1fr;
      }
    }
    .match-empty {
      color: var(--jv-text-muted);
      padding: var(--jv-md) 0;
    }
    .cvss {
      font-variant-numeric: tabular-nums;
      font-weight: 600;
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
      this.matchResults = m;
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
      toast("Komponente hinzugefügt", "success");
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

  async #confirmDelete(): Promise<void> {
    this.deleteOpen = false;
    try {
      await api.deleteProject(this.projectId);
      toast("Projekt gelöscht", "success");
      navigate("/projects");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  render() {
    if (this.loading) return html`<p>Lädt…</p>`;
    if (!this.project) return html`<p>Projekt nicht gefunden.</p>`;
    const p = this.project;
    return html`
      <div class="top">
        <div>
          <a href="/projects" data-link>← Projekte</a>
          <h2 style="margin:4px 0;">${this.editing ? this.edit.name : p.name}</h2>
          <div class="id">${p.id}</div>
        </div>
        <div style="display:flex; gap:8px;">
          ${this.editing
            ? html`<jv-button variant="primary" @click=${() => this.#saveEdit()}>Speichern</jv-button>
                <jv-button @click=${() => { this.editing = false; this.edit = { name: p.name, description: p.description }; }}>Abbrechen</jv-button>`
            : html`<jv-button @click=${() => (this.editing = true)}>Bearbeiten</jv-button>
                <jv-button variant="danger" @click=${() => (this.deleteOpen = true)}>Löschen</jv-button>`}
        </div>
      </div>

      ${this.editing
        ? html`<div class="section">
            <jv-input
              label="Name"
              .value=${this.edit.name}
              @change=${(e: CustomEvent<{ value: string }>) => (this.edit.name = e.detail.value)}
            ></jv-input>
            <jv-input
              label="Beschreibung"
              .value=${this.edit.description}
              @change=${(e: CustomEvent<{ value: string }>) => (this.edit.description = e.detail.value)}
            ></jv-input>
          </div>`
        : html`<p class="desc">${p.description || "Keine Beschreibung"}</p>`}

      <div class="section">
        <h3>Komponenten (${(p.components ?? []).length})</h3>
        <div class="form">
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
          <jv-button variant="primary" @click=${() => this.#addComponent()}>Hinzufügen</jv-button>
        </div>
        ${(p.components ?? []).length === 0
          ? html`<p style="color:var(--jv-text-muted);">Keine Komponenten.</p>`
          : html`<table>
              <thead>
                <tr><th>Komponente</th><th>Version</th><th></th></tr>
              </thead>
              <tbody>
                ${(p.components ?? []).map(
                  (c) => html`<tr>
                    <td>${c.component}</td>
                    <td><code>${c.version}</code></td>
                    <td>
                      <div class="row-actions">
                        <jv-button variant="danger" @click=${() => (this.removeComponent = c.component)}>Entfernen</jv-button>
                      </div>
                    </td>
                  </tr>`,
                )}
              </tbody>
            </table>`}
      </div>

      <div class="section">
        <h3>Matches (${this.matchResults.length})</h3>
        ${this.matchResults.length === 0
          ? html`<p class="match-empty">Keine Treffer für die eingesetzten Komponenten.</p>`
          : html`<table>
              <thead>
                <tr><th>Komponente</th><th>Version</th><th>Schwachstelle</th><th>CVSS</th></tr>
              </thead>
              <tbody>
                ${this.matchResults.map(
                  (m) => html`<tr>
                    <td>${m.component}</td>
                    <td><code>${m.version}</code></td>
                    <td>${m.vulnerability_identifier}</td>
                    <td><jv-badge tone=${cvssTone(m.cvss)}>${m.cvss.toFixed(1)}</jv-badge></td>
                  </tr>`,
                )}
              </tbody>
            </table>`}
      </div>

      <jv-confirm-dialog
        .open=${this.deleteOpen}
        heading="Projekt löschen?"
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
