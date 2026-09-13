import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../atoms/JvButton";
import "../atoms/JvInput";
import "../atoms/JvTextarea";

// Organism: dedicated page to create a new project. Lives at /projects/new.
@customElement("jv-project-new")
export class JvProjectNew extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .crumb a {
      color: var(--jv-text-muted);
      text-decoration: none;
      font-size: 0.85rem;
    }
    .crumb a:hover {
      color: var(--jv-text);
    }
    .head {
      margin: var(--jv-md) 0 var(--jv-2xl);
    }
    .head h1 {
      margin: 0;
      font-size: 1.6rem;
      letter-spacing: -0.02em;
    }
    .head p {
      margin: var(--jv-xs) 0 0;
      color: var(--jv-text-muted);
      font-size: 0.9rem;
      max-width: 60ch;
    }
    .card {
      max-width: 620px;
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-lg);
      padding: var(--jv-2xl);
      box-shadow: var(--jv-shadow-md);
    }
    .field {
      margin-bottom: var(--jv-lg);
    }
    .hint {
      font-size: 0.78rem;
      color: var(--jv-text-muted);
      margin-top: var(--jv-xs);
    }
    .actions {
      display: flex;
      gap: var(--jv-sm);
      margin-top: var(--jv-xl);
    }
  `;

  @state() private form = { id: "", name: "", description: "" };
  @state() private submitting = false;

  #submit = async (): Promise<void> => {
    if (!this.form.id.trim() || !this.form.name.trim()) {
      toast("ID und Name sind erforderlich", "error");
      return;
    }
    this.submitting = true;
    try {
      await api.createProject({
        id: this.form.id.trim(),
        name: this.form.name.trim(),
        description: this.form.description.trim(),
      });
      toast("Projekt angelegt", "success");
      navigate(`/projects/${encodeURIComponent(this.form.id.trim())}`);
    } catch (e) {
      toast((e as Error).message, "error");
    } finally {
      this.submitting = false;
    }
  };

  render() {
    return html`
      <div class="crumb"><a href="/projects" data-link>\u2190 Projekte</a></div>
      <div class="head">
        <h1>Neues Projekt</h1>
        <p>Lege ein Projekt an und erg\u00e4nze anschlie\u00dfend Komponenten \u00fcber die Detailseite oder per SBOM-Import.</p>
      </div>

      <div class="card">
        <div class="field">
          <jv-input
            label="ID"
            name="id"
            placeholder="z.B. app-backend"
            .value=${this.form.id}
            @change=${(e: CustomEvent<{ value: string }>) => (this.form.id = e.detail.value)}
          ></jv-input>
          <div class="hint">Eindeutiger, URL-tauglicher Bezeichner (keine Leerzeichen).</div>
        </div>
        <div class="field">
          <jv-input
            label="Name"
            name="name"
            placeholder="Projektname"
            .value=${this.form.name}
            @change=${(e: CustomEvent<{ value: string }>) => (this.form.name = e.detail.value)}
          ></jv-input>
        </div>
        <div class="field">
          <jv-textarea
            label="Beschreibung"
            name="description"
            placeholder="Kurzbeschreibung (optional)"
            .value=${this.form.description}
            @change=${(e: CustomEvent<{ value: string }>) => (this.form.description = e.detail.value)}
          ></jv-textarea>
        </div>
        <div class="actions">
          <jv-button variant="primary" ?disabled=${this.submitting} @click=${this.#submit}>
            ${this.submitting ? "Wird angelegt\u2026" : "Projekt anlegen"}
          </jv-button>
          <jv-button @click=${() => navigate("/projects")}>Abbrechen</jv-button>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-project-new": JvProjectNew;
  }
}
