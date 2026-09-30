import * as ImagePicker from "expo-image-picker";
import { router, useLocalSearchParams, type Href } from "expo-router";
import { Camera, Eye, Image as ImageIcon, Info, Send, Siren, X } from "lucide-react-native";
import { useRef, useState } from "react";
import { Alert, Image, Pressable, View } from "react-native";

import { ObservationBodyPicker } from "@/components/observation-body-picker";
import { Button, Hero, Icon, Input, Pill, Screen, SectionLabel, Text, ToggleGroup, ToggleGroupItem } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { useHorses } from "@/lib/api/horses";
import { useReportObservation } from "@/lib/api/observations";
import { horseRoutes } from "@/lib/horse-format";
import {
  CATEGORIES,
  categoryLabel,
  MAX_DESCRIPTION,
  MAX_PHOTOS,
  reportProblem,
  URGENT_CONFIRM,
  urgencyHint,
  type Urgency,
} from "@/lib/observations";
import { fileSource, uploadErrorMessage, uploadFile, type UploadInput } from "@/lib/upload";

function confirmUrgent(): Promise<boolean> {
  return new Promise((resolve) => {
    Alert.alert(
      URGENT_CONFIRM.title,
      URGENT_CONFIRM.message,
      [
        { text: "Abbrechen", style: "cancel", onPress: () => resolve(false) },
        { text: "Dringend melden", style: "destructive", onPress: () => resolve(true) },
      ],
      { cancelable: true, onDismiss: () => resolve(false) },
    );
  });
}

