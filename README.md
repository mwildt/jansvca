# jansvca

Schwachstellen-Tracking-Anwendung. Erfaßt **Projekte** (mit Komponenten und
Koordinaten) und **Schwachstellen** (mit betroffenen Version-Ranges) und
ermittelt, welche Schwachstellen auf die eingesetzten Komponenten zutreffen.

Die Anwendung besteht aus zwei eigenständigen Go-Projekten, die als getrennte
Container laufen:

- **`/service`** – das hexagonale Go-Backend mit der REST-API (Event Sourcing,
  OAuth2-Introspection). Eigenes Go-Modul (`github.com/mwildt/jansvca/service`).
- **`/app`** – die Web-Anwendung (BFF): liefert die SPA aus, steuert die
  Session und fungiert als authentifizierendes Gateway. Eigenes Go-Modul
  (`github.com/mwildt/jansvca/app`).

Das Frontend liegt in **`/app/frontend`** und basiert auf [Lit](https://lit.dev)
mit einem Atomic-Design-Ansatz (Atoms → Molecules → Organisms → Templates →
Page Shell).

## Architektur

```
Browser ──(Cookie-Session)──► app (BFF :8080)
                                 │
                                 ├─ /api/* ──► service (Backend :8080) ──► idp /introspect
                                 ├─ /proxy/* ─► externe Upstreams (Gateway)
                                 └─ /* ──► gebaute SPA (/app/dist)

Login: Browser ──► idp /authorize (:8090) ──(code)──► app /api/auth/callback
         app ──(code)──► idp /token ──(access_token in Session)
```

- **`/service`** – hexagonale Module (Ports & Adapters), Event Sourcing
  (WAL, append-only), pro Modul ein eigener Eventstore, OAuth2 via
  Introspection.
- **`/app`** – Backend-for-Frontend (BFF):
  - Liefert die gebaute SPA unter `/` aus (SPA-Fallback für Client-Routes).
  - Steuert die Session serverseitig: der OAuth2-Access-Token wird **nicht** an
    den Browser gegeben, sondern in einer Cookie-Session gehalten. Der Browser
    hält nur eine opake Session-ID.
  - Proxt `/api/*` an das Backend (`/service`) und injiziert dabei den
    Bearer-Token aus der Session, sodass das Backend weiter über OAuth2
    validiert.
  - Fungiert als **Gateway** vor weiteren Services: konfigurierte Upstreams
    werden unter einem Pfad-Prefix proxt, ebenfalls mit dem Session-Token.
- **`/idp`** – minimaler OAuth2-Provider (Authorization-Code-Flow):
  - Nutzer und Clients aus `idp-config.yaml`; Passwörter als bcrypt-Hash
    (Klartext-Passwörter werden beim Laden gehasht, nur für lokale Entwicklung).
  - Endpunkte: `GET/POST /authorize` (eingebautes Login-Formular),
    `POST /token` (Code-Austausch), `POST /revoke` (RFC 7009).
  - `POST /introspect` (RFC 7662) läuft auf einem **eigenen Port**
    (`IDP_INTROSPECT_ADDR`), damit Introspection serverseitig getrennt vom
    browserfähigen Haupt-Port betrieben werden kann (z. B. hinter einem
    TLS/mTLS-terminierenden Listener). Ohne `IDP_INTROSPECT_ADDR` bleibt
    Introspection auf dem Haupt-Port.
  - Pro Client kann in `idp-config.yaml` `require_mtls: true` gesetzt werden:
    dann wird Introspection für diesen Client nur über TLS-Verbindungen
    akzeptiert (Plain-HTTP-Calls werden mit `invalid_client` abgewiesen).
  - Codes und Tokens sind zufällige, kurze, einmalige/opake Strings im Speicher.
- **`/app/frontend`** – Lit-SPA (Vite + TypeScript), Atomic Design.

Siehe `docs/architecture.md` für die Backend-Architektur.

## Build & Test

Alle drei Module sind unabhängige Go-Projekte mit eigenem `go.mod`.

```bash
# Backend (service)
(cd service && go build ./... && go test ./...)

# Web-Anwendung (BFF + Gateway)
(cd app && go build ./... && go test ./...)

# OAuth2-Provider (idp)
(cd idp && go build ./... && go test ./...)

# Frontend (SPA)
cd app/frontend
npm install
npm run build        # baut nach app/dist
npm run typecheck
```

## Docker (drei Container)

Es gibt drei Images, die über `docker-compose` zusammen gestartet werden: der
`service` ist der Upstream der `app`, und der `idp` ist der OAuth2-Provider für
den Login. Das CI-Build veröffentlicht alle Images in die **GitHub Container
Registry**:

- `ghcr.io/mwildt/jansvca-idp`
- `ghcr.io/mwildt/jansvca-service`
- `ghcr.io/mwildt/jansvca-app`

### Mit docker compose

```bash
docker compose build
docker compose up
```

Die App ist unter `http://localhost:8080` erreichbar und proxt `/api/*` an den
Service-Container (`http://service:8080`). Der IdP ist unter `http://localhost:8090`
erreichbar (für den `/authorize`-Redirect im Browser); Token-Aufrufe laufen
serverseitig über das Compose-Netzwerk (`http://idp:8080`), Introspection
über den dedizierten Port (`http://idp:8081`).

Login: Benutzer `admin`, Passwort `admin` (siehe `idp-config.yaml`).

### Einzeln bauen

```bash
# IdP-Image (OAuth2-Provider, alpine)
docker build -t jansvca-idp -f idp/Dockerfile .

# Service-Image (FROM scratch, distroless)
docker build -t jansvca-service -f service/Dockerfile .

# App-Image (Lit SPA + Go-BFF, alpine)
docker build -t jansvca-app -f app/Dockerfile .
```

### Ausführen — lokal mit IdP (drei Container, OAuth2-Flow)

```bash
docker compose up
# App:  http://localhost:8080   (Login via http://localhost:8090/authorize)
# IdP: http://localhost:8090
# Login: admin / admin
```

### Ausführen — ohne OAuth2

```bash
# Service
docker run --rm -d --name jansvca-service -p 18080:8080 -v jansvca-data:/data \
  -e JANSVCA_AUTH=off \
  ghcr.io/mwildt/jansvca-service:latest

# App (proxt /api an den Service)
docker run --rm -d --name jansvca-app -p 8080:8080 \
  -e JANSVCA_BACKEND_URL=http://host.docker.internal:18080 \
  ghcr.io/mwildt/jansvca-app:latest
```

(Mit Docker Desktop ist `host.docker.internal` verfügbar; andernfalls ein
gemeinsames Netzwerk nutzen, siehe `docker-compose.yml`.)

### Ausführen — mit OAuth2 (Authorization-Code-Flow im BFF)

```bash
docker run --rm -p 8080:8080 -v jansvca-data:/data \
  -e JANSVCA_OAUTH2_AUTHORIZATION_URL=https://provider/authorize \
  -e JANSVCA_OAUTH2_TOKEN_URL=https://provider/token \
  -e JANSVCA_OAUTH2_INTROSPECTION_URL=https://provider/introspect \
  -e JANSVCA_OAUTH2_CLIENT_ID=... \
  -e JANSVCA_OAUTH2_CLIENT_SECRET=... \
  -e JANSVCA_OAUTH2_REDIRECT_URL=https://app.example.com/api/auth/callback \
  -e JANSVCA_SECURE_COOKIES=true \
  ghcr.io/mwildt/jansvca-app:latest
```

Gateway-Upstreams konfigurieren (komma-separiert, optional `/strip`):

```bash
-e JANSVCA_GATEWAY_UPSTREAMS="/proxy/svc=https://api.example.com/strip,/proxy/other=https://other.example.com"
```

## Lokal ausführen

### Entwicklung (Frontend-Dev-Server + BFF + Service + IdP)

In vier Terminals:

```bash
# 1) IdP (OAuth2-Provider, :8090; Introspection zusätzlich auf :8091)
(cd idp && IDP_CONFIG=../idp-config.yaml IDP_ADDR=:8090 IDP_INTROSPECT_ADDR=:8091 go run ./cmd/idp)

# 2) Backend-Service (:8081, validiert Token über den IdP)
(cd service && JANSVCA_DATA_DIR=./data JANSVCA_ADDR=:8081 \
   JANSVCA_OAUTH2_INTROSPECTION_URL=http://localhost:8091/introspect \
   JANSVCA_OAUTH2_CLIENT_ID=jansvca-app JANSVCA_OAUTH2_CLIENT_SECRET=jansvca-app-secret \
   go run ./cmd/jansvca)

# 3) BFF (proxt /api an :8081, OAuth2 gegen den IdP)
(cd app && JANSVCA_BACKEND_URL=http://localhost:8081 JANSVCA_ADDR=:8080 \
   JANSVCA_SPA_DIR=./dist \
   JANSVCA_OAUTH2_AUTHORIZATION_URL=http://localhost:8090/authorize \
   JANSVCA_OAUTH2_TOKEN_URL=http://localhost:8090/token \
   JANSVCA_OAUTH2_INTROSPECTION_URL=http://localhost:8091/introspect \
   JANSVCA_OAUTH2_CLIENT_ID=jansvca-app JANSVCA_OAUTH2_CLIENT_SECRET=jansvca-app-secret \
   JANSVCA_OAUTH2_REDIRECT_URL=http://localhost:8080/api/auth/callback \
   go run ./cmd/jansvca-app)

# 4) Frontend-Dev-Server (proxt /api an den BFF auf :8080)
cd app/frontend && npm run dev
```

Login über `http://localhost:8080/api/auth/login` leitet zum IdP auf `:8090`
weiter; nach Anmeldung (admin/admin) erfolgt der Callback am BFF.

### Produktion (BFF liefert gebaute SPA)

```bash
(cd app/frontend && npm run build)        # erzeugt app/dist
(cd service && JANSVCA_DATA_DIR=./data JANSVCA_ADDR=:8081 JANSVCA_AUTH=off go run ./cmd/jansvca) &
(cd app && JANSVCA_BACKEND_URL=http://localhost:8081 JANSVCA_ADDR=:8080 \
   JANSVCA_SPA_DIR=./dist go run ./cmd/jansvca-app)
```

## Umgebungsvariablen

### `/idp`

| Variable | Default | Bedeutung |
|----------|---------|----------|
| `IDP_ADDR` | `:8080` | Listen-Adresse des IdP |
| `IDP_CONFIG` | `./idp-config.yaml` | Pfad zur YAML-Konfig (Nutzer + Clients) |

### `/service`

| Variable | Default | Bedeutung |
|----------|---------|-----------|
| `JANSVCA_ADDR` | `:8080` | Listen-Adresse des Backends |
| `JANSVCA_DATA_DIR` | `./data` | Verzeichnis der WAL-Eventstores |
| `JANSVCA_AUTH` | `on` | Auth aktiviert; `off` deaktiviert sie explizit (nur für lokale Entwicklung) |
| `JANSVCA_OAUTH2_INTROSPECTION_URL` | – | OAuth2-Introspection-Endpoint |
| `JANSVCA_OAUTH2_CLIENT_ID` | – | OAuth2-Client-ID |
| `JANSVCA_OAUTH2_CLIENT_SECRET` | – | OAuth2-Client-Secret |
| `JANSVCA_OSV_SYNC` | `on` | OSV-Sync aktiviert (`off` deaktiviert) |
| `JANSVCA_OSV_BASE_URL` | `https://storage.googleapis.com/osv-vulnerabilities` | Basis-URL des OSV-Daten-Exports |
| `JANSVCA_OSV_INTERVAL` | `60m` | Aktualisierungs-Intervall der OSV-Sync (Go-Duration) |

### `/app` (BFF)

| Variable | Default | Bedeutung |
|----------|---------|-----------|
| `JANSVCA_ADDR` | `:8080` | Listen-Adresse des BFF |
| `JANSVCA_BACKEND_URL` | – | Basis-URL des Backends (http(s)://…) |
| `JANSVCA_SPA_DIR` | `/app/dist` | Verzeichnis der gebauten SPA |
| `JANSVCA_SECURE_COOKIES` | `true` | `Secure`-Flag der Session-Cookies (für lokales http auf `false` setzen) |
| `JANSVCA_OAUTH2_AUTHORIZATION_URL` | – | OAuth2-Authorize-Endpoint |
| `JANSVCA_OAUTH2_TOKEN_URL` | – | OAuth2-Token-Endpoint |
| `JANSVCA_OAUTH2_INTROSPECTION_URL` | – | OAuth2-Introspection-Endpoint |
| `JANSVCA_OAUTH2_REVOCATION_URL` | – | OAuth2-Revocation-Endpoint (RFC 7009, Logout) |
| `JANSVCA_OAUTH2_CLIENT_ID` | – | OAuth2-Client-ID |
| `JANSVCA_OAUTH2_CLIENT_SECRET` | – | OAuth2-Client-Secret |
| `JANSVCA_OAUTH2_REDIRECT_URL` | `http://localhost:8080/api/auth/callback` | BFF-Callback-URL |
| `JANSVCA_OAUTH2_SCOPE` | `openid profile` | angeforderte Scopes |
| `JANSVCA_GATEWAY_UPSTREAMS` | – | Gateway-Upstreams (s.o.) |

## REST-Übersicht (Backend)

- `POST   /api/projects`                              Projekt anlegen
- `GET    /api/projects`                              Projekte auflisten
- `GET    /api/projects/{id}`                         Projektdetail
- `PATCH  /api/projects/{id}`                         Projekt umbenennen/beschreiben
- `DELETE /api/projects/{id}`                         Projekt soft-löschen
- `POST   /api/projects/{id}/components`              Komponente hinzufügen
- `PUT    /api/projects/{id}/components/{component}`  Komponentenversion ändern
- `DELETE /api/projects/{id}/components/{component}`  Komponente entfernen
- `POST   /api/projects/{id}/sbom`                    CycloneDX-SBOM importieren (Komponenten anlegen/aktualisieren)
- `GET    /api/projects/{id}/matches`                 Treffer für Projekt
- `POST   /api/vulnerabilities`                       Schwachstelle anlegen
- `GET    /api/vulnerabilities`                       Schwachstellen auflisten
- `DELETE /api/vulnerabilities/{id}`                  Schwachstelle soft-löschen
- `POST   /api/vulnerabilities/{id}/affected-ranges`  Betroffenen Range anlegen
- `DELETE /api/vulnerabilities/{id}/affected-ranges/{component}` Range entfernen

## BFF-Auth-Endpunkte

- `GET /api/auth/login`    Redirect zum OAuth2-Provider (Authorization Code)
- `GET /api/auth/callback` Callback: Code → Token, abgelegt in der Session
- `GET /api/auth/logout`   Session löschen
- `GET /api/auth/user`     Aktuelle Sitzung (`{authenticated, subject, name}`)
