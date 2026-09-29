import Constants from "expo-constants";

// Base URL of the cloud API. On the Android emulator, 10.0.2.2 points to the
// host machine running the emulator.
export const API_URL: string =
  (Constants.expoConfig?.extra?.apiUrl as string | undefined) ??
  "http://10.0.2.2:8080";

export type Health = { status: string };

export async function fetchHealth(): Promise<Health> {
  const res = await fetch(`${API_URL}/healthz`);
  if (!res.ok) {
    throw new Error(`API responded with status ${res.status}`);
  }
  return (await res.json()) as Health;
}
