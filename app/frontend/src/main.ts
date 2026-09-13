import { globalTokens } from "./shared/tokens";
import "./app/JvApp";

// Inject the global CSS custom properties into the document so every component
// (and its nested shadow roots via inheritance) can use var(--jv-*).
const style = document.createElement("style");
style.textContent = globalTokens;
document.head.appendChild(style);

// Bootstrap: register the root custom element. The router and the shell take
// over from here.
