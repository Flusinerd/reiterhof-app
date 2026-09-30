import * as ImagePicker from "expo-image-picker";
import { Camera, Image as ImageIcon, Trash2 } from "lucide-react-native";
import { useEffect, useState } from "react";
import { Alert, Image, ScrollView, View, useWindowDimensions } from "react-native";

import { BlanketPhoto } from "@/components/blanket-photo";
import { Button, Input, Sheet, Text } from "@/components/ui";
import { blanketsApi, usePlanMutation } from "@/lib/api/blankets";
import { blanketErrorMessage, type Blanket } from "@/lib/blankets";
import { fileSource, uploadFile, type UploadInput } from "@/lib/upload";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  horseId: string;
  /** The blanket to edit, or null to create a new one. */
  blanket: Blanket | null;
};

type Photo = { kind: "keep" } | { kind: "remove" } | { kind: "new"; file: UploadInput };

/** Create or edit a blanket: name, fill weight, color, location and photo (JAN-29). */
export function BlanketEditSheet({ open, onOpenChange, horseId, blanket }: Props) {
  const { height } = useWindowDimensions();
  const [name, setName] = useState("");
  const [fill, setFill] = useState("");
  const [color, setColor] = useState("");
  const [location, setLocation] = useState("");
  const [photo, setPhoto] = useState<Photo>({ kind: "keep" });
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setName(blanket?.name ?? "");
    setFill(blanket ? String(blanket.fill_g) : "");
    setColor(blanket?.color ?? "");
    setLocation(blanket?.location ?? "");
    setPhoto({ kind: "keep" });
    setError(null);
  }, [open, blanket]);

  const save = usePlanMutation(async () => {
    const fillG = fill.trim() === "" ? 0 : Number(fill.trim());
    let photoPath: string | undefined;
    if (photo.kind === "new") photoPath = (await uploadFile(photo.file)).path;
    else if (photo.kind === "remove") photoPath = "";
    const input = {
      name: name.trim(),
      fill_g: fillG,
      color: color.trim(),
      location: location.trim(),
      ...(photoPath !== undefined ? { photo_path: photoPath } : {}),
    };
    return blanket ? blanketsApi.updateBlanket(horseId, blanket.id, input) : blanketsApi.createBlanket(horseId, input);
  });
  const remove = usePlanMutation(() => blanketsApi.removeBlanket(horseId, blanket!.id));

  async function submit() {
    setError(null);
    if (name.trim() === "") return setError("Bitte gib einen Namen ein, zum Beispiel „Regendecke“.");
    const fillG = fill.trim() === "" ? 0 : Number(fill.trim());
    if (!Number.isInteger(fillG) || fillG < 0 || fillG > 2000) {
      return setError("Die Füllung muss eine ganze Zahl in Gramm sein, zum Beispiel 100.");
    }
    try {
      await save.mutateAsync();
      onOpenChange(false);
    } catch (e) {
      setError(blanketErrorMessage(e));
    }
  }

  function confirmRemove() {
    if (!blanket) return;
    Alert.alert("Decke löschen?", `„${blanket.name}“ wird endgültig gelöscht.`, [
      { text: "Abbrechen", style: "cancel" },
      {
        text: "Löschen",
        style: "destructive",
        onPress: () =>
          void remove
            .mutateAsync()
            .then(() => onOpenChange(false))
            .catch((e) => setError(blanketErrorMessage(e))),
      },
    ]);
  }

  async function take(source: "camera" | "library") {
    setError(null);
    if (source === "camera") {
      const perm = await ImagePicker.requestCameraPermissionsAsync();
      if (!perm.granted) return setError("Ohne Kamerazugriff kann kein Foto aufgenommen werden.");
    }
    const options = { mediaTypes: ["images" as const], quality: 0.7 };
    const res =
      source === "camera"
        ? await ImagePicker.launchCameraAsync(options)
        : await ImagePicker.launchImageLibraryAsync(options);
    const a = res.assets?.[0];
    if (!res.canceled && a) {
      setPhoto({
        kind: "new",
        file: { uri: a.uri, name: a.fileName ?? "decke.jpg", mimeType: a.mimeType ?? "image/jpeg", size: a.fileSize },
      });
    }
  }

  const shownUrl = photo.kind === "keep" ? (blanket?.photo_url ?? null) : null;
  const busy = save.isPending || remove.isPending;

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title={blanket ? "Decke bearbeiten" : "Neue Decke"}
      description="Foto, Füllung und Aufbewahrungsort helfen den Helfern, die richtige Decke zu finden."
    >
      <ScrollView style={{ maxHeight: height * 0.55 }} contentContainerClassName="gap-4" keyboardShouldPersistTaps="handled">
        <View className="flex-row items-center gap-4">
          {photo.kind === "new" ? (
            <Image
              source={{ uri: photo.file.uri }}
              accessibilityIgnoresInvertColors
              style={{ width: 88, height: 88 }}
              className="rounded-tile bg-divider"
            />
          ) : shownUrl ? (
            <Image
              source={fileSource(shownUrl)}
              accessibilityIgnoresInvertColors
              style={{ width: 88, height: 88 }}
              className="rounded-tile bg-divider"
            />
          ) : (
            <BlanketPhoto url={null} size={88} />
          )}
          <View className="flex-1 gap-2">
            <Button label="Foto aufnehmen" icon={Camera} variant="outline" size="sm" fullWidth onPress={() => void take("camera")} />
            <Button label="Aus Fotos wählen" icon={ImageIcon} variant="outline" size="sm" fullWidth onPress={() => void take("library")} />
          </View>
        </View>
        {photo.kind !== "remove" && (photo.kind === "new" || blanket?.photo_url) ? (
          <Button label="Foto entfernen" variant="ghost" size="sm" onPress={() => setPhoto({ kind: "remove" })} />
        ) : null}

        <Input value={name} onChangeText={setName} accessibilityLabel="Name" placeholder="Name, z. B. Regendecke" maxLength={80} />
        <Input
          value={fill}
          onChangeText={setFill}
          accessibilityLabel="Füllung in Gramm"
          placeholder="Füllung in Gramm, z. B. 100 (0 = ohne)"
          keyboardType="number-pad"
        />
        <Input value={color} onChangeText={setColor} accessibilityLabel="Farbe" placeholder="Farbe, z. B. dunkelblau" maxLength={40} />
        <Input
          value={location}
          onChangeText={setLocation}
          accessibilityLabel="Aufbewahrungsort"
          placeholder="Wo hängt sie? z. B. Haken 3"
          maxLength={80}
        />
      </ScrollView>

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button label="Speichern" fullWidth loading={save.isPending} disabled={busy} onPress={() => void submit()} />
      {blanket ? (
        <Button label="Decke löschen" icon={Trash2} variant="ghost" fullWidth loading={remove.isPending} disabled={busy} onPress={confirmRemove} />
      ) : null}
    </Sheet>
  );
}
