// Decides how an `Alert.alert(title, message, buttons)` call is shown in a browser, where
// react-native-web's Alert does nothing. Pure and unit-tested; lib/web-setup.web.ts applies it.

export type AlertButtonLike = {
  text?: string;
  style?: "default" | "cancel" | "destructive";
  onPress?: (value?: never) => void;
};

export type AlertPlan =
  /** `window.alert(text)`, then `then` (the single button's handler) runs. */
  | { kind: "notice"; text: string; then: (() => void) | undefined }
  /** `window.confirm(text)`; OK runs `confirm`, Cancel runs `cancel`. */
  | { kind: "confirm"; text: string; confirm: () => void; cancel: (() => void) | undefined };

const handler = (b: AlertButtonLike | undefined) => (b?.onPress ? () => b.onPress?.() : undefined);

export function planAlert(title: string, message?: string, buttons?: AlertButtonLike[]): AlertPlan {
  const text = message ? `${title}\n\n${message}` : title;
  const list = buttons ?? [];
  const cancel = list.find((b) => b.style === "cancel");
  const actions = list.filter((b) => b !== cancel);
  if (actions.length === 0) return { kind: "notice", text, then: handler(cancel) };
  // A dialog has two outcomes, so several action buttons collapse to one (the destructive one
  // preferred); the alerts of the app have one action plus "Abbrechen".
  const main = actions.find((b) => b.style === "destructive") ?? actions[0];
  if (!cancel && actions.length === 1) return { kind: "notice", text, then: handler(main) };
  return { kind: "confirm", text, confirm: () => main?.onPress?.(), cancel: handler(cancel) };
}
