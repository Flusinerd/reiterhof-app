# Gait detection (JAN-64)

Pure TypeScript (no React Native imports) that turns phone accelerometer
samples, and optionally GPS speed, into `halt | walk | trot | canter`. The files
run under `node --test` (Node 22 type stripping: no enums, parameter
properties or namespaces).

```ts
import { GaitStream, gaitShares, exportWindows } from "@/lib/gait";

const stream = new GaitStream(); // or { calibration, smoothing, windowMs, hopMs }
accelerometer.addListener(({ x, y, z }) => stream.push({ t: Date.now(), x, y, z })); // expo-sensors, ~50 Hz
location.onFix((f) => stream.pushGps({ t: f.timestamp, speedKmh: (f.speed ?? 0) * 3.6 })); // before the samples of that instant

stream.windows;                  // all finished GaitWindow objects
gaitShares(stream.windows);      // minutes and percent per gait
exportWindows(stream.windows);   // compact records for storage / later training
```

`analyzeWindows(samples, gpsFixes, options)` does the same for a finished recording.

## Pipeline

1. **Windows**: 4 s, hop 2 s (50 % overlap). Each window is resampled linearly to
   50 Hz; a window with less than 60 % of the expected samples or a gap over
   500 ms is skipped (no result, not a guess).
2. **Gravity**: the mean vector of the window is the gravity estimate. The
   acceleration is split into a *vertical* part (along that direction) and the
   *horizontal* rest, then the vertical part is detrended. This works for any
   phone orientation and needs no orientation sensor.
3. **Features** (`Features`): dominant frequency (radix-2 FFT of the Hann-windowed
   vertical signal, zero padded to 512 points, parabolic peak interpolation,
   searched in 0.6-3.5 Hz), vertical and horizontal RMS energy, alternating-peak
   asymmetry (mean `1 - small/large` over consecutive peaks), 2nd-harmonic ratio,
   regularity (share of the 0.5-4 Hz power around the peak).
4. **Classification** (`classifyWindow`):
   halt if vertical energy is below 0.04 g or the spectrum is not peaky
   (regularity < 0.5); walk below 1.25 Hz; trot from 1.25 to 1.6 Hz; canter
   above 1.85 Hz; in the trot/canter overlap (1.6-1.85 Hz) the shape decides:
   `max(asymmetry, harmonicRatio) > 0.35` is canter. The result carries a
   confidence (distance of the deciding feature from its boundary).
5. **GPS fusion** (`fuseWithSpeed`): speed classes are halt < 1.5, walk < 7,
   trot 7-16, canter > 16 km/h. If the accelerometer and the speed class agree,
   that gait is used. Otherwise the source with the higher confidence wins: the
   accelerometer confidence from step 4, versus a speed confidence that is 0 on
   a class boundary and grows linearly to 1 at 3 km/h inside the class (for halt:
   1 at 0 km/h). Ties go to the accelerometer. So speed wins when the
   accelerometer is ambiguous or clearly contradicts a speed deep in another
   class, but a confident accelerometer result survives a speed near a boundary
   or a bad fix. Without a speed (indoors, no fix) only the accelerometer counts.
   The window speed is the mean of the GPS fixes inside the window (or the last
   fix at most 3 s before it).
6. **Smoothing** (`majorityVote`): majority vote over the last N fused windows
   (default 3, ties go to the most recent gait). Costs up to two hops (4 s) of
   delay at a real gait change.

## Calibration

`Calibration` holds the thresholds; `DEFAULT_CALIBRATION` works without any
learning. `learnCalibration(segments)` takes short labelled recordings
(`{ gait, samples }`, 20-30 s each, any subset of gaits) made with the phone at
its usual place and returns a `Calibration` with:
`energyScale` (amplitude relative to the reference), `haltEnergy` (between the
standing noise and the weakest moving window), `walkTrotFreq` and
`asymmetryThreshold` (midpoints between the labelled classes). Anything that
cannot be learned keeps its default. On the synthetic "phone on the arm" data
(signal at 25 %) the default calibration calls every walk a halt; the learned
one restores full accuracy.

## Export and summaries

- `exportWindows(windows)` returns `WindowRecord`s: `t` (window start), `f` (the
  features in `FEATURE_ORDER`, 3 decimals), `p` (predicted gait), `a`
  (accelerometer-only gait), `v` (GPS km/h), `c` (user-corrected label).
  `correctRecords(records, from, to, gait)` sets the corrected label for a time
  range; `toJsonLines` serialises. About 100 bytes per window.
- `gaitShares(windows)` returns minutes and percent per gait (each window counts
  for its hop length, so the overlap is not counted twice; corrected labels win;
  `excludeHalt` makes the percentages describe the riding only).

## Limitations - read before relying on this

- **Everything here is validated on synthetic data only** (`synth.ts`: sinusoids
  with harmonics, slow frequency/amplitude modulation, Gaussian noise, random
  phone orientation, timestamp jitter). The model encodes the spec's frequency
  ranges and my assumption that the canter waveform has a strong 2nd harmonic.
  Real horses, riders, saddles, footing and phone placements will differ. The
  accuracy figures below say the code implements the design, not that the design
  works on real horses.
- Needs real-world validation with labelled rides (walk/trot/canter, several
  horses, pocket vs arm vs jacket, indoor arena vs outdoors) before any threshold
  is trusted. The reference energies in `calibration.ts` and all thresholds in
  `DEFAULT_CALIBRATION` are placeholders derived from the synthetic model.
- Rising trot, sitting trot, lateral work, tölt/pace horses, transitions within a
  window, a rider dismounting or leading the horse, and the phone being handled
  are not modelled.
- Aperiodic movement (mucking out, phone in hand) is only partly rejected: broad
  band noise occasionally produces a peaky 4 s spectrum. In the synthetic test
  at least 60 % of such windows come out as halt; heavy sensor noise pushes a
  standing horse to about 79 % correct.
- Trot and canter overlap (1.6-1.85 Hz); there only the waveform shape or GPS
  speed can separate them, and canter is missed in about 5 % of synthetic windows
  (weak 2nd harmonic or a stride frequency below 1.6 Hz).
- GPS speed is noisy at low speed and not available indoors.

## Accuracy on synthetic data

100 recordings of 40 s per gait (6400 windows, first 2 windows of each recording
excluded as the vote has no history), random seeds, default settings:

| Setup | Accelerometer only | With smoothing | halt / walk / trot / canter (raw) |
| --- | --- | --- | --- |
| Default placement, noise 0.02 g | 98.8 % | 99.0 % | 100 / 100 / 100 / 95.3 |
| Noise 0.1 g | 93.5 % | 96.7 % | 78.6 / 99.9 / 100 / 95.5 |
| Noise 0.2 g | 91.1 % | 95.8 % | 79.1 / 89.9 / 100 / 95.4 |
| Arm (25 % signal), default calibration | 74.0 % | 74.0 % | 100 / 0.8 / 100 / 95.4 |
| Arm, calibrated from 20 s per gait | 98.8 % | 99.0 % | 100 / 99.8 / 100 / 95.4 |

Files: `types.ts`, `fft.ts`, `features.ts`, `classify.ts`, `fusion.ts`,
`detector.ts`, `calibration.ts`, `export.ts`, `shares.ts`, `index.ts`;
`synth.ts` is the test signal generator; tests in `gait.test.ts`.
