# Architekturdokumentation

## Überblick

**jansvca** ist eine Go-Anwendung zum Tracking von Schwachstellen. Sie ist als
hexagonale Module (Ports & Adapters) aufgebaut. Jedes Fachmodul kapselt seine
Domäne und kommuniziert mit anderen Modulen ausschließlich über **Events**.
Jedes Modul besitzt einen eigenen **Event-Sourcing-Store** (WAL, append-only),
den das Infrastruktur-Modul bereitstellt.

Zentrale Entitäten sind **Projekte** (mit Komponenten und Koordinaten) und
**Schwachstellen** (mit Version-Range-Zuordnungen). Das Matching ermittelt pro
Projekt, welche Schwachstellen auf die eingesetzten Komponenten zutreffen.

## Technische Entscheidungen

| Entscheidung          | Wert                                              |
|----------------------|---------------------------------------------------|
| Sprache/Runtime      | Go                                                |
| Architekturstil      | Hexagonale Module (Ports & Adapters)              |
| Kommunikation        | Event-basiert (asynchron über Event Bus)          |
| Persistenz           | Event Sourcing, pro Modul ein eigener Store       |
| Eventstore           | Selbstgebaut, Write-Ahead-Log (WAL), append-only  |
| Authentifizierung    | OAuth2 (Provider beliebig austauschbar)          |
| Autorisierung        | Erst eine Rolle `Administrator` mit allen Rechten |
| Löschen             | Soft-Delete, gelöschte Entitäten werden nicht angezeigt |
| Versionierung        | Semantic Versioning (semver) als Default          |
| Koordinatenformat    | Generisch (Bezeichner + Version), PURL unterstützt |

## Module (Bounded Contexts)

```
┌─────────────────────────────────────────────────────────┐
│                      Anwendungskern                      │
│                                                          │
│  ┌──────────────┐  Events   ┌────────────────────┐       │
│  │   Projekte    │ ───────► │   Event Bus          │      │
│  │  (hexagonal)  │ ◄─────── │   (Dispatcher)       │      │
│  └──────────────┘          └────────────────────┘       │
│         ▲                          ▲                    │
│         │ replay                   │ dispatch            │
│  ┌──────┴──────┐          ┌─────────┴──────────┐        │
│  │  Eventstore  │          │   Schwachstellen     │      │
│  │  (Projekte)  │          │   (hexagonal)       │       │
│  └─────────────┘          └────────────────────┘       │
│         ▲                          ▲                    │
│         │                          │ replay             │
│  ┌──────┴──────┐          ┌─────────┴──────────┐        │
│  │  Eventstore  │          │   Eventstore        │      │
│  │(Schwachstellen)│       │  (Schwachstellen)    │      │
│  └─────────────┘          └────────────────────┘       │
└─────────────────────────────────────────────────────────┘
                          │
              ┌───────────┴────────────┐
              │    Infrastruktur-Modul   │
              │  - WAL Eventstore        │
              │  - OAuth2 Auth           │
              │  - semver-Prüfung        │
              └─────────────────────────┘
```

### `projekte`

- **Verantwortung:** Projekte und deren Komponenten mit Koordinaten erfassen,
  sowohl einzeln als auch über einen CycloneDX-SBOM-Import.
- **Aggregate:** `Project` (mit Komponentenliste), `Component` (Koordinate + Version).
- **Events:** `ProjectCreated`, `ProjectRenamed`, `ProjectDescriptionChanged`,
  `ProjectDeleted`, `ComponentAdded`, `ComponentRemoved`,
  `ComponentVersionUpdated`.
- **Eigener Eventstore.**
- **SBOM-Import:** das `sbom`-Paket parst CycloneDX-JSON in eine Liste von
  Komponenten-Deklarationen; `ImportComponents` gleicht diese gegen den
  Projektzustand ab und emittiert `ComponentAdded`/`ComponentVersionUpdated`.

### `schwachstellen`

- **Verantwortung:** Schwachstellen erfassen und deren Version-Range-Zuordnungen
  zu Komponenten verwalten; Matching gegen die Komponenten der `projekte`-Module
  durchführen.
- **Aggregate:** `Vulnerability` (ID, CVSS-Score, Beschreibung, AffectedRanges),
  `AffectedRange` (Komponenten-Bezeichner + semver-Range).
- **Events:** `VulnerabilityCreated`, `VulnerabilityUpdated`,
  `VulnerabilityDeleted`, `AffectedRangeAdded`, `AffectedRangeRemoved`.
  (Die Event-Typen werden nur noch zur einmaligen Migration eines bestehenden
  WAL in den neuen Datei-Store genutzt; das Aggregat ist jetzt zustandsbasiert.)
