import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { ProjectView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvPageHeader";
import "../molecules/JvStatGrid";
import "../molecules/JvEmpty";
import "../molecules/JvConfirmDialog";
import "../atoms/JvButton";
import "../atoms/JvBadge";
import "../atoms/JvStat";
import "../atoms/JvSpinner";

// Organism: project overview list. Creation happens on the dedicated
// /projects/new page so this view stays a focused list.
@customElement("jv-project-list")
export class JvProjectList extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
      gap: var(--jv-md);
    }
    .card {
      display: flex;
      flex-direction: column;
      gap: var(--jv-sm);
      padding: var(--jv-lg);
      border-radius: var(--jv-r-md);
      box-shadow: var(--jv-shadow-sm);
    }
    .card .name {
      font-weight: 600;
      font-size: 1.05rem;
      color: var(--jv-text);
      display: flex;
      align-items: center;
      gap: var(--jv-sm);
    }
    .card .id {
      font-family: var(--jv-font-mono);
      font-size: 0.8rem;
      color: var(--jv-text-muted);
    }
    .card .desc {
      color: var(--jv-text-muted);
      font-size: 0.86rem;
      min-height: 1.3em;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    .card .footer {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-top: var(--jv-xs);
    }
    .card .footer .comp {
      display: inline-flex;
      align-items: center;
      gap: var(--jv-xs);
      color: var(--jv-text-muted);
      font-size: 0.8rem;
    }
    .card .footer .del {
      appearance: none;
      background: transparent;
      border: none;
      color: var(--jv-text-muted);
      cursor: pointer;
      padding: var(--jv-xs);
      border-radius: var(--jv-r-sm);
      font-size: 0.9rem;
      line-height: 1;
    }
    .card .footer .del:hover {
      color: var(--jv-danger);
      background: var(--jv-danger-soft);
    }
  `;

  @state() private projects: ProjectView[] = [];
  @state() private loading = true;
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

  async #confirmDelete(): Promise<void> {
    if (!this.deleteId) return;
    const id = this.deleteId;
    this.deleteId = null;
    try {
      await api.deleteProject(id);
      await this.#load();
      toast("Projekt gel\u00f6scht", "success");
    } catch (e) {
      toast((e as Error).message, "error");
    }
  }

  render() {
    const totalComponents = this.projects.reduce((n, p) => n + (p.components?.length ?? 0), 0);
    return html`
      <jv-page-header
        heading="Projekte"
        subtitle="Verwalte Projekte und ihre eingesetzten Komponenten."
      >
        <jv-button slot="actions" variant="primary" @click=${() => navigate("/projects/new")}
          >+ Neues Projekt</jv-button
        >
      </jv-page-header>

      ${!this.loading && this.projects.length > 0
        ? html`<jv-stat-grid>
            <jv-stat value=${this.projects.length}>Projekte</jv-stat>
            <jv-stat value=${totalComponents}>Komponenten</jv-stat>
          </jv-stat-grid>`
        : null}

      ${this.loading
        ? html`<jv-spinner></jv-spinner>`
        : this.projects.length === 0
          ? html`<jv-empty
              heading="Keine Projekte"
              message="Lege dein erstes Projekt an, um Komponenten und Schwachstellen zu tracken."
            >
              <jv-button variant="primary" @click=${() => navigate("/projects/new")}>Projekt anlegen</jv-button>
            </jv-empty>`
          : html`<div class="grid">
              ${this.projects.map(
                (p) => html`
                  <jv-card
                    interactive
                    class="card"
                    @click=${() => navigate(`/projects/${encodeURIComponent(p.id)}`)}
                  >
                    <div class="name">${p.name}</div>
                    <div class="id">${p.id}</div>
                    <div class="desc">${p.description || "Keine Beschreibung"}</div>
                    <div class="footer">
                      <span class="comp">
                        <jv-badge tone="info">${(p.components ?? []).length}</jv-badge>
                        Komponenten
                      </span>
                      <button
                        class="del"
                        title="L\u00f6schen"
                        @click=${(e: Event) => {
                          e.stopPropagation();
                          this.deleteId = p.id;
                        }}
                      >\u2715</button>
                    </div>
                  </jv-card>
                `,
              )}
            </div>`}

      <jv-confirm-dialog
        .open=${this.deleteId !== null}
        heading="Projekt l\u00f6schen?"
        message="Das Projekt wird soft-deleted und nicht mehr angezeigt."
        @confirm=${() => this.#confirmDelete()}
        @cancel=${() => (this.deleteId = null)}
      ></jv-confirm-dialog>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-project-list": JvProjectList;
  }
}