/** Report an observation (JAN-50): horse, what, where, description, photos, urgency. */
export default function NewObservation() {
  const params = useLocalSearchParams<{ horse?: string }>();
  const horses = useHorses();
  const report = useReportObservation();
  const [horseId, setHorseId] = useState<string | null>(params.horse ?? null);
  const [category, setCategory] = useState<string | null>(null);
  const [bodyPart, setBodyPart] = useState<string | null>(null);
  const [description, setDescription] = useState("");
  const [photos, setPhotos] = useState<UploadInput[]>([]);
  const [urgency, setUrgency] = useState<Urgency>("info");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Uploaded photos by local uri, so a retry after a failed report does not upload them again.
  const uploaded = useRef(new Map<string, string>());

  async function addPhotos(assets: ImagePicker.ImagePickerAsset[]) {
    const next = assets.map<UploadInput>((a) => ({
      uri: a.uri,
      name: a.fileName ?? "foto.jpg",
      mimeType: a.mimeType ?? "image/jpeg",
      size: a.fileSize,
    }));
    setPhotos((cur) => [...cur, ...next].slice(0, MAX_PHOTOS));
  }

  async function takePhoto() {
    const perm = await ImagePicker.requestCameraPermissionsAsync();
    if (!perm.granted) return setError("Ohne Kamerazugriff kann kein Foto aufgenommen werden.");
    const res = await ImagePicker.launchCameraAsync({ mediaTypes: ["images"], quality: 0.7 });
    if (!res.canceled) await addPhotos(res.assets);
  }

  async function pickPhotos() {
    const res = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images"],
      quality: 0.7,
      allowsMultipleSelection: true,
      selectionLimit: Math.max(1, MAX_PHOTOS - photos.length),
    });
    if (!res.canceled) await addPhotos(res.assets);
  }

  async function submit() {
    setError(null);
    const problem = reportProblem({ horseId, category, description, photos: photos.length });
    if (problem || !horseId || !category) return setError(problem);
    if (urgency === "urgent" && !(await confirmUrgent())) return;

    setSending(true);
    try {
      const media: string[] = [];
      for (const photo of photos) {
        let path = uploaded.current.get(photo.uri);
        if (!path) {
          try {
            path = (await uploadFile(photo)).path;
          } catch (err) {
            return setError(uploadErrorMessage(err));
          }
          uploaded.current.set(photo.uri, path);
        }
        media.push(path);
      }
      let result;
      try {
        result = await report.mutateAsync({
          horse_id: horseId,
          category,
          ...(bodyPart ? { body_part: bodyPart } : {}),
          ...(description.trim() ? { description: description.trim() } : {}),
          ...(media.length > 0 ? { media } : {}),
          urgency,
        });
      } catch (err) {
        return setError(errorMessage(err));
      }
      if (result.emergency) {
        // The card is already in the cache; replace this screen so "back" returns to where the user came from.
        router.replace(horseRoutes.emergency(horseId) as Href);
      } else {
        router.back();
      }
    } finally {
      setSending(false);
    }
  }

  const urgent = urgency === "urgent";

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <Hero
        eyebrow="Auffälligkeit"
        title="Melden"
        description="Was ist dir am Pferd aufgefallen? Besitzer und Reitbeteiligungen werden informiert."
      />

      <SectionLabel>Pferd</SectionLabel>
      <View className="flex-row flex-wrap gap-2">
        {(horses.data ?? []).map((h) => (
          <Pill key={h.id} label={h.name} selected={horseId === h.id} onPress={() => setHorseId(h.id)} />
        ))}
        {horses.isPending ? <Text variant="secondary">Pferde werden geladen …</Text> : null}
        {horses.isError ? <Text variant="secondary" tone="danger">{errorMessage(horses.error)}</Text> : null}
      </View>

      <SectionLabel>Was ist aufgefallen?</SectionLabel>
      <View className="flex-row flex-wrap gap-2">
        {CATEGORIES.map((c) => (
          <Pill key={c} label={categoryLabel(c)} selected={category === c} onPress={() => setCategory(c)} />
        ))}
      </View>

      <SectionLabel>Wo?</SectionLabel>
      <ObservationBodyPicker value={bodyPart} onChange={setBodyPart} />

      <SectionLabel>Beschreibung</SectionLabel>
      <Input
        value={description}
        onChangeText={setDescription}
        accessibilityLabel="Beschreibung"
        placeholder="Seit wann, wie stark, was hast du gesehen? (optional)"
        multiline
        textAlignVertical="top"
        maxLength={MAX_DESCRIPTION}
        className="h-32 py-3"
      />

      <SectionLabel>Fotos</SectionLabel>
      <View className="gap-3">
        {photos.length > 0 ? (
          <View className="flex-row flex-wrap gap-3">
            {photos.map((p) => (
              <View key={p.uri} className="h-24 w-24">
                <Image
                  source={{ uri: p.uri }}
                  accessibilityIgnoresInvertColors
                  className="h-24 w-24 rounded-tile bg-divider"
                />
                <Pressable
                  accessibilityRole="button"
                  accessibilityLabel="Foto entfernen"
                  hitSlop={8}
                  onPress={() => setPhotos((cur) => cur.filter((x) => x.uri !== p.uri))}
                  className="absolute -right-2 -top-2 h-7 w-7 items-center justify-center rounded-pill border border-border bg-card"
                >
                  <Icon as={X} size={16} className="text-foreground" />
                </Pressable>
              </View>
            ))}
          </View>
        ) : null}
        {photos.length < MAX_PHOTOS ? (
          <View className="flex-row gap-3">
            <Button label="Aufnehmen" icon={Camera} variant="outline" className="flex-1" onPress={() => void takePhoto()} />
            <Button label="Aus Fotos" icon={ImageIcon} variant="outline" className="flex-1" onPress={() => void pickPhotos()} />
          </View>
        ) : null}
        <Text variant="caption">Bis zu {MAX_PHOTOS} Fotos.</Text>
      </View>

      <SectionLabel>Wie dringend?</SectionLabel>
      <View className="gap-3">
        <ToggleGroup type="single" value={urgency} onValueChange={(v) => setUrgency(v as Urgency)}>
          <ToggleGroupItem value="info" label="Info" icon={Info} />
          <ToggleGroupItem value="check" label="Bitte ansehen" icon={Eye} />
          <ToggleGroupItem value="urgent" label="Dringend" icon={Siren} />
        </ToggleGroup>
        <Text variant="secondary" tone={urgent ? "danger" : "muted"}>
          {urgencyHint(urgency)}
        </Text>
      </View>

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button
        label={urgent ? "Dringend melden" : "Melden"}
        icon={urgent ? Siren : Send}
        variant={urgent ? "danger" : "primary"}
        size="lg"
        fullWidth
        loading={sending}
        onPress={() => void submit()}
      />
    </Screen>
  );
}
