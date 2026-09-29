import Constants from "expo-constants";

// Basisadresse der Cloud-API. Auf dem Android-Emulator zeigt 10.0.2.2 auf den
// Rechner, auf dem der Emulator läuft.
export const API_URL: string =
  (Constants.expoConfig?.extra?.apiUrl as string | undefined) ??
  "http://10.0.2.2:8080";

export type Health = { status: string };

export async function fetchHealth(): Promise<Health> {
  const res = await fetch(`${API_URL}/healthz`);
  if (!res.ok) {
    throw new Error(`API antwortet mit Status ${res.status}`);
  }
  return (await res.json()) as Health;
}
