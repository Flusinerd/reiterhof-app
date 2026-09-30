import * as Device from "expo-device";
import * as Notifications from "expo-notifications";
import { Platform } from "react-native";

export type PushRegistration = {
  /** Native device token: the APNs token (hex) on iOS, the FCM registration token on Android. */
  token: string;
  platform: "ios" | "android";
};

export type PushResult =
  | { ok: true; registration: PushRegistration }
  | { ok: false; reason: "not_a_device" | "permission_denied" | "error" };

// Android requires a channel before any notification is shown. The backend names it in
// every FCM message (`channelId`), so keep the id in sync with internal/push/fcm.go.
async function ensureAndroidChannel(): Promise<void> {
  if (Platform.OS !== "android") return;
  await Notifications.setNotificationChannelAsync("default", {
    name: "Erinnerungen",
    importance: Notifications.AndroidImportance.DEFAULT,
  });
}

/**
 * Asks for notification permission (if not yet decided) and returns the native device
 * token. The backend sends to APNs and FCM directly (JAN-88), so no Expo project is
 * involved; on Android this needs the Firebase config (`google-services.json`, see
 * README), without it the token call throws and the result is `error`.
 * `useDeviceSetup` sends the token to `POST /me/push-tokens`. Never throws.
 */
export async function registerForPush(): Promise<PushResult> {
  if (!Device.isDevice) return { ok: false, reason: "not_a_device" };
  if (Platform.OS !== "ios" && Platform.OS !== "android") {
    return { ok: false, reason: "not_a_device" };
  }
  try {
    await ensureAndroidChannel();
    const existing = await Notifications.getPermissionsAsync();
    let status = existing.status;
    if (status !== "granted") {
      status = (await Notifications.requestPermissionsAsync()).status;
    }
    if (status !== "granted") return { ok: false, reason: "permission_denied" };

    const { data } = await Notifications.getDevicePushTokenAsync();
    if (typeof data !== "string" || data === "") return { ok: false, reason: "error" };
    return { ok: true, registration: { token: data, platform: Platform.OS } };
  } catch {
    return { ok: false, reason: "error" };
  }
}
