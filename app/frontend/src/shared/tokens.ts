// Design tokens, exposed as CSS custom properties on the host. Components
// reference them via var(--jv-*) in their static styles, which avoids needing
// `unsafeCSS` for string interpolation inside `css` tagged templates.
import { css, unsafeCSS, type CSSResultGroup } from "lit";

export const tokens = {
  color: {
    bg: "#0b1120",
    surface: "#111827",
    surfaceAlt: "#1f2937",
    border: "#273244",
    text: "#e5e7eb",
    textMuted: "#94a3b8",
    primary: "#6366f1",
    primaryHover: "#4f46e5",
    danger: "#ef4444",
    dangerHover: "#dc2626",
    success: "#22c55e",
    warn: "#f59e0b",
  },
  radius: {
    sm: "8px",
    md: "12px",
    lg: "18px",
  },
  space: {
    xs: "4px",
    sm: "8px",
    md: "12px",
    lg: "16px",
    xl: "24px",
    "2xl": "32px",
  },
  font: {
    sans: "Inter, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif",
    mono: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
  },
} as const;

export type Tokens = typeof tokens;

// tokenVars are the CSS custom property declarations, reused both in the
// global document style and as :host styles inside components.
const tokenVars = `
    --jv-bg: #0b1120;
    --jv-surface: #111827;
    --jv-surface-alt: #1f2937;
    --jv-surface-2: #0f172a;
    --jv-border: #273244;
    --jv-border-strong: #3b4862;
    --jv-text: #e5e7eb;
    --jv-text-muted: #94a3b8;
    --jv-primary: #6366f1;
    --jv-primary-hover: #4f46e5;
    --jv-primary-soft: rgba(99, 102, 241, 0.18);
    --jv-danger: #ef4444;
    --jv-danger-hover: #dc2626;
    --jv-danger-soft: rgba(239, 68, 68, 0.18);
    --jv-success: #22c55e;
    --jv-warn: #f59e0b;
    --jv-radius-sm: 10px;
    --jv-radius-md: 12px;
    --jv-radius-lg: 18px;
    --jv-space-xs: 4px;
    --jv-space-sm: 8px;
    --jv-space-md: 12px;
    --jv-space-lg: 16px;
    --jv-space-xl: 24px;
    --jv-space-2xl: 32px;
    --jv-font-sans: Inter, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif;
    --jv-font-mono: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    --jv-shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.4);
    --jv-shadow-md: 0 8px 24px rgba(0, 0, 0, 0.45);
    --jv-shadow-lg: 0 20px 50px rgba(0, 0, 0, 0.55);
    --jv-ring: 0 0 0 3px var(--jv-primary-soft);
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
