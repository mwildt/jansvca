import { globalTokens } from "./shared/tokens";
import "./app/JvApp";

// Inject the global CSS custom properties into the document so every component
// (and its nested shadow roots via inheritance) can use var(--jv-*).
const style = document.createElement("style");
style.textContent = globalTokens;
document.head.appendChild(style);

// Base document styling: dark backdrop, Inter font, antialiased text and a
// reset of default margins so the app shell owns the full viewport.
const base = document.createElement("style");
base.textContent = `
  html, body { margin: 0; }
  body {
    background: var(--jv-bg);
    color: var(--jv-text);
    font-family: var(--jv-font-sans);
    -webkit-font-smoothing: antialiased;
    text-rendering: optimizeLegibility;
  }
  ::selection { background: var(--jv-primary-soft); color: var(--jv-text); }
`;
document.head.appendChild(base);
