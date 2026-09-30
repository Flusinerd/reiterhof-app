import { Check } from "lucide-react-native";
import { useState } from "react";
import { Pressable, View } from "react-native";

import {
  Button,
  Icon,
  Input,
  Pill,
  Section,
  Text,
  ToggleGroup,
  ToggleGroupItem,
} from "@/components/ui";
import type { EmergencyCard, Horse, HorseInput, Member } from "@/lib/api/horses";
import { COLOR_KEYS, COLOR_PAIRS } from "@/lib/color-keys";
import { parseWholeNumber, sexLabel } from "@/lib/horse-format";
import { cn } from "@/lib/cn";

const MIN_BIRTH_YEAR = 1970;

type Props = {
  horse?: Horse;
  card?: EmergencyCard;
  /** Admins may choose the owner. */
  isAdmin: boolean;
  members?: Member[];
  submitLabel: string;
  saving: boolean;
  /** German error text from the last attempt. */
  error?: string | null;
  onSubmit: (input: HorseInput) => void;
};

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <View className="gap-1.5">
      <Text variant="label">{label}</Text>
      {children}
    </View>
  );
}

/**
 * Create / edit form for a horse including the emergency card fields. Used by
 * `app/horses/new.tsx` and `app/horses/[id]/edit.tsx`. Empty text clears a field on the server.
 */
