// Minimal hash-free history router. Components subscribe to route changes via
// `subscribe` and render based on the current path. Anchor links with
// `data-link` are intercepted for client-side navigation.

export interface Route {
  path: string;
}

type Listener = (route: Route) => void;

const listeners = new Set<Listener>();

export function currentPath(): string {
  return window.location.pathname + window.location.search;
}

export function navigate(to: string, replace = false): void {
  if (to === currentPath()) {
    notify();
    return;
  }
  if (replace) {
    window.history.replaceState({}, "", to);
  } else {
    window.history.pushState({}, "", to);
  }
  notify();
}

export function subscribe(fn: Listener): () => void {
  listeners.add(fn);
  fn({ path: currentPath() });
  return () => listeners.delete(fn);
}

function notify(): void {
  const route: Route = { path: currentPath() };
  for (const fn of listeners) fn(route);
}

window.addEventListener("popstate", notify);
document.addEventListener("click", onClick);

function onClick(e: MouseEvent): void {
  if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey) {
    return;
  }
  const target = (e.target as HTMLElement | null)?.closest("a[data-link]");
  if (!target) return;
  const href = target.getAttribute("href");
  if (!href || href.startsWith("http") || href.startsWith("//")) return;
  e.preventDefault();
  navigate(href);
}

export interface MatchResult {
  params: Record<string, string>;
}

// matchPattern matches a path like "/projects/:id" against "/projects/abc".
export function matchPattern(pattern: string, path: string): MatchResult | null {
  const cleanPath = path.split("?")[0];
  const pParts = pattern.split("/").filter(Boolean);
  const aParts = cleanPath.split("/").filter(Boolean);
  if (pParts.length !== aParts.length) return null;
  const params: Record<string, string> = {};
  for (let i = 0; i < pParts.length; i++) {
    const p = pParts[i];
    const a = aParts[i];
    if (p.startsWith(":")) {
      params[p.slice(1)] = decodeURIComponent(a);
    } else if (p !== a) {
      return null;
    }
  }
  return { params };
}
