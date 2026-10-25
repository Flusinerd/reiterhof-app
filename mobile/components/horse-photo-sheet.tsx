import * as ImagePicker from "expo-image-picker";
import { Camera, Image as ImageIcon, Trash2 } from "lucide-react-native";
import { useState, type ReactElement } from "react";
import { Alert } from "react-native";

import { useConsentPrompt } from "@/components/consent-prompt";
import { Button, Sheet, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { horsesApi, useHorseMutation, type Horse } from "@/lib/api/horses";
import { PHOTO_PICKER_OPTIONS, pickedFile, uploadErrorMessage, uploadFile, type UploadInput } from "@/lib/upload";

/**
 * The photo of a horse: a sheet to take or pick a picture ("Foto aufnehmen", "Aus Fotos wählen") or
 * remove it. The file goes through `uploadFile()`, then PATCH /horses/{id} stores its path (owner or
 * admin). Render `sheet` once in the screen and call `open()` from the hero.
 */
export function useHorsePhotoSheet(horse: Pick<Horse, "id" | "name" | "photo_url">): {
  open: () => void;
  sheet: ReactElement;
} {
  const [visible, setVisible] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const consent = useConsentPrompt();
  const save = useHorseMutation(
    (photoPath: string) => horsesApi.update(horse.id, { photo_path: photoPath }),
    () => [{ type: "horseUpdated", horseId: horse.id }],
  );
  const busy = save.isPending;

  async function attach(file: UploadInput) {
    setError(null);
    try {
      const saved = await uploadFile(file);
      await save.mutateAsync(saved.path);
      setVisible(false);
    } catch (err) {
      setError(uploadErrorMessage(err));
    }
  }

  // Camera and photo library need the photos consent (JAN-19). Two sheets must not be open at once.
  async function photoConsent(): Promise<boolean> {
    let closed = false;
    const ok = await consent.ensure("photos", () => {
      closed = true;
      setVisible(false);
    });
    if (closed) setVisible(true);
    return ok;
  }

  async function takePhoto() {
    if (!(await photoConsent())) return;
    const perm = await ImagePicker.requestCameraPermissionsAsync();
    if (!perm.granted) return setError("Kamerazugriff fehlt.");
    const res = await ImagePicker.launchCameraAsync({ ...PHOTO_PICKER_OPTIONS, allowsEditing: true, aspect: [4, 3], quality: 0.8 });
    const a = res.assets?.[0];
    if (!res.canceled && a) await attach(pickedFile(a, "pferd.jpg"));
  }

  async function pickPhoto() {
    if (!(await photoConsent())) return;
    const res = await ImagePicker.launchImageLibraryAsync({ ...PHOTO_PICKER_OPTIONS, allowsEditing: true, aspect: [4, 3], quality: 0.8 });
    const a = res.assets?.[0];
    if (!res.canceled && a) await attach(pickedFile(a, "pferd.jpg"));
  }

  function confirmRemove() {
    Alert.alert("Foto entfernen?", `Das Foto von ${horse.name} wird gelöscht.`, [
      { text: "Abbrechen", style: "cancel" },
      {
        text: "Entfernen",
        style: "destructive",
        onPress: () =>
          void save
            .mutateAsync("")
            .then(() => setVisible(false))
            .catch((e) => setError(errorMessage(e))),
      },
    ]);
  }

  const sheet = (
    <>
      <Sheet
        open={visible}
        onOpenChange={setVisible}
        title={`Foto von ${horse.name}`}
        description="Alle im Stall sehen das Foto."
      >
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        <Button label="Foto aufnehmen" icon={Camera} variant="outline" fullWidth loading={busy} onPress={() => void takePhoto()} />
        <Button label="Aus Fotos wählen" icon={ImageIcon} variant="outline" fullWidth loading={busy} onPress={() => void pickPhoto()} />
        {horse.photo_url ? (
          <Button label="Foto entfernen" icon={Trash2} variant="ghost" fullWidth disabled={busy} onPress={confirmRemove} />
        ) : null}
      </Sheet>
      {consent.sheet}
    </>
  );

  return {
    open: () => {
      setError(null);
      setVisible(true);
    },
    sheet,
  };
}
