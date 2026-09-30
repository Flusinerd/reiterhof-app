// Browser helpers for handing a generated file to the user. Only imported by `.web.ts` modules
// (they use the DOM). The decision logic is in web-share-core.ts.

import { chooseShareStrategy } from "./web-share-core.ts";

/** Saves a blob as a file through a temporary link (iOS Safari shows its download/preview sheet). */
export function downloadBlob(blob: Blob, fileName: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  link.rel = "noopener";
  document.body.appendChild(link);
  link.click();
  link.remove();
  // Revoke later: Safari needs the URL until the download started.
  setTimeout(() => URL.revokeObjectURL(url), 30_000);
}

export type DeliverResult = "shared" | "downloaded" | "cancelled";

/**
 * Shares the file through the Web Share API when the browser can, otherwise downloads it.
 * A dismissed share sheet ("AbortError") is reported as "cancelled", not as an error.
 * Call it from a tap; iOS refuses `navigator.share` a few seconds after the gesture.
 */
export async function deliverFile(blob: Blob, fileName: string, title: string): Promise<DeliverResult> {
  const file = new File([blob], fileName, { type: blob.type });
  if (chooseShareStrategy(navigator, file) === "share-file") {
    try {
      await navigator.share({ files: [file], title });
      return "shared";
    } catch (err) {
      if ((err as { name?: string }).name === "AbortError") return "cancelled";
      // NotAllowedError (gesture expired) and others: fall through to the download
    }
  }
  downloadBlob(blob, fileName);
  return "downloaded";
}
