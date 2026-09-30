// German texts for the error codes of the requests API (see docs/domains/requests.md).
// Pure (no React Native imports) so it can be unit-tested.

import { ApiError, errorMessage } from "./api-core.ts";

export function requestErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case "request_full":
        return "Leider schon vergeben: Alle Plätze sind belegt.";
      case "not_open":
        return "Diese Anfrage ist bereits erledigt oder abgesagt.";
      case "expired":
        return "Diese Anfrage liegt in der Vergangenheit.";
      case "own_request":
        return "Du kannst deine eigene Anfrage nicht annehmen.";
      case "forbidden":
        return "Das darfst du bei dieser Anfrage nicht.";
      case "not_found":
        return "Diese Anfrage gibt es nicht mehr.";
      case "too_many_helpers":
        return "Es haben sich schon mehr Helfer eingetragen. Bitte sie zuerst, sich abzumelden.";
      case "conflict":
        return "An diesem Tag gibt es schon eine Anfrage dieser Serie.";
    }
  }
  return errorMessage(err);
}
