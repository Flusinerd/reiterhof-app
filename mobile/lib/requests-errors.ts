// German texts for the error codes of the requests API (see docs/domains/requests.md).
// Pure (no React Native imports) so it can be unit-tested.

import { ApiError, errorMessage } from "./api-core.ts";

export function requestErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case "request_full":
        return "Schon vergeben.";
      case "not_open":
        return "Schon erledigt oder abgesagt.";
      case "expired":
        return "Liegt in der Vergangenheit.";
      case "own_request":
        return "Das ist deine eigene Anfrage.";
      case "forbidden":
        return "Nicht erlaubt.";
      case "not_found":
        return "Diese Anfrage gibt es nicht mehr.";
      case "too_many_helpers":
        return "Schon mehr Helfer eingetragen. Sie müssen sich erst abmelden.";
      case "conflict":
        return "Für diesen Tag gibt es schon eine Anfrage der Serie.";
    }
  }
  return errorMessage(err);
}
