import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { ProjectView, VulnerabilityView } from "../../shared/api/types";
import { navigate } from "../../shared/router";

// Template: dashboard overview with counts and quick links.
@customElement("jv-dashboard")
export class JvDashboard extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: var(--jv-lg);
      margin-bottom: var(--jv-xl);
    }
    .stat {
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      padding: var(--jv-lg) var(--jv-xl);
      cursor: pointer;
    }
    .stat:hover {
      border-color: var(--jv-primary);
    }
    .num {
      font-size: 2rem;
      font-weight: 700;
      color: var(--jv-text);
    }
    .label {
      color: var(--jv-text-muted);
      font-size: 0.85rem;
    }
    .intro {
      color: var(--jv-text-muted);
      max-width: 60ch;
      margin-bottom: var(--jv-xl);
    }
  `;

  @state() private projects = 0;
  @state() private vulns = 0;
  @state() private loading = true;

  connectedCallback(): void {
    super.connectedCallback();
    this.#load();
  }

  async #load(): Promise<void> {
    try {
      const [p, v] = await Promise.all([
        api.listProjects().catch(() => [] as ProjectView[]),
        api.listVulnerabilities().catch(() => [] as VulnerabilityView[]),
      ]);
      this.projects = p.length;
      this.vulns = v.length;
    } finally {
      this.loading = false;
    }
  }

  render() {
    return html`
      <h2>Übersicht</h2>
      <p class="intro">
        Schwachstellen-Tracking: erfasse Projekte mit ihren Komponenten und
        Schwachstellen mit Version-Ranges. jansvca ermittelt, welche
        Schwachstellen auf deine eingesetzten Komponenten zutreffen.
      </p>
      <div class="grid">
        <div class="stat" @click=${() => navigate("/projects")}>
          <div class="num">${this.loading ? "…" : this.projects}</div>
          <div class="label">Projekte</div>
        </div>
        <div class="stat" @click=${() => navigate("/vulnerabilities")}>
          <div class="num">${this.loading ? "…" : this.vulns}</div>
          <div class="label">Schwachstellen</div>
        </div>
      </div>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-dashboard": JvDashboard;
  }
}