export function HorseForm({ horse, card, isAdmin, members, submitLabel, saving, error, onSubmit }: Props) {
  const [name, setName] = useState(horse?.name ?? "");
  const [box, setBox] = useState(horse?.box ?? "");
  const [sex, setSex] = useState<string>(horse?.sex ?? "");
  const [birthYear, setBirthYear] = useState(horse?.birth_year ? String(horse.birth_year) : "");
  const [breed, setBreed] = useState(horse?.breed ?? "");
  const [colorKey, setColorKey] = useState(horse?.color_key ?? "");
  const [weight, setWeight] = useState(horse?.weight_kg ? String(horse.weight_kg) : "");
  const [helperNote, setHelperNote] = useState(horse?.helper_note ?? "");
  const [ownerId, setOwnerId] = useState(horse?.owner?.id ?? "");
  const [vetName, setVetName] = useState(card?.vet_name ?? "");
  const [vetPhone, setVetPhone] = useState(card?.vet_phone ?? "");
  const [emergencyNote, setEmergencyNote] = useState(card?.emergency_note ?? "");
  const [emergencyMedication, setEmergencyMedication] = useState(card?.emergency_medication ?? "");
  const [permanentMedication, setPermanentMedication] = useState(card?.permanent_medication ?? "");
  const [allergies, setAllergies] = useState(card?.allergies ?? "");
  const [insurance, setInsurance] = useState(card?.insurance ?? "");
  const [problem, setProblem] = useState<string | null>(null);

  function submit() {
    const year = birthYear.trim() === "" ? 0 : parseWholeNumber(birthYear);
    const kg = weight.trim() === "" ? 0 : parseWholeNumber(weight);
    const thisYear = new Date().getFullYear();
    if (name.trim() === "") return setProblem("Name fehlt.");
    if (year === null || (year !== 0 && (year < MIN_BIRTH_YEAR || year > thisYear))) {
      return setProblem(`Geburtsjahr zwischen ${MIN_BIRTH_YEAR} und ${thisYear}.`);
    }
    if (kg === null || (kg !== 0 && (kg < 20 || kg > 1500))) {
      return setProblem("Gewicht zwischen 20 und 1500 kg.");
    }
    setProblem(null);
    const input: HorseInput = {
      name: name.trim(),
      box: box.trim(),
      sex,
      birth_year: year,
      breed: breed.trim(),
      color_key: colorKey,
      weight_kg: kg,
      helper_note: helperNote.trim(),
      vet_name: vetName.trim(),
      vet_phone: vetPhone.trim(),
      emergency_note: emergencyNote.trim(),
      emergency_medication: emergencyMedication.trim(),
      permanent_medication: permanentMedication.trim(),
      allergies: allergies.trim(),
      insurance: insurance.trim(),
    };
    if (isAdmin && ownerId && ownerId !== horse?.owner?.id) input.owner_id = ownerId;
    onSubmit(input);
  }

  return (
    <>
      <Section title="Stammdaten" className="gap-4">
        <Field label="Name">
          <Input value={name} onChangeText={setName} accessibilityLabel="Name" placeholder="z. B. Luna" />
        </Field>
        <Field label="Box">
          <Input value={box} onChangeText={setBox} accessibilityLabel="Box" placeholder="z. B. 12" />
        </Field>
        <Field label="Geschlecht">
          <ToggleGroup type="single" value={sex} onValueChange={(v) => setSex(v === sex ? "" : v)}>
            {(["mare", "gelding", "stallion"] as const).map((s) => (
              <ToggleGroupItem key={s} value={s} label={sexLabel(s)} />
            ))}
          </ToggleGroup>
        </Field>
        <Field label="Geburtsjahr">
          <Input
            value={birthYear}
            onChangeText={setBirthYear}
            accessibilityLabel="Geburtsjahr"
            keyboardType="number-pad"
            maxLength={4}
            placeholder="z. B. 2015"
          />
        </Field>
        <Field label="Rasse">
          <Input value={breed} onChangeText={setBreed} accessibilityLabel="Rasse" />
        </Field>
        <Field label="Gewicht (kg)">
          <Input
            value={weight}
            onChangeText={setWeight}
            accessibilityLabel="Gewicht in Kilogramm"
            keyboardType="number-pad"
            maxLength={4}
          />
        </Field>
        <Field label="Farbe in der App">
          <View className="flex-row flex-wrap gap-1">
            {COLOR_KEYS.filter((k) => k !== "neutral").map((key) => {
              const selected = colorKey === key;
              return (
                <Pressable
                  key={key}
                  accessibilityRole="button"
                  accessibilityLabel={`Farbe ${key}`}
                  accessibilityState={{ selected }}
                  onPress={() => setColorKey(selected ? "" : key)}
                  className="h-11 w-11 items-center justify-center"
                >
                  <View
                    className={cn(
                      "h-9 w-9 items-center justify-center rounded-pill border-2",
                      selected ? "border-primary" : "border-transparent",
                    )}
                    style={{ backgroundColor: COLOR_PAIRS[key].bg }}
                  >
                    {selected ? <Icon as={Check} size={16} color={COLOR_PAIRS[key].fg} /> : null}
                  </View>
                </Pressable>
              );
            })}
          </View>
        </Field>
        <Field label="Hinweise für Helfer">
          <Input
            value={helperNote}
            onChangeText={setHelperNote}
            accessibilityLabel="Hinweise für Helfer"
            multiline
            textAlignVertical="top"
            className="h-auto min-h-24 py-3"
            placeholder="z. B. Futter, Weide, Besonderheiten"
          />
        </Field>
      </Section>

      {isAdmin && members ? (
        <Section title="Besitzer" description="Als Admin wählst du den Besitzer.">
          <View className="flex-row flex-wrap gap-2">
            {members.map((m) => (
              <Pill key={m.id} label={m.name} selected={ownerId === m.id} onPress={() => setOwnerId(m.id)} />
            ))}
          </View>
        </Section>
      ) : null}

      <Section title="Notfallkarte" description="Sichtbar für alle im Stall." className="gap-4">
        <Field label="Tierarzt">
          <Input value={vetName} onChangeText={setVetName} accessibilityLabel="Name des Tierarztes" />
        </Field>
        <Field label="Telefon Tierarzt">
          <Input
            value={vetPhone}
            onChangeText={setVetPhone}
            accessibilityLabel="Telefonnummer des Tierarztes"
            keyboardType="phone-pad"
          />
        </Field>
        <Field label="Notfall-Hinweis">
          <Input
            value={emergencyNote}
            onChangeText={setEmergencyNote}
            accessibilityLabel="Notfall-Hinweis"
            multiline
            textAlignVertical="top"
            className="h-auto min-h-24 py-3"
            placeholder="z. B. neigt zu Koliken"
          />
        </Field>
        <Field label="Notfall-Medikament">
          <Input value={emergencyMedication} onChangeText={setEmergencyMedication} accessibilityLabel="Notfall-Medikament" />
        </Field>
        <Field label="Dauermedikation">
          <Input value={permanentMedication} onChangeText={setPermanentMedication} accessibilityLabel="Dauermedikation" />
        </Field>
        <Field label="Allergien">
          <Input value={allergies} onChangeText={setAllergies} accessibilityLabel="Allergien" />
        </Field>
        <Field label="Versicherung">
          <Input value={insurance} onChangeText={setInsurance} accessibilityLabel="Versicherung" />
        </Field>
      </Section>

      {problem || error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {problem ?? error}
        </Text>
      ) : null}
      <Button label={submitLabel} size="lg" fullWidth loading={saving} onPress={submit} />
    </>
  );
}
