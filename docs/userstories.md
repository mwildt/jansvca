# Userstories

Dieses Dokument dient der Sammlung und Strukturierung der Anforderungen an das Projekt **jansvca**. Jede User Story folgt der Form:

> Als `<Rolle>` möchte ich `<Funktionalität>`, damit `<Nutzen>`.

Priorisierung:

- `MUST` – zwingend erforderlich für das MVP
- `SHOULD` – wichtig, aber nicht MVP-kritisch
- `COULD` – nice-to-have, optional
- `WONT` – bewusst zurückgestellt

Status: `Offen` · `In Arbeit` · `Umgesetzt` · `Verifiziert`

Rollen:

- `Nutzer` – Bedient die App zur Erfassung und Auswertung von Schwachstellen
- `Administrator` – Verwaltet Zugriff und globale Stammdaten (später)

---

## Projektübersicht

- **Projekt:** jansvca
- **Zweck:** App zum Tracking von Schwachstellen. Zentrale Entitäten sind **Projekte** und **Schwachstellen**. Beide können vom Nutzer hinzugefügt werden. Projekte bestehen aus **Komponenten** mit **Koordinaten** (z. B. Bezeichner + Version). Schwachstellen können **Komponenten in bestimmten Version Ranges** betreffen. Ziel ist es, pro Projekt zu erkennen, welche Schwachstellen auf die eingesetzten Komponenten zutreffen.
- **Definition of Done (DoD):** Funktional umgesetzt, gegen Akzeptanzkriterien geprüft, dokumentiert, ohne offene Fehler.

---

## Technische Rahmenentscheidungen

- **Sprache/Runtime:** Go
- **Architektur:** Event Sourcing – Zustand wird aus Ereignissen (Events) rekonstruiert; Commands erzeugen Events, Read Models werden aus den Events projiziert
- **Module:** Zwei Fachmodule + ein Infrastrukturmodul:
  - `Projekte` – Projekte und deren Komponenten/Koordinaten
  - `Schwachstellen` – Schwachstellen und deren Version-Range-Zuordnungen inkl. Matching
  - `Infrastruktur` – Querschnitt: selbstgebauter Eventstore (WAL), Authentifizierung/Autorisierung, semver-Prüfung
- **Eventstores:** Pro Fachmodul ein eigener Eventstore; der Eventstore wird selbst implementiert (Write-Ahead-Log, append-only) und im Infrastruktur-Modul bereitgestellt
- **Authentifizierung:** OAuth2
- **Autorisierung:** Erst eine Rolle (`Administrator`) mit allen Rechten; feingranulares RBAC später
- **Löschen:** Soft-Delete – Entitäten werden als gelöscht markiert, nicht physisch entfernt
- **Versionierung:** Semantic Versioning (semver) als Default – Komponenten-Versionen und Version Ranges werden nach semver interpretiert
- **Version-Range-Syntax:** semver-kompatibel (z. B. `>=1.0.0 <2.0.0`, `^1.2.0`)
- **Koordinatenformat:** Generisch (Bezeichner + Version), zusätzlich PURL (`pkg:gem/rails@7.0.0`) als unterstütztes Format

---

## Epics / Themenbereiche

- `EP-00` – Infrastruktur (selbstgebauter WAL-Eventstore, OAuth2-Auth, Rollen/Rechte, semver-Prüfung)
- `EP-01` – Projektverwaltung (Projekte anlegen, anzeigen, bearbeiten)
- `EP-02` – Komponenten & Koordinaten (Komponenten mit Version zu Projekten erfassen)
- `EP-03` – Schwachstellenverwaltung (Schwachstellen anlegen, erfassen, was sie betreffen)
- `EP-04` – Version Ranges & Matching (Schwachstellen Komponenten in Versionsbereichen zuordnen, Treffer pro Projekt ermitteln)

---

## User Stories

