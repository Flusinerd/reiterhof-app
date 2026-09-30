import { authed } from "../api";
import type { ConsentItem, ConsentKind, ConsentList } from "../consent-core.ts";
import { LEGAL_TEXT_VERSION } from "../consent-legal-texts.ts";

// Mirrors docs/domains/privacy.md.

export const CONSENTS_KEY = ["consents"] as const;

export const privacyApi = {
  consents: () => authed.get<ConsentList>("/api/v1/me/consents"),
  /** Granting sends the text version the app shows, the server refuses a different one (409). */
  setConsent: (kind: ConsentKind, granted: boolean) =>
    authed.put<ConsentItem>(
      `/api/v1/me/consents/${kind}`,
      granted ? { granted: true, version: LEGAL_TEXT_VERSION } : { granted: false },
    ),
  /** All personal data of the user as a JSON document. */
  exportData: () => authed.get<Record<string, unknown>>("/api/v1/me/export"),
  /** Anonymises the account. 409 `owns_horses` / `last_admin` when it is not allowed yet. */
  deleteAccount: () => authed.post<void>("/api/v1/me/delete", { confirm: true }),
};
