# Budget Projection

Eine Web-Anwendung zur Budgetplanung und finanziellen Zukunftsprojektion.

## Features

- **Personen verwalten** – Anlegen und Verwalten mehrerer Personen
- **Konten verwalten** – Konten erstellen, die einer oder mehreren Personen gehören können
- **Positionen verwalten** – Wiederkehrende Einnahmen, Ausgaben und Überweisungen mit flexibler Frequenzsteuerung:
  - Täglich, wöchentlich, zweiwöchentlich, monatlich, quartalsweise, halbjährlich, jährlich
  - Frei wählbarer Tag im Monat
  - Geschäftstag-Regeln (exakt, letzter Geschäftstag davor, erster Geschäftstag danach, letzter Geschäftstag des Monats)

## Tech Stack

- **Backend:** Go (Gin Framework, GORM, SQLite)
- **Frontend:** React + TypeScript (Vite)
- **Deployment:** Docker (Single Container)

## Entwicklung

### Voraussetzungen

- Go 1.24+
- Node.js 20+
- npm

### Backend starten

```bash
cd backend
go mod tidy
go run .
```

### Frontend starten (Entwicklung)

```bash
cd frontend
npm install
npm run dev
```

Der Vite Dev-Server proxied `/api` Anfragen automatisch an `http://localhost:8080`.

## Docker

### Image bauen

```bash
docker build -t budget-projection .
```

### Container starten

```bash
docker run -d -p 8080:8080 -v budget-data:/app/data budget-projection
```

Die Anwendung ist dann unter `http://localhost:8080` erreichbar.

## API-Endpunkte

| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| GET | `/api/persons` | Alle Personen auflisten |
| POST | `/api/persons` | Person erstellen |
| GET | `/api/persons/:id` | Person abrufen |
| PUT | `/api/persons/:id` | Person aktualisieren |
| DELETE | `/api/persons/:id` | Person löschen |
| GET | `/api/accounts` | Alle Konten auflisten |
| POST | `/api/accounts` | Konto erstellen |
| GET | `/api/accounts/:id` | Konto abrufen |
| PUT | `/api/accounts/:id` | Konto aktualisieren |
| DELETE | `/api/accounts/:id` | Konto löschen |
| POST | `/api/accounts/:id/owners` | Eigentümer hinzufügen |
| DELETE | `/api/accounts/:id/owners/:personId` | Eigentümer entfernen |
| GET | `/api/positions` | Alle Positionen auflisten |
| POST | `/api/positions` | Position erstellen |
| GET | `/api/positions/:id` | Position abrufen |
| PUT | `/api/positions/:id` | Position aktualisieren |
| DELETE | `/api/positions/:id` | Position löschen |
