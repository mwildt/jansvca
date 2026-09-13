import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { ProjectView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvEmpty";
import "../molecules/JvConfirmDialog";
import "../atoms/JvButton";
import "../atoms/JvInput";
import "../atoms/JvBadge";

// Organism: list of projects plus a create form.
@customElement("jv-project-list")
export class JvProjectList extends LitElement {
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
    .meta {
      display: flex;
      gap: var(--jv-lg);
      color: var(--jv-text-muted);
      font-size: 0.85rem;
      margin-top: var(--jv-xs);
    }
    .form {
      display: grid;
      grid-template-columns: 160px 1fr 2fr auto;
      gap: var(--jv-md);
      align-items: end;
      background: var(--jv-surface-alt);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      padding: var(--jv-lg) var(--jv-xl);
      margin-bottom: var(--jv-xl);
    }
    @media (max-width: 760px) {
      .form {
        grid-template-columns: 1fr;
      }
    }
  `;

  @state() private projects: ProjectView[] = [];
  @state() private loading = true;
  @state() private form = { id: "", name: "", description: "" };
  @state() private submitting = false;
  @state() private deleteId: string | null = null;

  connectedCallback(): void {
    super.connectedCallback();
    this.#load();
  }

  async #load(): Promise<void> {
    this.loading = true;
    try {
      this.projects = await api.listProjects();
    } catch (e) {
      toast((e as Error).message, "error");
    } finally {
      this.loading = false;
    }
  }

  async #submit(): Promise<void> {
    if (!this.form.id || !this.form.name) {
      toast("ID und Name sind erforderlich", "error");
      return;
    }
    this.submitting = true;
    try {
      await api.createProject({
        id: this.form.id,
        name: this.form.name,
        description: this.form.description,
      });
      this.form = { id: "", name: "", description: "" };
      await this.#load();
      toast("Projekt angelegt", "success");
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
      await api.deleteProject(id);
      await this.#load();
      toast("Projekt gelöscht", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  render() {
    return html`<div class="header">
        <h2>Projekte</h2>
        <jv-button variant="primary" @click=${() => this.#submit()} ?disabled=${this.submitting}
          >Projekt anlegen</jv-button
        >
      </div>

      <div class="form">
        <jv-input
          label="ID"
          name="id"
          placeholder="z.B. app-backend"
          .value=${this.form.id}
          @change=${(e: CustomEvent<{ value: string }>) => (this.form.id = e.detail.value)}
        ></jv-input>
        <jv-input
          label="Name"
          name="name"
          placeholder="Projektname"
          .value=${this.form.name}
          @change=${(e: CustomEvent<{ value: string }>) => (this.form.name = e.detail.value)}
        ></jv-input>
        <jv-input
          label="Beschreibung"
          name="description"
          placeholder="optional"
          .value=${this.form.description}
          @change=${(e: CustomEvent<{ value: string }>) => (this.form.description = e.detail.value)}
        ></jv-input>
        <jv-button variant="primary" @click=${() => this.#submit()} ?disabled=${this.submitting}
          >Anlegen</jv-button
        >
      </div>

      ${this.loading
        ? html`<p>Lädt…</p>`
        : this.projects.length === 0
          ? html`<jv-empty
              heading="Keine Projekte"
              message="Lege dein erstes Projekt über das Formular an."
            ></jv-empty>`
          : this.projects.map(
              (p) => html`
                <div class="card">
                  <a href=${`/projects/${encodeURIComponent(p.id)}`} data-link>${p.name}</a>
                  <div class="meta">
                    <span>${p.id}</span>
                    <span>${(p.components ?? []).length} Komponenten</span>
                    ${p.description ? html`<span>${p.description}</span>` : null}
                  </div>
                  <div style="margin-top:8px; display:flex; gap:8px;">
                    <jv-button
                      @click=${() => navigate(`/projects/${encodeURIComponent(p.id)}`)}
                      >Öffnen</jv-button
                    >
                    <jv-button variant="danger" @click=${() => (this.deleteId = p.id)}
                      >Löschen</jv-button
                    >
                  </div>
                </div>
              `,
            )}

      <jv-confirm-dialog
        .open=${this.deleteId !== null}
        heading="Projekt löschen?"
        message="Das Projekt wird soft-deleted und nicht mehr angezeigt."
        @confirm=${() => this.#confirmDelete()}
        @cancel=${() => (this.deleteId = null)}
      ></jv-confirm-dialog>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-project-list": JvProjectList;
  }
}
