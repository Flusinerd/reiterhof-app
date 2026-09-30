import { deliverFile } from "./web-download";

/** Web variant of `lib/export-file.ts`: share sheet with the file where supported, else a download. */
export async function shareTextFile(fileName: string, text: string): Promise<void> {
  await deliverFile(new Blob([text], { type: "application/json" }), fileName, "Meine Stallfunk-Daten");
}
