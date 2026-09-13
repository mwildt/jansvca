import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import { navigate } from "../../shared/router";
import { toast } from "../molecules/JvToast";
import "../molecules/JvBreadcrumb";
import "../molecules/JvPageHeader";
import "../molecules/JvButtonRow";
import "../atoms/JvButton";
import "../atoms/JvCard";
import "../atoms/JvInput";
import "../atoms/JvTextarea";

// Organism: dedicated page to create a new project. Lives at /projects/new.
@customElement("jv-project-new")
export class JvProjectNew extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .card {
      max-width: 620px;
    }
    .hint {
      font-size: 0.78rem;
      color: var(--jv-text-muted);
      margin-top: var(--jv-xs);
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
      <jv-breadcrumb href="/projects">\u2190 Projekte</jv-breadcrumb>
      <jv-page-header
        heading="Neues Projekt"
        subtitle="Lege ein Projekt an und erg\u00e4nze anschlie\u00dfend Komponenten \u00fcber die Detailseite oder per SBOM-Import."
      ></jv-page-header>

      <jv-card class="card">
        <div class="hint" style="margin-bottom:var(--jv-lg);">
          <jv-input
            label="ID"
            name="id"
            placeholder="z.B. app-backend"
            .value=${this.form.id}
            @change=${(e: CustomEvent<{ value: string }>) => (this.form.id = e.detail.value)}
          ></jv-input>
          <div class="hint">Eindeutiger, URL-tauglicher Bezeichner (keine Leerzeichen).</div>
        </div>
        <jv-input
          label="Name"
          name="name"
          placeholder="Projektname"
          .value=${this.form.name}
          @change=${(e: CustomEvent<{ value: string }>) => (this.form.name = e.detail.value)}
        ></jv-input>
        <div style="margin-top:var(--jv-lg);">
          <jv-textarea
            label="Beschreibung"
            name="description"
            placeholder="Kurzbeschreibung (optional)"
            .value=${this.form.description}
            @change=${(e: CustomEvent<{ value: string }>) => (this.form.description = e.detail.value)}
          ></jv-textarea>
        </div>
        <jv-button-row style="margin-top:var(--jv-xl);">
          <jv-button variant="primary" ?disabled=${this.submitting} @click=${this.#submit}>
            ${this.submitting ? "Wird angelegt\u2026" : "Projekt anlegen"}
          </jv-button>
          <jv-button @click=${() => navigate("/projects")}>Abbrechen</jv-button>
        </jv-button-row>
      </jv-card>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-project-new": JvProjectNew;
  }
}