- **Persistenz (Datei-Store + Bleve):** Schwachstellen werden als eine
  JSON-Datei pro Record unter `<dataDir>/vulnerabilities/<id>.json`
  gespeichert. Ein [Bleve](https://github.com/blevesearch/bleve)-Index unter
  `<dataDir>/vuln-index/` übernimmt Volltextsuche, Filter (Ecosystem, Quelle,
  CVSS-Min) und Pagination. Ein sekundärer Komponenten-Index
  (`component -> vulnerability-IDs`) im RAM ermöglicht Matching ohne
  Vollscan. Der Datei-Store ersetzt den früheren WAL für Schwachstellen.
- **Konsumierte Events:** vom `projekte`-Modul veröffentlichte Komponenten-Events
  werden von der `ProjectProjection` gelesen; die Komponenten-Map
  (`ProjectProjection.Components`) ist der Read-Port für das Matching.
- **OSV-Import:** das `osv`-Paket lädt Schwachstellen von [osv.dev](https://osv.dev)
  (GCS-Export `gs://osv-vulnerabilities`): initial die komplette Datenbank als
  `all.zip`, danach inkrementell über `modified_id.csv` (nur Einträge neuer als der
  letzte Sync). Der `osv/sync`-Service mappt OSV-Records auf das interne Modell
  und upsertet sie über `CommandHandler.Import`. Der Sync läuft beim Start einmal
  initial und dann im konfigurierten Intervall (Default 60 Minuten) als
  Hintergrund-Goroutine.
- **Herkunft & Persistenz:** OSV-Importe werden mit `source="osv"` markiert
  (manuelle API-Einträge erhalten `source="manual"`), das Feld ist im Read-Model
  (`VulnerabilityView.Source`) sichtbar. Der Sync-Fortschritt (höchster gesehener
  `modified`-Timestamp) wird in `osv-sync.json` unter `JANSVCA_DATA_DIR`
  persistiert, sodass nach einem Neustart inkrementell weitergemacht wird statt
  `all.zip` neu zu laden.
- **Migration:** existiert beim Start eine Legacy-`schwachstellen.wal`, wird sie
  einmalig in den Datei-Store migriert und danach zu `schwachstellen.wal.migrated`
  umbenannt.

### `infrastruktur` (Querschnitt)

- **Eventstore-Implementierung** (WAL, append-only) – genutzt vom
  `projekte`-Modul (eigenem Stream/Store); das `schwachstellen`-Modul nutzt
  stattdessen den Datei-Store + Bleve.
- **Event Bus / Dispatcher** – asynchrone Verteilung veröffentlichter Events an
  interessierte Module.
- **OAuth2-Authentifizierung** + einfache `Administrator`-Rolle.
- **semver-Prüfung** (Version-Range-Matching) als wiederverwendbare Komponente.

## Hexagonale Struktur pro Fachmodul

Jedes Fachmodul folgt dem Ports & Adapters-Pattern:

```
modul/<name>/
├── domain/            # Entitäten, Aggregate, Domain-Events (rein, keine I/O)
│   ├── event.go       #   Event-Typen dieses Moduls
│   └── aggregate.go   #   Aggregate + Geschäftsregeln + Event-Applikation
├── application/       # Use Cases (Commands), orchestriert Ports
│   ├── command.go     #   Command-Handler
│   └── port.go        #   Ports (Interfaces): EventStore, EventBus, Repositories
└── adapter/
    ├── eventstore/    # Adapter: nutzt Infrastruktur-WAL-Eventstore
    ├── eventbus/      # Adapter: Publish/Subscribe über globalen Bus
    └── api/           # Adapter: HTTP/REST-Endpoints (Driving Adapter)
```

- **Domain** hat keine Abhängigkeiten nach außen (reine Geschäftslogik).
- **Application** definiert Ports (Interfaces) und Use-Case-Handler.
- **Adapter** implementieren die Ports gegen Infrastruktur (Eventstore, Bus) und
  nach außen (HTTP-API).

## Event Sourcing

- Zustand jedes Aggregats wird ausschließlich aus seinem Event-Stream
  rekonstruiert (`replay`).
- Commands werden gegen den rekonstruierten Zustand validiert und erzeugen neue
- Events.
- Events werden im Modul-Eventstore append-only persisted (WAL) **bevor** sie
- veröffentlicht werden (Outbox-Charakter: Persistenz vor Verteilung).
- Read Models werden aus Events projiziert (z. B. Projektliste, Matching-Ergebnis).

### WAL-Eventstore

- Jeder Store ist ein append-only Log (Write-Ahead-Log).
- Eintrag: `StreamID | EventID | EventType | Payload (JSON) | Timestamp | Version`
- Leseoperation: `Load(streamID) []Event` lädt alle Events eines Streams in
  Reihenfolge.
- Schreiboperation: `Append(streamID, events)` hängt an; Version/Sequenz
  schützt vor verlorenen Updates (optimistic concurrency).
- Storage-Backend initial dateibasiert (eine Datei pro Store), austauschbar.

## Kommunikation über Events

- Module rufen einander **nicht** direkt auf.
- Ein Modul veröffentlicht seine Events über den Event Bus; andere Module
  abonnieren relevante Events und pfifen ihre Read Models nach.
- Beispiel: `projekte` veröffentlicht `ComponentAdded` → `schwachstellen`
  aktualisiert sein Komponenten-Read-Model → Matching liefert neue Treffer.

```
projekte (ComponentAdded) ──► EventBus ──► schwachstellen (Projektor: Komponenten-Read-Model)
                                                  │
                                                  └──► Matching (semver-Range vs. Version)
```

## Sicherheitsmodell

- Alle Endpoints erfordern OAuth2-Authentifizierung (Bearer-Token).
- Erst eine Rolle `Administrator` mit allen Rechten; jeder authentifizierte
  Nutzer hat diese Rolle.
- RBAC ist so modelliert, dass später weitere Rollen/Rechte ergänzt werden
  können.

## Build & Ausführung

```bash
# Abhängigkeiten laden
go mod download

# Build
go build ./...

# Tests
go test ./...

# Anwendung starten (Beispiel)
go run ./cmd/jansvca
```
