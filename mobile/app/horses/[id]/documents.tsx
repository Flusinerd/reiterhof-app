import * as DocumentPicker from "expo-document-picker";
import * as ImagePicker from "expo-image-picker";
import { useLocalSearchParams } from "expo-router";
import { Camera, FileText, Image as ImageIcon, Paperclip, Plus, Trash2 } from "lucide-react-native";
import { useState } from "react";
import { Alert, View } from "react-native";

import { AuthImage } from "@/components/auth-image";
import { useConsentPrompt } from "@/components/consent-prompt";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Button, Icon, Input, PageHeader, Pill, Screen, Sheet, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { errorMessage } from "@/lib/api";
import { horsesApi, useDocuments, useHorse, useHorseMutation, type HorseDocument } from "@/lib/api/horses";
import { DOCUMENT_KINDS, documentKindLabel, formatDate } from "@/lib/horse-format";
import {
  openStoredFile,
  PHOTO_PICKER_OPTIONS,
  pickedFile,
  uploadErrorMessage,
  uploadFile,
  type UploadInput,
} from "@/lib/upload";

/** Documents of a horse (JAN-53): equine passport, vaccination record, insurance, other. */
export default function Documents() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const horse = useHorse(id);
  const docs = useDocuments(id);
  const add = useHorseMutation(
    (v: { kind: string; title: string; file: UploadInput }) =>
      uploadFile(v.file).then((saved) =>
        horsesApi.addDocument(id, { kind: v.kind, title: v.title, file_path: saved.path }),
      ),
    () => [{ type: "documents", horseId: id }],
  );
  const remove = useHorseMutation(
    (docId: string) => horsesApi.removeDocument(id, docId),
    () => [{ type: "documents", horseId: id }],
  );
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<string>("passport");
  const [title, setTitle] = useState("");
  const [error, setError] = useState<string | null>(null);
  const consent = useConsentPrompt();

  if (!docs.data) {
    return (
      <Screen back>
        {docs.isError ? (
          <HorseError
            error={docs.error}
            onRetry={() => docs.refetch()}
            forbiddenText="Nur für Besitzer, Reitbeteiligungen und Admins."
          />
        ) : (
          <HorseLoading />
        )}
      </Screen>
    );
  }

  const canManage = horse.data?.can_manage ?? false;

  async function attach(file: UploadInput) {
    setError(null);
    const fallbackTitle = documentKindLabel(kind);
    const name = title.trim() || file.name.replace(/\.[^.]+$/, "") || fallbackTitle;
    try {
      await add.mutateAsync({ kind, title: name, file });
      setOpen(false);
      setTitle("");
    } catch (err) {
      setError(uploadErrorMessage(err));
    }
  }

  // Camera and photo library need the photos consent (JAN-19). Two sheets must not be open at once.
  async function photoConsent(): Promise<boolean> {
    let closed = false;
    const ok = await consent.ensure("photos", () => {
      closed = true;
      setOpen(false);
    });
    if (closed) setOpen(true);
    return ok;
  }

  async function takePhoto() {
    if (!(await photoConsent())) return;
    const perm = await ImagePicker.requestCameraPermissionsAsync();
    if (!perm.granted) return setError("Kamerazugriff fehlt.");
    const res = await ImagePicker.launchCameraAsync({ ...PHOTO_PICKER_OPTIONS, quality: 0.8 });
    const a = res.assets?.[0];
    if (!res.canceled && a) await attach(pickedFile(a, "foto.jpg"));
  }

  async function pickPhoto() {
    if (!(await photoConsent())) return;
    const res = await ImagePicker.launchImageLibraryAsync({ ...PHOTO_PICKER_OPTIONS, quality: 0.8 });
    const a = res.assets?.[0];
    if (!res.canceled && a) await attach(pickedFile(a, "foto.jpg"));
  }

  async function pickFile() {
    const res = await DocumentPicker.getDocumentAsync({
      // Only what the server stores; "image/*" offered HEIC files that it refuses.
      type: ["application/pdf", "image/jpeg", "image/png", "image/webp"],
      copyToCacheDirectory: true,
    });
    const a = res.assets?.[0];
    if (!res.canceled && a) await attach(pickedFile(a, "dokument.pdf"));
  }

  function confirmRemove(doc: HorseDocument) {
    Alert.alert("Dokument löschen?", `„${doc.title}“ wird gelöscht.`, [
      { text: "Abbrechen", style: "cancel" },
      {
        text: "Löschen",
        style: "destructive",
        onPress: () => void remove.mutateAsync(doc.id).catch((e) => Alert.alert("Das hat nicht geklappt", errorMessage(e))),
      },
    ]);
  }

  const busy = add.isPending;
  const list = docs.data;

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <PageHeader
        eyebrow={horse.data?.name ?? "Pferdeakte"}
        title="Dokumente"
        description="Sichtbar für Besitzer, Reitbeteiligungen und Admins."
        action={
          canManage ? (
            <Button
              icon={Plus}
              variant="outline"
              size="icon"
              accessibilityLabel="Dokument hinzufügen"
              onPress={() => {
                setError(null);
                setOpen(true);
              }}
            />
          ) : undefined
        }
      />

      {list.length === 0 ? (
        <Text variant="body" tone="muted">
          Noch keine Dokumente.
        </Text>
      ) : (
        <RowList>
          {list.map((doc) => (
            <ListRow
              key={doc.id}
              leading={
                doc.content_type.startsWith("image/") ? (
                  <AuthImage
                    url={doc.url}
                    accessibilityIgnoresInvertColors
                    className="h-12 w-12 rounded-button-sm bg-divider"
                  />
                ) : (
                  <View className="h-12 w-12 items-center justify-center rounded-button-sm bg-primary-soft">
                    <Icon as={FileText} size={22} className="text-primary-deep" />
                  </View>
                )
              }
              eyebrow={documentKindLabel(doc.kind)}
              title={doc.title}
              detail={formatDate(doc.created_at)}
              accessibilityLabel={`${doc.title} öffnen`}
              chevron={false}
              onPress={() => void openStoredFile(doc.url).catch((e) => setError(errorMessage(e)))}
              trailing={
                canManage ? (
                  <Button
                    size="icon"
                    variant="ghost"
                    icon={Trash2}
                    accessibilityLabel={`${doc.title} löschen`}
                    onPress={() => confirmRemove(doc)}
                  />
                ) : null
              }
            />
          ))}
        </RowList>
      )}

      <Sheet open={open} onOpenChange={setOpen} title="Dokument hinzufügen" description="Foto oder PDF, höchstens 20 MB.">
        <View className="flex-row flex-wrap gap-2">
          {DOCUMENT_KINDS.map((k) => (
            <Pill key={k} label={documentKindLabel(k)} selected={kind === k} onPress={() => setKind(k)} />
          ))}
        </View>
        <Input value={title} onChangeText={setTitle} accessibilityLabel="Titel" placeholder="Titel (optional)" />
        {error ? (
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        ) : null}
        <Button label="Foto aufnehmen" icon={Camera} variant="outline" fullWidth loading={busy} onPress={() => void takePhoto()} />
        <Button label="Aus Fotos wählen" icon={ImageIcon} variant="outline" fullWidth loading={busy} onPress={() => void pickPhoto()} />
        <Button label="Datei wählen (PDF)" icon={Paperclip} fullWidth loading={busy} onPress={() => void pickFile()} />
      </Sheet>
      {consent.sheet}
    </Screen>
  );
}
