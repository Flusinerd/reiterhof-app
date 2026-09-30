/** Native stub: there is no service worker outside the web build (see `service-worker.web.ts`). */
export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
  return null;
}
