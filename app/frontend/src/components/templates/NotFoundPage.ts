import { LitElement, html, css } from "lit";
import { customElement } from "lit/decorators.js";
import "../molecules/JvEmpty";
import "../atoms/JvButton";
import { navigate } from "../../shared/router";

@customElement("jv-not-found")
export class JvNotFound extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    jv-empty {
      margin-top: var(--jv-2xl);
    }
  `;
  render() {
    return html`<jv-empty heading="404" message="Seite nicht gefunden.">
      <jv-button variant="primary" @click=${() => navigate("/")}>Zur \u00dcbersicht</jv-button>
    </jv-empty>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    "jv-not-found": JvNotFound;
  }
}