### US-001 – Projekt anlegen

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-01`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich ein neues Projekt mit Name und Beschreibung anlegen, damit ich Schwachstellen und Komponenten einem konkreten Projekt zuordnen kann.

**Akzeptanzkriterien:**

- [ ] Formular zum Anlegen eines Projekts (mind. Name, optional Beschreibung)
- [ ] Projektname ist verpflichtend und eindeutig
- [ ] Projekt erscheint nach dem Speichern in der Projektliste
- [ ] Validierung bei fehlendem/leerem Namen

**Notizen:** _

---

### US-002 – Projekte auflisten und anzeigen

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-01`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich alle Projekte in einer Liste sehen und ein einzelnes Projekt im Detail aufrufen, damit ich den Status und die zugehörigen Komponenten überblicken kann.

**Akzeptanzkriterien:**

- [ ] Listenansicht aller Projekte
- [ ] Detailansicht je Projekt mit Name, Beschreibung und zugehörigen Komponenten
- [ ] Komponenten sind in der Projekt-Detailansicht sichtbar

**Notizen:** _

---

### US-003 – Projekt bearbeiten und löschen

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-01`                                         |
| Priorität    | SHOULD                                          |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich ein Projekt bearbeiten oder löschen, damit veraltete oder fehlerhafte Projekteinträge korrigiert werden können.

**Akzeptanzkriterien:**

- [ ] Bearbeiten von Name und Beschreibung möglich
- [ ] Löschen mit Bestätigung
- [ ] Löschen entfernt verwaiste Zuordnungen sauber

**Notizen:** Löschen als Soft-Delete (Entität wird als gelöscht markiert). _

---

### US-004 – Komponente mit Koordinaten zu Projekt hinzufügen

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-02`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich einem Projekt eine Komponente mit Koordinaten (Bezeichner + konkrete Version) hinzufügen, damit die eingesetzten Bausteine des Projekts erfasst sind.

**Akzeptanzkriterien:**

- [ ] Komponente kann innerhalb eines Projekts angelegt werden
- [ ] Koordinaten: generisches Format (Bezeichner + konkrete Version); PURL (`pkg:gem/rails@7.0.0`) wird als Format unterstützt
- [ ] Komponente ist in der Projekt-Detailansicht sichtbar
- [ ] Mehrere Komponenten pro Projekt möglich

**Notizen:** Koordinatenformat generisch, PURL-Unterstützung. _

---

### US-005 – Komponenten bearbeiten und entfernen

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-02`                                         |
| Priorität    | SHOULD                                          |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich Komponenten bearbeiten (z. B. Version aktualisieren) oder entfernen, damit der Projektbestand aktuell bleibt.

**Akzeptanzkriterien:**

- [ ] Version/Bezeichner einer Komponente editierbar
- [ ] Komponente aus Projekt entfernbar (Soft-Delete)
- [ ] Änderungen in Projekt-Detailansicht sichtbar

**Notizen:** _

---

### US-006 – Schwachstelle anlegen

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-03`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich eine Schwachstelle mit ID, Beschreibung und Schweregrad anlegen, damit bekannte Schwachstellen im System erfasst sind.

**Akzeptanzkriterien:**

- [ ] Formular zum Anlegen einer Schwachstelle (ID z. B. CVE, Titel, Beschreibung, Schweregrad)
- [ ] Schwachstellen-ID ist verpflichtend
- [ ] Schwachstelle erscheint nach Speichern in der Schwachstellen-Liste

**Notizen:** Schweregrad z. B. CVSS-Score oder Kritisch/Hoch/Mittel/Niedrig. _

---

### US-007 – Schwachstellen auflisten und anzeigen

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-03`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich alle Schwachstellen listen und im Detail aufrufen, damit ich den erfassten Bestand überblicken kann.

**Akzeptanzkriterien:**

- [ ] Listenansicht aller Schwachstellen
- [ ] Detailansicht mit ID, Beschreibung, Schweregrad und betroffenen Komponenten/Version Ranges

**Notizen:** _

---

### US-008 – Schwachstelle Komponente in Version Range zuordnen

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-04`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich einer Schwachstelle angeben, welche Komponente sie betrifft und in welchem **Version Range** (z. B. `>=1.0,<2.5`), damit später Treffer für eingesetzte Versionen ermittelt werden können.

