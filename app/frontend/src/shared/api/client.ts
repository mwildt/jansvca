// Thin API client. All requests are same-origin relative paths; the BFF
// (app/cmd/jansvca-app) proxies /api/* to the backend and injects the Bearer
// token from the server-side session. The frontend never touches tokens.
import type {
  AffectedRangeView,
  AuthUser,
  Match,
  ProjectView,
  SbomImportResult,
  VulnerabilityView,
} from "./types";

export interface NewProject {
  id: string;
  name: string;
  description?: string;
}

export interface NewVulnerability {
  id: string;
  identifier: string;
  title: string;
  description?: string;
  cvss?: number;
}

export interface NewComponent {
  component: string;
  version: string;
}

export interface NewAffectedRange {
  component: string;
  version_range: string;
}

async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method, headers: {} };
  if (body !== undefined) {
    init.headers = { "Content-Type": "application/json" };
    init.body = JSON.stringify(body);
  }
  const resp = await fetch(url, init);
  if (resp.status === 204) {
    return undefined as T;
  }
  const text = await resp.text();
  let parsed: unknown = undefined;
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = { error: text };
    }
  }
  if (!resp.ok) {
    const msg =
      parsed && typeof parsed === "object" && "error" in parsed
        ? String((parsed as { error: unknown }).error)
        : `HTTP ${resp.status}`;
    throw new Error(msg);
  }
  return parsed as T;
}

export const api = {
  // --- auth -------------------------------------------------------------
  user(): Promise<AuthUser> {
    return request<AuthUser>("GET", "/api/auth/user");
  },
  loginURL(): string {
    return "/api/auth/login";
  },
  logoutURL(): string {
    return "/api/auth/logout";
  },

  // --- projects ---------------------------------------------------------
  listProjects(): Promise<ProjectView[]> {
    return request<ProjectView[]>("GET", "/api/projects");
  },
  getProject(id: string): Promise<ProjectView> {
    return request<ProjectView>("GET", `/api/projects/${encodeURIComponent(id)}`);
  },
  createProject(p: NewProject): Promise<ProjectView> {
    return request<ProjectView>("POST", "/api/projects", p);
  },
  updateProject(id: string, patch: { name?: string; description?: string }): Promise<ProjectView> {
    return request<ProjectView>("PATCH", `/api/projects/${encodeURIComponent(id)}`, patch);
  },
  deleteProject(id: string): Promise<void> {
    return request<void>("DELETE", `/api/projects/${encodeURIComponent(id)}`);
  },
  addComponent(id: string, c: NewComponent): Promise<ProjectView> {
    return request<ProjectView>("POST", `/api/projects/${encodeURIComponent(id)}/components`, c);
  },
  removeComponent(id: string, component: string): Promise<void> {
    return request<void>(
      "DELETE",
      `/api/projects/${encodeURIComponent(id)}/components/${encodeURIComponent(component)}`,
    );
  },
  updateComponent(id: string, component: string, version: string): Promise<ProjectView> {
    return request<ProjectView>(
      "PUT",
      `/api/projects/${encodeURIComponent(id)}/components/${encodeURIComponent(component)}`,
      { version },
    );
  },
  importSbom(id: string, data: string): Promise<SbomImportResult> {
    const init: RequestInit = {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: data,
    };
    return fetch(`/api/projects/${encodeURIComponent(id)}/sbom`, init).then(async (resp) => {
      const text = await resp.text();
      let parsed: unknown = undefined;
      if (text) {
        try {
          parsed = JSON.parse(text);
        } catch {
          parsed = { error: text };
        }
      }
      if (!resp.ok) {
        const msg =
          parsed && typeof parsed === "object" && "error" in parsed
            ? String((parsed as { error: unknown }).error)
            : `HTTP ${resp.status}`;
        throw new Error(msg);
      }
      return parsed as SbomImportResult;
    });
  },
  matches(id: string): Promise<Match[]> {
    return request<Match[]>("GET", `/api/projects/${encodeURIComponent(id)}/matches`);
  },

  // --- vulnerabilities --------------------------------------------------
  listVulnerabilities(): Promise<VulnerabilityView[]> {
    return request<VulnerabilityView[]>("GET", "/api/vulnerabilities");
  },
  createVulnerability(v: NewVulnerability): Promise<VulnerabilityView[]> {
    return request<VulnerabilityView[]>("POST", "/api/vulnerabilities", v);
  },
  deleteVulnerability(id: string): Promise<void> {
    return request<void>("DELETE", `/api/vulnerabilities/${encodeURIComponent(id)}`);
  },
  addAffectedRange(id: string, r: NewAffectedRange): Promise<VulnerabilityView[]> {
    return request<VulnerabilityView[]>(
      "POST",
      `/api/vulnerabilities/${encodeURIComponent(id)}/affected-ranges`,
      r,
    );
  },
  removeAffectedRange(id: string, component: string): Promise<void> {
    return request<void>(
      "DELETE",
      `/api/vulnerabilities/${encodeURIComponent(id)}/affected-ranges/${encodeURIComponent(component)}`,
    );
  },
};

export type { AffectedRangeView, Match, ProjectView, SbomImportResult, VulnerabilityView };
