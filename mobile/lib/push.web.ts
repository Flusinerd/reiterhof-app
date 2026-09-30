// Web build: Expo push tokens do not exist in the browser and expo-notifications stays out of the
// web bundle. The PWA uses Web Push instead (lib/webpush.ts, `enableWebPush` / `syncWebPush`).
export type { PushRegistration, PushResult } from "./push.ts";

/** Always "not_a_device" on the web; `useDeviceSetup` takes the Web Push path there. */
export async function registerForPush(): Promise<import("./push.ts").PushResult> {
  return { ok: false, reason: "not_a_device" };
}
