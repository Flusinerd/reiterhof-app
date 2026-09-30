// Pure helpers for downloading and sharing generated files from the browser (unit-tested).

export type ShareStrategy = "share-file" | "download";

/**
 * How to hand a generated file to the user: the Web Share API when the browser can share
 * this file (iOS Safari can), otherwise a download.
 */
export function chooseShareStrategy(
  nav: { share?: unknown; canShare?: ((data: { files: File[] }) => boolean) | undefined },
  file: File,
): ShareStrategy {
  if (typeof nav.share !== "function" || typeof nav.canShare !== "function") return "download";
  try {
    return nav.canShare({ files: [file] }) ? "share-file" : "download";
  } catch {
    return "download";
  }
}

/** File name of the ICS file of one request; the snapshot of the whole calendar has no id. */
export function icsFileName(id?: string): string {
  const safe = id?.replace(/[^A-Za-z0-9_-]/g, "");
  return safe ? `stallfunk-termin-${safe}.ics` : "stallfunk-termine.ics";
}
