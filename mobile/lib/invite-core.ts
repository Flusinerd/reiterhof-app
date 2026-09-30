// Pure helpers for inviting people to the stable (unit-tested).

export type Invite = { code: string; expires_at: string; max_uses: number };

/** "07.10." in the stable's time zone. */
export function inviteExpiry(expiresAt: string, timeZone: string): string {
  return new Intl.DateTimeFormat("de-DE", { day: "2-digit", month: "2-digit", timeZone }).format(new Date(expiresAt));
}

/** The text of the share sheet: where to go, the code, how long it is valid. */
export function inviteMessage(invite: Invite, stableName: string, webUrl: string, timeZone: string): string {
  return [
    `Komm in den Stall „${stableName}“ bei Stallfunk:`,
    `1. ${webUrl} öffnen und mit deiner E-Mail anmelden`,
    `2. Einladungscode eingeben: ${invite.code}`,
    `Gültig bis ${inviteExpiry(invite.expires_at, timeZone)}`,
  ].join("\n");
}
