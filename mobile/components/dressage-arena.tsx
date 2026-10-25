import { useMemo } from "react";
import Svg, { Circle, Line, Path, Rect, Text as SvgText } from "react-native-svg";

import { arenaLength, arenaLetters, fitArena, letterPosition, pathToSvg, type ArenaSize, type PlanLine, type Point } from "@/lib/dressage";
import { useColors } from "@/lib/theme";

export type DressageArenaProps = {
  arena: ArenaSize;
  lines: readonly PlanLine[];
  /** Index of the highlighted line; null shows only the faint full path. */
  selected: number | null;
  /** Position of the playback dot in metres. */
  dot?: Point | null;
  /** Width of the box in pixels; the height follows (at most `maxHeight`). */
  width: number;
  maxHeight?: number;
};

/**
 * The arena from above (A at the bottom): fence, centre line, letters in Fraunces, the whole path of the
 * test faint and the selected line in brass. Pure drawing, no touch handling (the lines below select).
 */
export function DressageArena({ arena, lines, selected, dot, width, maxHeight = 520 }: DressageArenaProps) {
  const c = useColors();
  const fit = useMemo(() => fitArena(arena, width, maxHeight), [arena, width, maxHeight]);
  const letters = useMemo(() => arenaLetters(arena), [arena]);
  const all = useMemo(() => pathToSvg(lines.flatMap((l) => l.path), fit), [lines, fit]);
  const current = selected !== null ? lines[selected] : undefined;
  const highlight = current ? pathToSvg(current.path, fit) : "";
  const [left, top] = fit.toPx(0, arenaLength(arena));
  const [cx0, cy0] = fit.toPx(10, 0);
  const [, cy1] = fit.toPx(10, arenaLength(arena));
  const dotPx = dot ? fit.toPx(dot[0], dot[1]) : null;
  const text = lines.length > 0 ? `Viereck ${arena === "20x60" ? "20×60" : "20×40"} mit ${lines.length} Linien` : "Viereck";

  return (
    <Svg width={fit.width} height={fit.height} accessibilityLabel={text} accessible>
      <Rect x={left} y={top} width={20 * fit.scale} height={arenaLength(arena) * fit.scale} fill={c.card} stroke={c.border} strokeWidth={1.5} rx={3} />
      <Line x1={cx0} y1={cy0} x2={cx0} y2={cy1} stroke={c.divider} strokeWidth={1} strokeDasharray="3 5" />
      {all ? <Path d={all} stroke={c.muted} strokeOpacity={0.35} strokeWidth={1.5} fill="none" strokeLinejoin="round" strokeLinecap="round" /> : null}
      {highlight ? <Path d={highlight} stroke={c.brass.DEFAULT} strokeWidth={4} fill="none" strokeLinejoin="round" strokeLinecap="round" /> : null}
      {letters.map((l) => {
        const [x, y] = letterPosition(fit, l, l.side === "centre" ? 0 : 1.5);
        const inside = l.side === "centre";
        return (
          <SvgText
            key={l.letter}
            x={x}
            y={y + 4}
            fontSize={inside ? 11 : 13}
            fontFamily="Fraunces_600SemiBold"
            fill={inside ? c.muted : c.foreground}
            textAnchor="middle"
          >
            {l.letter}
          </SvgText>
        );
      })}
      {dotPx ? <Circle cx={dotPx[0]} cy={dotPx[1]} r={7} fill={c.primary.DEFAULT} stroke={c.card} strokeWidth={2} /> : null}
    </Svg>
  );
}
