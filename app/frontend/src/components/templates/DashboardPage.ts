import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { ProjectView, VulnerabilityView } from "../../shared/api/types";
import { navigate } from "../../shared/router";
import "../molecules/JvPageHeader";
import "../molecules/JvStatGrid";
import "../atoms/JvStat";
import "../atoms/JvButton";

// Template: dashboard overview with counts and quick links.
@customElement("jv-dashboard")
export class JvDashboard extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .hero {
      background:
        radial-gradient(600px 300px at 0% 0%, rgba(99, 102, 241, 0.18), transparent 70%),
        var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-lg);
      padding: var(--jv-2xl);
      margin-bottom: var(--jv-xl);
      box-shadow: var(--jv-shadow-md);
    }
    .hero h2 {
      margin: 0 0 var(--jv-sm);
      font-size: 1.5rem;
      font-weight: 700;
      letter-spacing: -0.02em;
    }
    .hero p {
      margin: 0 0 var(--jv-xl);
      color: var(--jv-text-muted);
      max-width: 62ch;
      line-height: 1.6;
    }
    jv-stat {
      cursor: pointer;
      transition: border-color 0.16s ease, transform 0.12s ease, box-shadow 0.16s ease;
    }
    jv-stat:hover {
      border-color: var(--jv-primary);
      transform: translateY(-3px);
      box-shadow: var(--jv-shadow-md);
    }
    .quick {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: var(--jv-md);
      margin-top: var(--jv-xl);
    }
    .quick .card {
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-md);
      padding: var(--jv-lg) var(--jv-xl);
      box-shadow: var(--jv-shadow-sm);
      display: flex;
      flex-direction: column;
      gap: var(--jv-sm);
      cursor: pointer;
      transition: border-color 0.16s ease, transform 0.12s ease, box-shadow 0.16s ease;
    }
    .quick .card:hover {
      border-color: var(--jv-primary);
      transform: translateY(-3px);
      box-shadow: var(--jv-shadow-md);
    }
    .quick .card .ttl {
      font-weight: 600;
      font-size: 1rem;
      color: var(--jv-text);
    }
    .quick .card .sub {
      color: var(--jv-text-muted);
      font-size: 0.82rem;
      line-height: 1.5;
    }
    .quick .card .go {
      margin-top: auto;
      color: var(--jv-primary);
      font-size: 0.82rem;
      font-weight: 600;
    }
  `;

  @state() private projects: ProjectView[] = [];
  @state() private vulns: VulnerabilityView[] = [];
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
      this.projects = p;
      this.vulns = v;
    } finally {
      this.loading = false;
    }
  }

  render() {
    const totalComponents = this.projects.reduce((n, p) => n + (p.components?.length ?? 0), 0);
    return html`
      <jv-page-header heading="Übersicht"></jv-page-header>

      <div class="hero">
        <h2>Schwachstellen-Tracking</h2>
        <p>
          Erfasse Projekte mit ihren Komponenten und Schwachstellen mit
          Version-Ranges. jansvca ermittelt, welche Schwachstellen auf deine
          eingesetzten Komponenten zutreffen.
        </p>
        <jv-button variant="primary" @click=${() => navigate("/projects/new")}
          >+ Neues Projekt</jv-button
        >
      </div>

      <jv-stat-grid>
        <jv-stat
          tone="primary"
          value=${this.loading ? "…" : this.projects.length}
          @click=${() => navigate("/projects")}
        >
          Projekte
        </jv-stat>
        <jv-stat
          value=${this.loading ? "…" : totalComponents}
          @click=${() => navigate("/projects")}
        >
          Komponenten
        </jv-stat>
        <jv-stat
          tone="danger"
          value=${this.loading ? "…" : this.vulns.length}
          @click=${() => navigate("/vulnerabilities")}
        >
          Schwachstellen
        </jv-stat>
      </jv-stat-grid>

      <div class="quick">
        <div class="card" @click=${() => navigate("/projects")}>
          <span class="ttl">Projekte</span>
          <span class="sub">Verwalte Projekte und ihre eingesetzten Komponenten.</span>
          <span class="go">Öffnen →</span>
        </div>
        <div class="card" @click=${() => navigate("/vulnerabilities")}>
          <span class="ttl">Schwachstellen</span>
          <span class="sub">Lege Schwachstellen mit betroffenen Version-Ranges an.</span>
          <span class="go">Öffnen →</span>
        </div>
        <div class="card" @click=${() => navigate("/projects/new")}>
          <span class="ttl">Neues Projekt</span>
          <span class="sub">Lege ein Projekt an und importiere ein SBOM.</span>
          <span class="go">Anlegen →</span>
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
