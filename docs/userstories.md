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

## Epics / Themenbereiche

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

**Notizen:** Klären: hartes vs. soft-löschen. _

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
- [ ] Koordinaten mindestens: Komponenten-Bezeichner (z. B. `pkg:name`) und konkrete Version
- [ ] Komponente ist in der Projekt-Detailansicht sichtbar
- [ ] Mehrere Komponenten pro Projekt möglich

**Notizen:** Koordinatenformat offen – z. B. PURL (`pkg:gem/rails@7.0.0`). _

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
- [ ] Komponente aus Projekt entfernbar
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

---

## Offene Fragen

- [ ] Koordinatenformat für Komponenten festlegen (z. B. PURL `pkg:gem/rails@7.0.0`)?
- [ ] Syntax für Version Ranges festlegen (semver, maven, eigenes Format)?
- [ ] Schweregrad-Darstellung: CVSS-Score vs. Kategorien (Kritisch/Hoch/Mittel/Niedrig)?
- [ ] Ist Authentifizierung/Mehrnutzerbetrieb für das MVP nötig?
- [ ] Technologie-Stack und Hosting festlegen?
- [ ] Hartes oder soft-Löschen von Projekten/Komponenten/Schwachstellen?

---

## Änderungshistorie

| Datum       | Version | Änderung                                       |
|-------------|---------|------------------------------------------------|
| 2025-09-13  | 0.1     | Initiale Userstories-Liste                     |
| 2025-09-13  | 0.2     | Konkrete Epics/Stories für Schwachstellen-App  |
