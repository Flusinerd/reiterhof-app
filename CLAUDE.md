# Stallfunk — Anweisungen für KI-Agenten

- `backend/`: Go-Service, `net/http` ServeMux, kein Framework, Tests nur mit der Standardbibliothek.
- `mobile/`: Expo-/React-Native-App (TypeScript, Expo Router).
- Abhängigkeiten exakt pinnen (`"clsx": "2.1.1"`, nie `^` oder `~`). In `mobile/` bestimmt die Expo-SDK die Versionen von `react`, `react-native` usw. — nicht `npm outdated`, sondern `npx expo install --check`.
- Code, Bezeichner, Code-Kommentare und Commit-Betreffs sind englisch; nur die UI-Texte sind deutsch. Conventional Commits.
- `just lint` und `just test` müssen grün sein.
- Planung und Tickets: Linear-Projekt „Stallfunk“ (P-JAN-1).
- Linear immer mitziehen: Bei jeder Änderung das zugehörige Ticket aktualisieren (Status, Beschreibung, Kommentar). Neue Arbeit ohne Ticket bekommt ein neues Ticket im passenden Meilenstein; Tickets, die durch eine Änderung überholt sind, anpassen oder schließen.
