import { Alert, type AlertButton } from "react-native";

import { registerServiceWorker } from "./service-worker";
import { planAlert } from "./web-alert-core";

/**
 * Web-only setup, called once by the root layout.
 *
 * `Alert.alert` does nothing in react-native-web, and the app uses it for every confirmation
 * ("Dokument löschen?", "Konto löschen?", ...). Without a replacement the delete buttons would
 * be dead. It is routed to `window.confirm` / `window.alert`, which iOS Safari shows as native
 * dialogs, so the screens stay shared.
 *
 * It also registers the service worker (`public/sw.js`) at start: Chromium only offers
 * installation with a registered worker, and web push reuses the same registration.
 */
export function setupWeb(): void {
  void registerServiceWorker();
  Alert.alert = (title: string, message?: string, buttons?: AlertButton[]) => {
    const plan = planAlert(title, message, buttons);
    if (plan.kind === "notice") {
      window.alert(plan.text);
      plan.then?.();
    } else if (window.confirm(plan.text)) {
      plan.confirm();
    } else {
      plan.cancel?.();
    }
  };
}
