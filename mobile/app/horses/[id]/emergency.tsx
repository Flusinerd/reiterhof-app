import { router, useLocalSearchParams, type Href } from "expo-router";
import { Pencil, Phone, Plus } from "lucide-react-native";
import { useState } from "react";
import { Alert, Linking, View } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Button, Card, Divider, Hero, Input, Screen, SectionLabel, Sheet, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { horsesApi, useEmergency, useHorseMutation, type Contact } from "@/lib/api/horses";
import { horseRoutes, telUrl } from "@/lib/horse-format";

type CallRowProps = { label: string; name: string | null | undefined; phone: string | null | undefined; onEdit?: () => void };

/** One person to call: label, name and a 44 px call button (tel: link). */
function CallRow({ label, name, phone, onEdit }: CallRowProps) {
  const url = telUrl(phone);
  return (
    <View className="min-h-touch flex-row items-center gap-3">
      <View className="flex-1">
        <Text variant="secondary">{label}</Text>
        <Text variant="bodyStrong">{name || "Nicht eingetragen"}</Text>
        {phone ? <Text variant="bodySm">{phone}</Text> : null}
      </View>
      {onEdit ? (
        <Button size="icon" variant="ghost" icon={Pencil} accessibilityLabel={`${label} bearbeiten`} onPress={onEdit} />
      ) : null}
      {url ? (
        <Button
          size="icon"
          icon={Phone}
          accessibilityLabel={`${name ?? label} anrufen`}
          onPress={() => void Linking.openURL(url)}
        />
      ) : null}
    </View>
  );
}

function Fact({ label, value }: { label: string; value: string | number | null | undefined }) {
  if (value === null || value === undefined || value === "") return null;
  return (
    <View className="gap-1">
      <Text variant="secondary">{label}</Text>
      <Text variant="body">{value}</Text>
    </View>
  );
}

type ContactDraft = { id?: string; label: string; name: string; phone: string };

/** Full emergency card of a horse (JAN-49): visible to everybody in the stable, tap to call. */
export default function Emergency() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const card = useEmergency(id);
  const save = useHorseMutation((c: ContactDraft) =>
    c.id
      ? horsesApi.updateContact(id, c.id, { label: c.label, name: c.name, phone: c.phone })
      : horsesApi.addContact(id, { label: c.label, name: c.name, phone: c.phone }),
  );
  const remove = useHorseMutation((contactId: string) => horsesApi.removeContact(id, contactId));
  const [draft, setDraft] = useState<ContactDraft | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (!card.data) {
    return (
      <Screen back>
        {card.isError ? <HorseError error={card.error} onRetry={() => card.refetch()} /> : <HorseLoading />}
      </Screen>
    );
  }
  const c = card.data;

  async function saveDraft() {
    if (!draft) return;
    if (!draft.label.trim() || !draft.name.trim() || !draft.phone.trim()) {
      setError("Bezeichnung, Name und Telefonnummer fehlen.");
      return;
    }
    setError(null);
    try {
      await save.mutateAsync(draft);
      setDraft(null);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  function confirmRemove(contact: Contact) {
    Alert.alert("Kontakt löschen?", `${contact.name} wird von der Notfallkarte entfernt.`, [
      { text: "Abbrechen", style: "cancel" },
      {
        text: "Löschen",
        style: "destructive",
        onPress: () => void remove.mutateAsync(contact.id).then(() => setDraft(null)).catch((e) => setError(errorMessage(e))),
      },
    ]);
  }

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <Hero
        tone="warm"
        eyebrow="Notfallkarte"
        title={c.horse_name}
        description={c.emergency_note ?? "Kein Notfall-Hinweis."}
      >
        {c.can_manage ? (
          <Button
            label="Bearbeiten"
            icon={Pencil}
            variant="outline"
            size="sm"
            onPress={() => router.push(horseRoutes.edit(c.horse_id) as Href)}
          />
        ) : null}
      </Hero>

      <SectionLabel>Anrufen</SectionLabel>
      <Card className="gap-3">
        <CallRow label="Besitzer" name={c.owner?.name} phone={c.owner?.phone} />
        <Divider />
        <CallRow label="Tierarzt" name={c.vet_name} phone={c.vet_phone} />
        {c.contacts.map((contact) => (
          <View key={contact.id} className="gap-3">
            <Divider />
            <CallRow
              label={contact.label}
              name={contact.name}
              phone={contact.phone}
              onEdit={c.can_manage ? () => { setError(null); setDraft({ ...contact }); } : undefined}
            />
          </View>
        ))}
        {c.can_manage ? (
          <Button
            label="Kontakt hinzufügen"
            icon={Plus}
            variant="outline"
            size="sm"
            onPress={() => {
              setError(null);
              setDraft({ label: "", name: "", phone: "" });
            }}
          />
        ) : null}
      </Card>

      <SectionLabel>Medizinisches</SectionLabel>
      <Card className="gap-4">
        <Fact label="Notfall-Medikament" value={c.emergency_medication} />
        <Fact label="Dauermedikation" value={c.permanent_medication} />
        <Fact label="Allergien" value={c.allergies} />
        <Fact label="Gewicht" value={c.weight_kg ? `${c.weight_kg} kg` : null} />
        <Fact label="Versicherung" value={c.insurance} />
        {!c.emergency_medication && !c.permanent_medication && !c.allergies && !c.weight_kg && !c.insurance ? (
          <Text variant="secondary">Noch keine Angaben.</Text>
        ) : null}
      </Card>

      <Sheet
        open={draft !== null}
        onOpenChange={(open) => !open && setDraft(null)}
        title={draft?.id ? "Kontakt bearbeiten" : "Kontakt hinzufügen"}
      >
        {draft ? (
          <>
            <Input
              value={draft.label}
              onChangeText={(label) => setDraft({ ...draft, label })}
              accessibilityLabel="Bezeichnung"
              placeholder="Bezeichnung, z. B. Hufschmied"
            />
            <Input
              value={draft.name}
              onChangeText={(name) => setDraft({ ...draft, name })}
              accessibilityLabel="Name"
              placeholder="Name"
            />
            <Input
              value={draft.phone}
              onChangeText={(phone) => setDraft({ ...draft, phone })}
              accessibilityLabel="Telefonnummer"
              placeholder="Telefonnummer"
              keyboardType="phone-pad"
            />
            {error ? (
              <Text variant="bodySm" tone="danger" accessibilityRole="alert">
                {error}
              </Text>
            ) : null}
            <Button label="Speichern" fullWidth loading={save.isPending} onPress={saveDraft} />
            {draft.id ? (
              <Button
                label="Kontakt löschen"
                variant="ghost"
                fullWidth
                onPress={() => confirmRemove(draft as Contact)}
              />
            ) : null}
          </>
        ) : null}
      </Sheet>
    </Screen>
  );
}
