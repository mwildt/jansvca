// Mirror of the backend read-model types (see /service projection.go). The
// frontend only knows these shapes; it never references Go internals.

export interface ComponentView {
  component: string;
  version: string;
}

export interface ProjectView {
  id: string;
  name: string;
  description: string;
  components: ComponentView[];
}

export interface AffectedRangeView {
  component: string;
  version_range: string;
}

export interface VulnerabilityView {
  id: string;
  identifier: string;
  title: string;
  description: string;
  cvss: number;
  affected: AffectedRangeView[];
}

export interface Match {
  project_id: string;
  component: string;
  version: string;
  vulnerability_id: string;
  vulnerability_identifier: string;
  cvss: number;
}

export interface AuthUser {
  authenticated: boolean;
  subject?: string;
  name?: string;
}

export interface ApiError {
  error: string;
}
