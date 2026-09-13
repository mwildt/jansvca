import { LitElement, html, css, nothing, type TemplateResult } from "lit";
import { customElement, state } from "lit/decorators.js";
import { tokenStyles } from "../shared/tokens";
import { api } from "../shared/api/client";
import type { AuthUser } from "../shared/api/types";
import { navigate, subscribe, matchPattern, currentPath } from "../shared/router";
import "../components/atoms/index";
import "../components/molecules/JvToast";
import "../components/templates/index";
import "../components/organisms/index";

// Page shell: top navigation, auth gate and routed content area.
@customElement("jv-app")
export class JvApp extends LitElement {
  static styles = [tokenStyles, css`
    :host {
      display: block;
      min-height: 100vh;
      background: var(--jv-bg);
    }
    .topbar {
      background: var(--jv-surface);
      border-bottom: 1px solid var(--jv-border);
      padding: 0 var(--jv-xl);
      display: flex;
      align-items: center;
      height: 56px;
      gap: var(--jv-xl);
    }
    .brand {
      font-weight: 700;
      font-size: 1.1rem;
      letter-spacing: -0.01em;
    }
    .nav {
      display: flex;
      gap: var(--jv-md);
      flex: 1;
    }
    .nav a {
      color: var(--jv-text-muted);
      font-size: 0.92rem;
      padding: var(--jv-xs) var(--jv-sm);
      border-radius: var(--jv-sm);
    }
    .nav a.active {
      color: var(--jv-text);
      background: var(--jv-surface-alt);
    }
    .user {
      color: var(--jv-text-muted);
      font-size: 0.85rem;
      display: flex;
      align-items: center;
      gap: var(--jv-md);
    }
    .main {
      max-width: 960px;
      margin: 0 auto;
      padding: var(--jv-xl);
    }
    .login {
      max-width: 420px;
      margin: 80px auto;
      text-align: center;
      background: var(--jv-surface);
      border: 1px solid var(--jv-border);
      border-radius: var(--jv-lg);
      padding: var(--jv-xl);
    }
    .login h2 {
      margin: 0 0 var(--jv-md);
    }

    .login p {
      color: var(--jv-text-muted);
      margin-bottom: var(--jv-xl);
    }
  `];

  @state() private user: AuthUser = { authenticated: false };
  @state() private authReady = false;
  @state() private path = currentPath();

  connectedCallback(): void {
    super.connectedCallback();
    this.#checkAuth();
    subscribe(() => {
      this.path = currentPath();
    });
  }

  async #checkAuth(): Promise<void> {
    try {
      this.user = await api.user();
    } catch {
      this.user = { authenticated: false };
    } finally {
      this.authReady = true;
    }
  }

  #logout(e: Event): void {
    e.preventDefault();
    window.location.href = api.logoutURL();
  }

  #login(e: Event): void {
    e.preventDefault();
    window.location.href = api.loginURL();
  }

  render() {
    if (!this.authReady) return html`<div class="main"><p>Lädt…</p></div>`;
    const authEnabled = !window.location.search.includes("noauth");
    if (!this.user.authenticated && authEnabled) {
      return html`<div class="login">
        <h2>Anmeldung erforderlich</h2>
        <p>Bitte melde dich über den OAuth2-Provider an.</p>
        <jv-button variant="primary" @click=${this.#login}>Anmelden</jv-button>
      </div>`;
    }
    return html`
      <div class="topbar">
        <div class="brand">jansvca</div>
        <nav class="nav">${this.#navItems()}</nav>
        <div class="user">
          ${this.user.name ? html`<span>${this.user.name}</span>` : nothing}
          ${this.user.authenticated
            ? html`<a href="/api/auth/logout" @click=${this.#logout}>Abmelden</a>`
            : nothing}
        </div>
      </div>
      <div class="main">${this.#route()}</div>
      <jv-toast></jv-toast>
    `;
  }

  #navItems(): TemplateResult[] {
    const items: { label: string; href: string; match: string }[] = [
      { label: "Übersicht", href: "/", match: "/" },
      { label: "Projekte", href: "/projects", match: "/projects" },
      { label: "Schwachstellen", href: "/vulnerabilities", match: "/vulnerabilities" },
    ];
    return items.map(
      (i) => html`<a
        href=${i.href}
        data-link
        class=${this.path === i.href || (i.match !== "/" && this.path.startsWith(i.match))
          ? "active"
          : ""}
        @click=${(e: Event) => {
          e.preventDefault();
          navigate(i.href);
        }}
        >${i.label}</a
      >`,
    );
  }

  #route(): TemplateResult {
    if (this.path === "/" || this.path === "") return html`<jv-dashboard></jv-dashboard>`;
    if (this.path === "/projects") return html`<jv-project-list></jv-project-list>`;
    const pd = matchPattern("/projects/:id", this.path);
    if (pd) return html`<jv-project-detail .projectId=${pd.params.id}></jv-project-detail>`;
    if (this.path === "/vulnerabilities") return html`<jv-vuln-list></jv-vuln-list>`;
    const vd = matchPattern("/vulnerabilities/:id", this.path);
    if (vd) return html`<jv-vuln-detail .vulnId=${vd.params.id}></jv-vuln-detail>`;
    return html`<jv-not-found></jv-not-found>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-app": JvApp;
  }
}
