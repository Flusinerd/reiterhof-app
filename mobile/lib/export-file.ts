import { File, Paths } from "expo-file-system";
import * as Sharing from "expo-sharing";
import { Share } from "react-native";

/**
 * Hands the data export (JSON text) to the user: writes a file to the cache and opens the
 * share sheet, or shares the text when file sharing is unavailable.
 * The web variant (`export-file.web.ts`) downloads the file instead.
 */
export async function shareTextFile(fileName: string, text: string): Promise<void> {
  const file = new File(Paths.cache, fileName);
  file.create({ overwrite: true });
  file.write(text);
  if (await Sharing.isAvailableAsync()) {
    await Sharing.shareAsync(file.uri, { mimeType: "application/json", dialogTitle: "Meine Stallfunk-Daten" });
  } else {
    await Share.share({ message: text });
  }
}
