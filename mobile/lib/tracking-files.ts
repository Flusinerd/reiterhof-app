import * as FileSystem from "expo-file-system/legacy";

/**
 * Raw storage behind `lib/tracking-store.ts` on native platforms: text files in the app's
 * document directory. The web variant (`tracking-files.web.ts`) uses IndexedDB.
 * `tracking-store.ts` serializes the calls per file and swallows errors.
 */

const dir = FileSystem.documentDirectory;

export async function writeFile(name: string, text: string): Promise<void> {
  if (!dir) return;
  await FileSystem.writeAsStringAsync(dir + name, text);
}

export async function readFile(name: string): Promise<string | null> {
  if (!dir) return null;
  const info = await FileSystem.getInfoAsync(dir + name);
  return info.exists ? FileSystem.readAsStringAsync(dir + name) : null;
}

export async function removeFile(name: string): Promise<void> {
  if (!dir) return;
  await FileSystem.deleteAsync(dir + name, { idempotent: true });
}
