# jansvca

Schwachstellen-Tracking-Anwendung. Erfaßt **Projekte** (mit Komponenten und
Koordinaten) und **Schwachstellen** (mit betroffenen Version-Ranges) und ermittelt,
welche Schwachstellen auf die eingesetzten Komponenten zutreffen.

## Architektur

- **Go**, hexagonale Module (Ports & Adapters)
- **Event Sourcing** mit selbstgebautem, WAL-basiertem Eventstore
- **Pro Modul ein eigener Eventstore** (Module: `projekte`, `schwachstellen`)
- **Kommunikation zwischen Modulen ausschließlich über Events** (Event Bus)
- **OAuth2**-Authentifizierung, erst eine Rolle `Administrator` mit allen Rechten
- **Soft-Delete** (gelöschte Entitäten werden nicht angezeigt)
- **Semantic Versioning** als Default; Koordinaten generisch, PURL unterstützt

Siehe `docs/architecture.md` für Details.

## Build & Test

```bash
go build ./...
go test ./...
```

## Docker

Multistage-Build (Go-Build → `FROM scratch`). Das fertige Image enthält nur
statisches Binary + CA-Zertifikate und ist distroless.

Das CI-Build veröffentlicht das Image in die **GitHub Container Registry**
(`ghcr.io/mwildt/jansvca`), nicht nach Docker Hub.

Lokal bauen:

```bash
docker build -t jansvca .
```

Image von GHCR ziehen (nach CI-Build auf `main`):

```bash
docker pull ghcr.io/mwildt/jansvca:latest
```

Ausführen — ohne OAuth2 (API läuft ohne Auth):

```bash
docker run --rm -p 8080:8080 -v jansvca-data:/data ghcr.io/mwildt/jansvca:latest
```

Ausführen — mit OAuth2-Introspection:

```bash
docker run --rm -p 8080:8080 -v jansvca-data:/data \
  -e JANSVCA_OAUTH2_INTROSPECTION_URL=https://provider/oauth2/introspect \
  -e JANSVCA_OAUTH2_CLIENT_ID=... \
  -e JANSVCA_OAUTH2_CLIENT_SECRET=... \
  ghcr.io/mwildt/jansvca:latest
```

## Ausführen

```bash
# Optionaler OAuth2-Introspection-Endpoint (ohne Setting läuft die API ohne Auth)
export JANSVCA_ADDR=:8080
export JANSVCA_DATA_DIR=./data
export JANSVCA_OAUTH2_INTROSPECTION_URL=https://provider/oauth2/introspect
export JANSVCA_OAUTH2_CLIENT_ID=...
export JANSVCA_OAUTH2_CLIENT_SECRET=...

go run ./cmd/jansvca
```

## REST-Übersicht

- `POST   /api/projects`                              Projekt anlegen
- `GET    /api/projects`                              Projekte auflisten
- `GET    /api/projects/{id}`                         Projektdetail
- `PATCH  /api/projects/{id}`                         Projekt umbenennen/beschreiben
- `DELETE /api/projects/{id}`                         Projekt soft-löschen
- `POST   /api/projects/{id}/components`              Komponente hinzufügen
- `DELETE /api/projects/{id}/components/{component}`  Komponente entfernen
- `GET    /api/projects/{id}/matches`                 Treffer für Projekt
- `POST   /api/vulnerabilities`                       Schwachstelle anlegen
- `GET    /api/vulnerabilities`                       Schwachstellen auflisten
- `DELETE /api/vulnerabilities/{id}`                  Schwachstelle soft-löschen
- `POST   /api/vulnerabilities/{id}/affected-ranges`  Betroffenen Range anlegen
- `DELETE /api/vulnerabilities/{id}/affected-ranges/{component}` Range entfernen
