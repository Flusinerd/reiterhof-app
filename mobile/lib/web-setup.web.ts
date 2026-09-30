import { Alert, type AlertButton } from "react-native";

import { planAlert } from "./web-alert-core";

/**
 * Web-only setup, called once by the root layout.
 *
 * `Alert.alert` does nothing in react-native-web, and the app uses it for every confirmation
 * ("Dokument löschen?", "Konto löschen?", ...). Without a replacement the delete buttons would
 * be dead. It is routed to `window.confirm` / `window.alert`, which iOS Safari shows as native
 * dialogs, so the screens stay shared.
 *
 * Hook point for the service worker: the web push agent's `registerServiceWorker()` from
 * `lib/service-worker.web.ts` is meant to be called here once that module exists on main.
 */
export function setupWeb(): void {
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
