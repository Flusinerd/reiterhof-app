// Tiny Markdown parser for the legal texts (no React Native imports, unit-tested).
// Supported: `#`/`##`/`###` headings, paragraphs, `- ` bullet lists, `> ` quotes and
// `**bold**` inline. Nothing else is used in docs/legal/*.md; keep it that way.

export type Span = { text: string; bold: boolean };

export type Block =
  | { type: "heading"; level: 1 | 2 | 3; text: string }
  | { type: "paragraph"; spans: Span[] }
  | { type: "quote"; spans: Span[] }
  | { type: "list"; items: Span[][] };

/** Splits `a **b** c` into spans. An unmatched `**` stays literal. */
export function parseInline(text: string): Span[] {
  const spans: Span[] = [];
  const parts = text.split("**");
  // An odd number of parts means the markers pair up completely.
  const balanced = parts.length % 2 === 1;
  if (!balanced) return text === "" ? [] : [{ text, bold: false }];
  parts.forEach((part, i) => {
    if (part === "") return;
    spans.push({ text: part, bold: i % 2 === 1 });
  });
  return spans;
}

export function parseMarkdown(markdown: string): Block[] {
  const blocks: Block[] = [];
  let paragraph: string[] = [];
  let quote: string[] = [];
  let list: Span[][] | null = null;

  const flush = () => {
    if (paragraph.length > 0) blocks.push({ type: "paragraph", spans: parseInline(paragraph.join(" ")) });
    if (quote.length > 0) blocks.push({ type: "quote", spans: parseInline(quote.join(" ")) });
    if (list) blocks.push({ type: "list", items: list });
    paragraph = [];
    quote = [];
    list = null;
  };

  for (const raw of markdown.replace(/\r\n/g, "\n").split("\n")) {
    const line = raw.trimEnd();
    if (line.trim() === "") {
      flush();
      continue;
    }
    const heading = /^(#{1,3})\s+(.*)$/.exec(line);
    if (heading) {
      flush();
      blocks.push({ type: "heading", level: heading[1].length as 1 | 2 | 3, text: heading[2].trim() });
      continue;
    }
    const bullet = /^\s*-\s+(.*)$/.exec(line);
    if (bullet) {
      if (!list) {
        flush();
        list = [];
      }
      list.push(parseInline(bullet[1].trim()));
      continue;
    }
    const q = /^>\s?(.*)$/.exec(line);
    if (q) {
      if (quote.length === 0) flush();
      quote.push(q[1].trim());
      continue;
    }
    if (list) flush(); // a plain line ends a list
    paragraph.push(line.trim());
  }
  flush();
  return blocks;
}