**Akzeptanzkriterien:**

- [ ] Schwachstelle kann mehreren Komponenten-Bezeichnern mit je einem Version Range zugeordnet werden
- [ ] Version Range ist verpflichtend pro Zuordnung
- [ ] Zuordnungen in der Schwachstellen-Detailansicht sichtbar und editierbar

**Notizen:** Range-Syntax festlegen – z. B. semver/maven-stil. _

---

### US-009 – Treffer pro Projekt ermitteln (Matching)

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-04`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich für ein Projekt sehen, welche Schwachstellen auf die eingesetzten Komponenten (anhand Koordinaten + Version vs. Version Range) zutreffen, damit ich gezielt reagieren kann.

**Akzeptanzkriterien:**

- [ ] Für jede Projektkomponente wird geprüft, ob eine Schwachstelle mit passendem Bezeichner und treffendem Version Range existiert
- [ ] Trefferliste pro Projekt (Komponente → betroffene Schwachstellen)
- [ ] Anzeige von Schweregrad je Treffer
- [ ] Aktualisierung der Treffer bei Änderung an Komponente oder Schwachstelle

**Notizen:** Kernfeature der App. _

---

### US-010 – Schwachstellen nach Projekt filtern/auswerten

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-04`                                         |
| Priorität    | SHOULD                                          |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich Schwachstellen pro Projekt gefiltert nach Schweregrad sehen, damit ich kritische Treffer priorisieren kann.

**Akzeptanzkriterien:**

- [ ] Filter nach Schweregrad in der Trefferansicht
- [ ] Sortierung nach Schweregrad

**Notizen:** _

---

### US-011 – Authentifizierung

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-00`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Nutzer` möchte ich mich authentifizieren (Login), damit nur berechtigte Personen auf die App und deren Daten zugreifen können.

**Akzeptanzkriterien:**

- [ ] Login erforderlich vor Zugriff auf beliebige Funktionalität
- [ ] Authentifizierung erfolgt über OAuth2 (Authorization-Server/Provider)
- [ ] Ungültige Anmeldedaten/Token werden abgewiesen
- [ ] Token-basierter Zugriff auf alle Modul-APIs

**Notizen:** OAuth2-Provider festlegen. _

---

### US-012 – Rollen und Rechte (RBAC)

