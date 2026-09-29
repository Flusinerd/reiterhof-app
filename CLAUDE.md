# Reiterhof — Anweisungen für KI-Agenten

- `backend/`: Go-Service, `net/http` ServeMux, kein Framework, Tests nur mit der Standardbibliothek.
- `mobile/`: Expo-/React-Native-App (TypeScript, Expo Router).
- Abhängigkeiten exakt pinnen (`"clsx": "2.1.1"`, nie `^` oder `~`). In `mobile/` bestimmt die Expo-SDK die Versionen von `react`, `react-native` usw. — nicht `npm outdated`, sondern `npx expo install --check`.
- Code, Bezeichner, Code-Kommentare und Commit-Betreffs sind englisch; nur die UI-Texte sind deutsch. Conventional Commits.
- `just lint` und `just test` müssen grün sein.
- Planung und Tickets: Linear-Projekt „Reiterhof App“ (P-JAN-1).
