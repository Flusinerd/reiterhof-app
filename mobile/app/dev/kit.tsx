import { Check, FileText, HeartPulse, Plus } from "lucide-react-native";
import { useState } from "react";
import { View } from "react-native";

import { Brand } from "@/components/brand";
import {
  Avatar,
  Badge,
  Button,
  Card,
  DateField,
  Divider,
  Icon,
  Input,
  LinkRow,
  LivePanel,
  NumberStepper,
  PageHeader,
  Pill,
  RangeStepper,
  Screen,
  Section,
  Sheet,
  Stepper,
  Switch,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Text,
  ToggleGroup,
  ToggleGroupItem,
} from "@/components/ui";
import { COLOR_KEYS } from "@/lib/color-keys";
import { colors } from "@/lib/theme";

// Hidden dev route (/dev/kit): shows every kit component for review.
export default function Kit() {
  const [tab, setTab] = useState("today");
  const [gait, setGait] = useState("trot");
  const [days, setDays] = useState<string[]>(["mon"]);
  const [filter, setFilter] = useState("all");
  const [on, setOn] = useState(true);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [code, setCode] = useState("");
  const [step, setStep] = useState(1);
  const [count, setCount] = useState(4);
  const [range, setRange] = useState({ min: 4, max: 5 });
  const [date, setDate] = useState("2026-05-17");

  return (
    <Screen back>
      <PageHeader
        eyebrow="Entwicklung"
        title="Komponenten"
        value="28"
        valueSize="sm"
        unit="Bausteine"
        description="Alle Bausteine des Design-Systems auf einen Blick."
        action={<Button size="icon" variant="outline" icon={Plus} accessibilityLabel="Hinzufügen" />}
      >
        <View className="flex-row flex-wrap gap-2">
          <Badge variant="primary" label="Seitenkopf" />
          <Badge label="ohne Fläche" />
        </View>
      </PageHeader>

      <Section title="Marke">
        <Brand />
      </Section>

      <Section title="Text">
        <Text variant="display">Seitentitel 32</Text>
        <Text variant="title">Titel 26</Text>
        <Text variant="heading">Überschrift 20</Text>
        <Text variant="heroNumberSm">44</Text>
        <Text variant="body">Fließtext 15 px</Text>
        <Text variant="bodySm">Fließtext 14 px</Text>
        <Text variant="secondary">Sekundärtext 13 px</Text>
        <Text variant="label">Feldbeschriftung 13 px</Text>
        <Text variant="caption">Beschriftung 12 px</Text>
      </Section>

      <Section title="Live-Anzeige" description="Nur für die laufende Aufzeichnung.">
        <LivePanel eyebrow="Ausritt" value="12:34">
          <View className="flex-row gap-4">
            {(
              [
                ["Schritt", colors.gait.walk],
                ["Trab", colors.gait.trot],
                ["Galopp", colors.gait.canter],
              ] as const
            ).map(([label, color]) => (
              <View key={label} className="flex-row items-center gap-2">
                <View className="h-3 w-3 rounded-pill" style={{ backgroundColor: color }} />
                <Text variant="secondary">{label}</Text>
              </View>
            ))}
          </View>
        </LivePanel>
      </Section>

      <Section title="Buttons">
        <Button label="Primär" icon={Check} />
        <Button label="Sekundär" variant="secondary" />
        <Button label="Umrandet" variant="outline" />
        <Button label="Ghost" variant="ghost" />
        <Button label="Löschen" variant="danger" />
        <Button label="Lädt" loading />
        <Button label="Deaktiviert" disabled />
        <View className="flex-row items-center gap-3">
          <Button label="Klein" size="sm" />
          <Button label="Groß" size="lg" />
          <Button size="icon" icon={Plus} variant="outline" accessibilityLabel="Hinzufügen" />
        </View>
      </Section>

      <Section title="Eingaben" description="Felder stehen direkt auf der Seite, ohne Karte.">
        <Input placeholder="E-Mail-Adresse" accessibilityLabel="E-Mail-Adresse" />
        <Input
          value={code}
          onChangeText={setCode}
          placeholder="123 456"
          accessibilityLabel="Code"
          keyboardType="number-pad"
          className="h-20 text-center font-display text-title-xl"
          style={{ letterSpacing: 6 }}
        />
        <Divider label="oder" />
        <Button label="Mit Google anmelden" variant="outline" fullWidth />
      </Section>

      <Section title="Badges und Pills">
        <View className="flex-row flex-wrap gap-2">
          <Badge label="Neutral" />
          <Badge variant="primary" label="Erledigt" />
          <Badge variant="accent" label="Dringend" />
          <Badge variant="info" label="Info" />
          <Badge variant="danger" label="Abgelehnt" />
        </View>
        <View className="flex-row flex-wrap gap-2">
          {[
            ["all", "Alle"],
            ["open", "Offen"],
            ["done", "Erledigt"],
          ].map(([value, label]) => (
            <Pill key={value} label={label} selected={filter === value} onPress={() => setFilter(value)} />
          ))}
        </View>
      </Section>

      <Section title="Avatare">
        <View className="flex-row flex-wrap gap-3">
          {COLOR_KEYS.map((key) => (
            <Avatar key={key} name={key} colorKey={key} />
          ))}
        </View>
        <View className="flex-row items-end gap-3">
          <Avatar name="Luna" colorKey="green" size="sm" />
          <Avatar name="Fanta" colorKey="amber" size="md" />
          <Avatar name="Balu" colorKey="blue" size="lg" />
          <Avatar name="Cookie" colorKey="rose" size="xl" />
        </View>
      </Section>

      <Section title="Tabs, Schalter, Gruppen">
        <Tabs value={tab} onValueChange={setTab}>
          <TabsList>
            <TabsTrigger value="today" label="Heute" />
            <TabsTrigger value="week" label="Woche" />
          </TabsList>
          <TabsContent value="today">
            <Text variant="secondary">Inhalt für „Heute“.</Text>
          </TabsContent>
          <TabsContent value="week">
            <Text variant="secondary">Inhalt für „Woche“.</Text>
          </TabsContent>
        </Tabs>
        <Switch label="Erinnerungen" description="Push am Morgen" value={on} onValueChange={setOn} />
        <ToggleGroup type="single" value={gait} onValueChange={setGait}>
          <ToggleGroupItem value="walk" label="Schritt" />
          <ToggleGroupItem value="trot" label="Trab" />
          <ToggleGroupItem value="canter" label="Galopp" />
        </ToggleGroup>
        <ToggleGroup type="multiple" value={days} onValueChange={setDays}>
          <ToggleGroupItem value="mon" label="Mo" />
          <ToggleGroupItem value="tue" label="Di" />
          <ToggleGroupItem value="wed" label="Mi" />
        </ToggleGroup>
      </Section>

      <Section title="Karten und Sheet" description="Karten nur für Objekte: ein Pferd, eine Anfrage, eine Liste.">
        <Card padded={false}>
          <View className="flex-row items-center gap-3 px-5 py-4">
            <Avatar name="Luna" colorKey="green" size="sm" />
            <Text variant="bodyStrong">Luna</Text>
          </View>
          <Divider />
          <View className="flex-row items-center gap-3 px-5 py-4">
            <Avatar name="Balu" colorKey="blue" size="sm" />
            <Text variant="bodyStrong">Balu</Text>
          </View>
        </Card>
        <Card shape="tile" className="flex-row items-center gap-3">
          <Icon as={Check} className="text-primary" />
          <Text variant="body">Kachel mit Radius 16</Text>
        </Card>
        <Card padded={false}>
          <LinkRow icon={FileText} label="Dokumente" description="3 Dateien" onPress={() => {}} />
          <Divider />
          <LinkRow icon={HeartPulse} label="Reha-Plan" onPress={() => {}} />
        </Card>
        <Button label="Sheet öffnen" variant="outline" onPress={() => setSheetOpen(true)} />
        <Sheet open={sheetOpen} onOpenChange={setSheetOpen} title="Neue Anfrage" description="Beispiel für ein Bottom-Sheet.">
          <Button label="Senden" fullWidth onPress={() => setSheetOpen(false)} />
        </Sheet>
      </Section>

      <Section title="Schritte und Datum">
        <Stepper steps={5} current={step} label="Aktivitäten" />
        <View className="flex-row gap-3">
          <Button label="Zurück" variant="outline" size="sm" disabled={step === 0} onPress={() => setStep((s) => s - 1)} />
          <Button label="Weiter" size="sm" disabled={step === 4} onPress={() => setStep((s) => s + 1)} />
        </View>
        <NumberStepper
          value={count}
          min={1}
          max={7}
          onChange={setCount}
          accessibilityLabel="Einheiten pro Woche"
          format={(v) => `${v}×`}
        />
        <RangeStepper
          label="Einheiten pro Woche"
          min={range.min}
          max={range.max}
          lowerBound={1}
          upperBound={7}
          onChange={(min, max) => setRange({ min, max })}
        />
        <DateField value={date} onChange={setDate} accessibilityLabel="Datum" clearable />
      </Section>
    </Screen>
  );
}