| Feld         | Wert                                            |
--------------|-------------------------------------------------|
| Epic         | `EP-00`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Entwickler` möchte ich erst nur eine Rolle (`Administrator`) mit allen Rechten vorsehen, damit der Erststart ohne komplexes Rechtemanagement auskommt, das Modell aber später erweiterbar bleibt.

**Akzeptanzkriterien:**

- [ ] Eine Rolle `Administrator` mit allen Rechten
- [ ] Jeder authentifizierte Nutzer hat die Administrator-Rolle
- [ ] Aktionen ohne Authentifizierung werden abgewiesen
- [ ] RBAC ist so modelliert, dass später weitere Rollen/Rechte ergänzt werden können

**Notizen:** Feingranulares RBAC später. _

---

### US-013 – Event-Sourcing-Grundgerüst pro Modul

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-00`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Entwickler` möchte ich einen selbstgebauten Eventstore mit Write-Ahead-Log (WAL) im Infrastruktur-Modul, damit die beiden Fachmodule `Projekte` und `Schwachstellen` ihre Events unabhängig persistieren und projizieren können.

**Akzeptanzkriterien:**

- [ ] Eventstore-Abstraktion im Infrastruktur-Modul implementiert (append-only, WAL, Event-Reihenfolge, IDs)
- [ ] Pro Fachmodul ein eigener Eventstore (Projekte, Schwachstellen)
- [ ] Aggregate rekonstruieren Zustand aus Events (Replay)
- [ ] Commands validieren und erzeugen Events
- [ ] Read Models werden aus Events projiziert

**Notizen:** WAL selbst implementiert, keine externe ES-Library. _

---

### US-014 – semver-Vergleich und Version-Range-Prüfung

| Feld         | Wert                                            |
|--------------|-------------------------------------------------|
| Epic         | `EP-04`                                         |
| Priorität    | MUST                                            |
| Status       | Offen                                           |
| Aufwand      | _TBD_                                           |

**Story:** Als `Entwickler` möchte ich eine semver-basierte Prüfung, ob eine konkrete Version in einem Version Range liegt, damit das Matching verlässlich arbeitet.

**Akzeptanzkriterien:**

- [ ] Komponenten-Versionen werden nach semver interpretiert
- [ ] Version Ranges nach semver-Syntax (z. B. `>=1.0.0 <2.0.0`, `^1.2.0`)
- [ ] Funktion prüft: liegt Version X im Range R?
- [ ] Ungültige Versionen/Ranges werden erkannt

**Notizen:** Basis für US-009. _

---

## Prioritäten-Matrix

| ID      | Titel                                              | Epic    | Priorität | Status |
|---------|----------------------------------------------------|---------|-----------|--------|
| US-001  | Projekt anlegen                                    | EP-01   | MUST      | Offen  |
| US-002  | Projekte auflisten und anzeigen                     | EP-01   | MUST      | Offen  |
| US-003  | Projekt bearbeiten und löschen                     | EP-01   | SHOULD    | Offen  |
| US-004  | Komponente mit Koordinaten zu Projekt hinzufügen   | EP-02   | MUST      | Offen  |
| US-005  | Komponenten bearbeiten und entfernen               | EP-02   | SHOULD    | Offen  |
| US-006  | Schwachstelle anlegen                              | EP-03   | MUST      | Offen  |
| US-007  | Schwachstellen auflisten und anzeigen              | EP-03   | MUST      | Offen  |
| US-008  | Schwachstelle Komponente in Version Range zuordnen | EP-04   | MUST      | Offen  |
| US-009  | Treffer pro Projekt ermitteln (Matching)           | EP-04   | MUST      | Offen  |
| US-010  | Schwachstellen nach Projekt filtern/auswerten      | EP-04   | SHOULD    | Offen  |
| US-011  | Authentifizierung                                  | EP-00   | MUST      | Offen  |
| US-012  | Rollen und Rechte (RBAC)                          | EP-00   | MUST      | Offen  |
| US-013  | Event-Sourcing-Grundgerüst pro Modul              | EP-00   | MUST      | Offen  |
| US-014  | semver-Vergleich und Version-Range-Prüfung        | EP-04   | MUST      | Offen  |

---

## Offene Fragen

- [x] Koordinatenformat festgelegt: generisch (Bezeichner + Version), PURL unterstützt
- [x] Auth: OAuth2
- [x] Rollen: erst eine `Administrator`-Rolle mit allen Rechten
- [x] Eventstore: selbstgebaut mit WAL im Infrastruktur-Modul
- [x] Module: `Projekte`, `Schwachstellen` (+ `Infrastruktur` als Querschnitt)
- [x] Löschen: Soft-Delete
- [ ] OAuth2-Provider konkret festlegen
- [ ] Soft-Delete: Anzeige/Filterung gelöschter Entitäten in UI/API festlegen
- [ ] Schweregrad-Darstellung: CVSS-Score vs. Kategorien (Kritisch/Hoch/Mittel/Niedrig)

---

## Änderungshistorie

| Datum       | Version | Änderung                                       |
|-------------|---------|------------------------------------------------|
| 2025-09-13  | 0.1     | Initiale Userstories-Liste                     |
| 2025-09-13  | 0.3     | Stack: Go + Event Sourcing, Auth/Rollen, semver  |
| 2025-09-13  | 0.4     | Koordinaten generisch+PURL, OAuth2, eine Rolle, Soft-Delete, WAL-Eventstore, 2 Module  |
