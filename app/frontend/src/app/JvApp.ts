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

// Page shell: top navigation, auth gate and routed content area. The login
// gate mirrors the idp login design (brand mark, card surface, primary CTA).
@customElement("jv-app")
export class JvApp extends LitElement {
  static styles = [tokenStyles, css`
    :host {
      display: block;
      min-height: 100vh;
      background:
        radial-gradient(1200px 600px at 100% -10%, rgba(99, 102, 241, 0.12), transparent 60%),
        radial-gradient(900px 500px at -10% 0%, rgba(34, 197, 94, 0.06), transparent 55%),
        var(--jv-bg);
      font-family: var(--jv-font-sans);
      color: var(--jv-text);
    }
    .topbar {
      position: sticky;
      top: 0;
      z-index: 50;
      background: rgba(15, 23, 42, 0.72);
      backdrop-filter: blur(14px);
      -webkit-backdrop-filter: blur(14px);
      border-bottom: 1px solid var(--jv-border);
      padding: 0 var(--jv-2xl);
      display: flex;
      align-items: center;
      height: 60px;
      gap: var(--jv-xl);
    }
    .brand {
      display: flex;
      align-items: center;
      gap: var(--jv-sm);
      font-weight: 700;
      font-size: 1.05rem;
      letter-spacing: -0.02em;
    }
    .brand .mark {
      width: 26px;
      height: 26px;
      border-radius: 8px;
      background: linear-gradient(135deg, var(--jv-primary), #8b5cf6);
      box-shadow: 0 6px 16px rgba(99, 102, 241, 0.45);
    }
    .nav {
      display: flex;
      gap: var(--jv-xs);
      flex: 1;
    }
    .nav a {
      color: var(--jv-text-muted);
      font-size: 0.9rem;
      font-weight: 500;
      padding: var(--jv-sm) var(--jv-md);
      border-radius: var(--jv-sm);
      transition: color 0.12s ease, background 0.12s ease;
    }
    .nav a:hover {
      color: var(--jv-text);
      background: var(--jv-surface-alt);
    }
    .nav a.active {
      color: var(--jv-text);
      background: var(--jv-surface-alt);
      box-shadow: inset 0 -2px 0 var(--jv-primary);
    }
    .user {
      color: var(--jv-text-muted);
      font-size: 0.85rem;
      display: flex;
      align-items: center;
      gap: var(--jv-md);
    }
    .user .avatar {
      width: 28px;
      height: 28px;
      border-radius: 999px;
      background: linear-gradient(135deg, #6366f1, #8b5cf6);
      color: #fff;
      font-size: 0.8rem;
      font-weight: 600;
      display: inline-flex;
      align-items: center;
      justify-content: center;
    }
    .user a {
      color: var(--jv-text-muted);
      text-decoration: none;
    }
    .user a:hover {
      color: var(--jv-text);
    }
    .main {
      max-width: 1120px;
      margin: 0 auto;
      padding: var(--jv-2xl) var(--jv-2xl) var(--jv-2xl);
    }
    .gate {
      min-height: calc(100vh - 60px);
      display: flex;
      align-items: center;
      justify-content: center;
      padding: var(--jv-2xl);
    }
    .login {
      width: 100%;
      max-width: 380px;
      text-align: center;
    }
    .login .brand {
      justify-content: center;
      margin-bottom: var(--jv-xl);
    }
    .login .brand .mark {
      width: 30px;
      height: 30px;
      border-radius: 9px;
    }
    .login h2 {
      margin: 0 0 var(--jv-sm);
      font-size: 1.25rem;
      font-weight: 700;
    }
    .login p {
      color: var(--jv-text-muted);
      font-size: 0.85rem;
      margin: 0 0 var(--jv-xl);
    }
    .loading {
      padding: var(--jv-2xl);
      display: flex;
      justify-content: center;
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
    if (!this.authReady) return html`<div class="loading"><jv-spinner></jv-spinner></div>`;
    const authEnabled = !window.location.search.includes("noauth");
    if (!this.user.authenticated && authEnabled) {
      return html`<div class="gate">
        <jv-card class="login">
          <div class="brand"><span class="mark"></span><span>jansvca</span></div>
          <h2>Anmeldung</h2>
          <p>Melde dich über den OAuth2-Provider an.</p>
          <jv-button variant="primary" @click=${this.#login}>Anmelden</jv-button>
        </jv-card>
      </div>`;
    }
    return html`
      <div class="topbar">
        <div class="brand"><span class="mark"></span><span>jansvca</span></div>
        <nav class="nav">${this.#navItems()}</nav>
        <div class="user">
          ${this.user.name ? html`<span class="avatar">${this.user.name.slice(0, 1).toUpperCase()}</span>` : nothing}
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
      { label: "\u00dcbersicht", href: "/", match: "/" },
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
        >${i.label}</a>
      >`,
    );
  }

  #route(): TemplateResult {
    if (this.path === "/" || this.path === "") return html`<jv-dashboard></jv-dashboard>`;
    if (this.path === "/projects" || this.path === "/projects/") return html`<jv-project-list></jv-project-list>`;
    if (this.path === "/projects/new") return html`<jv-project-new></jv-project-new>`;
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
