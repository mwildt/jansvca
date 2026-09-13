// Design tokens, exposed as CSS custom properties on the host. Components
// reference them via var(--jv-*) in their static styles, which avoids needing
// `unsafeCSS` for string interpolation inside `css` tagged templates.
import { css, unsafeCSS, type CSSResultGroup } from "lit";

export const tokens = {
  color: {
    bg: "#f8fafc",
    surface: "#ffffff",
    surfaceAlt: "#f1f5f9",
    border: "#e2e8f0",
    text: "#0f172a",
    textMuted: "#64748b",
    primary: "#1d4ed8",
    primaryHover: "#1e40af",
    danger: "#dc2626",
    dangerHover: "#b91c1c",
    success: "#16a34a",
    warn: "#d97706",
  },
  radius: {
    sm: "6px",
    md: "10px",
    lg: "14px",
  },
  space: {
    xs: "4px",
    sm: "8px",
    md: "12px",
    lg: "16px",
    xl: "24px",
  },
  font: {
    sans: "system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif",
    mono: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
  },
} as const;

export type Tokens = typeof tokens;

// tokenVars are the CSS custom property declarations, reused both in the
// global document style and as :host styles inside components.
const tokenVars = `
    --jv-bg: #f8fafc;
    --jv-surface: #ffffff;
    --jv-surface-alt: #f1f5f9;
    --jv-border: #e2e8f0;
    --jv-text: #0f172a;
    --jv-text-muted: #64748b;
    --jv-primary: #1d4ed8;
    --jv-primary-hover: #1e40af;
    --jv-danger: #dc2626;
    --jv-danger-hover: #b91c1c;
    --jv-success: #16a34a;
    --jv-warn: #d97706;
    --jv-radius-sm: 6px;
    --jv-radius-md: 10px;
    --jv-radius-lg: 14px;
    --jv-space-xs: 4px;
    --jv-space-sm: 8px;
    --jv-space-md: 12px;
    --jv-space-lg: 16px;
    --jv-space-xl: 24px;
    --jv-font-sans: system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif;
    --jv-font-mono: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
`;

// globalTokens is injected into the document head so the custom properties are
// available to all (shadow) roots via inheritance.
export const globalTokens = `:root {${tokenVars}}`;

// tokenStyles can be included in a component's static styles to also declare the
// properties on its own :host (useful for top-level shells).
export const tokenStyles: CSSResultGroup = css`
  :host {
    ${unsafeCSS(tokenVars)}
  }
`;
