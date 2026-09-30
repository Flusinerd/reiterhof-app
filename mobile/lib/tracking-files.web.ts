/**
 * Web variant of `lib/tracking-files.ts`: one IndexedDB object store with the text of each
 * "file". IndexedDB instead of localStorage because a ride snapshot is hundreds of kilobytes
 * and localStorage holds only about 5 MB per origin (and the rest of the app uses it too).
 * Safari can evict site data of a website that was not used for seven days; an installed PWA
 * is exempt from that, which is another reason to install it (docs/domains/pwa.md).
 */

const DB_NAME = "stallfunk-tracking";
const STORE = "files";

let opened: Promise<IDBDatabase> | null = null;

function open(): Promise<IDBDatabase> {
  opened ??= new Promise<IDBDatabase>((resolve, reject) => {
    const request = window.indexedDB.open(DB_NAME, 1);
    request.onupgradeneeded = () => request.result.createObjectStore(STORE);
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => {
      opened = null;
      reject(request.error);
    };
  });
  return opened;
}

async function run<T>(mode: IDBTransactionMode, job: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await open();
  return new Promise<T>((resolve, reject) => {
    const tx = db.transaction(STORE, mode);
    const request = job(tx.objectStore(STORE));
    tx.oncomplete = () => resolve(request.result);
    tx.onerror = () => reject(tx.error);
    tx.onabort = () => reject(tx.error);
  });
}

export async function writeFile(name: string, text: string): Promise<void> {
  await run("readwrite", (store) => store.put(text, name));
}

export async function readFile(name: string): Promise<string | null> {
  const value = await run<unknown>("readonly", (store) => store.get(name));
  return typeof value === "string" ? value : null;
}

export async function removeFile(name: string): Promise<void> {
  await run("readwrite", (store) => store.delete(name));
}
