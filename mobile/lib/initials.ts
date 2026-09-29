/** First letter of a name, upper-cased. Returns "?" for empty input. */
export function initialOf(name: string | null | undefined): string {
  const first = Array.from(name?.trim() ?? "")[0];
  return first ? first.toLocaleUpperCase("de-DE") : "?";
}
