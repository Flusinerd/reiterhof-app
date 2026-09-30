/**
 * Registers the service worker `/sw.js` (Web Push today; the PWA shell may add caching to it
 * later in a separate file it imports with `importScripts`). Safe to call repeatedly: the browser
 * returns the existing registration. Resolves to null where service workers do not exist
 * (or in a non-secure context); never throws.
 *
 * Call this from anywhere in the web app instead of `navigator.serviceWorker.register` so there
 * is only one registration. The push flow (`lib/webpush.ts`) calls it before subscribing.
 */
export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  if (typeof navigator === "undefined" || !("serviceWorker" in navigator)) return null;
  try {
    return await navigator.serviceWorker.register("/sw.js", { scope: "/" });
  } catch {
    return null;
  }
}
