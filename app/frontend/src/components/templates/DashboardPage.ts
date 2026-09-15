import { LitElement, html, css } from "lit";
import { customElement, state } from "lit/decorators.js";
import { api } from "../../shared/api/client";
import type { ProjectView } from "../../shared/api/types";
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
    .intro {
      color: var(--jv-text-muted);
      max-width: 60ch;
      margin-bottom: var(--jv-xl);
      line-height: 1.6;
    }
    jv-stat {
      cursor: pointer;
      transition: border-color 0.14s ease, transform 0.1s ease, box-shadow 0.14s ease;
    }
    jv-stat:hover {
      border-color: var(--jv-primary);
      transform: translateY(-2px);
      box-shadow: var(--jv-shadow-md);
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
        api.listVulnerabilities({ page_size: 1 }).catch(() => ({ items: [], total: 0 })),
      ]);
      this.projects = p.length;
      this.vulns = v.total;
    } finally {
      this.loading = false;
    }
  }

  render() {
    return html`
      <jv-page-header heading="\u00dcbersicht"></jv-page-header>
      <p class="intro">
        Schwachstellen-Tracking: erfasse Projekte mit ihren Komponenten und
        Schwachstellen mit Version-Ranges. jansvca ermittelt, welche
        Schwachstellen auf deine eingesetzten Komponenten zutreffen.
      </p>
      <jv-stat-grid>
        <jv-stat value=${this.loading ? "\u2026" : this.projects} @click=${() => navigate("/projects")}>
          Projekte
        </jv-stat>
        <jv-stat value=${this.loading ? "\u2026" : this.vulns} @click=${() => navigate("/vulnerabilities")}>
          Schwachstellen
        </jv-stat>
      </jv-stat-grid>
    `;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-dashboard": JvDashboard;
  }
}
