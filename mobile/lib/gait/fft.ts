// Small radix-2 FFT. In place, iterative Cooley-Tukey.

export function nextPow2(n: number): number {
  let p = 1;
  while (p < n) p <<= 1;
  return p;
}

/** In-place complex FFT. `re.length` must be a power of two and equal `im.length`. */
export function fft(re: Float64Array, im: Float64Array): void {
  const n = re.length;
  if (n !== im.length || n < 1 || (n & (n - 1)) !== 0) {
    throw new Error("fft: length must be a power of two");
  }
  // Bit reversal permutation.
  for (let i = 1, j = 0; i < n; i++) {
    let bit = n >> 1;
    for (; j & bit; bit >>= 1) j ^= bit;
    j ^= bit;
    if (i < j) {
      let t = re[i]!;
      re[i] = re[j]!;
      re[j] = t;
      t = im[i]!;
      im[i] = im[j]!;
      im[j] = t;
    }
  }
  for (let len = 2; len <= n; len <<= 1) {
    const ang = (-2 * Math.PI) / len;
    const wRe = Math.cos(ang);
    const wIm = Math.sin(ang);
    for (let i = 0; i < n; i += len) {
      let cRe = 1;
      let cIm = 0;
      for (let k = 0; k < len / 2; k++) {
        const a = i + k;
        const b = a + len / 2;
        const tRe = re[b]! * cRe - im[b]! * cIm;
        const tIm = re[b]! * cIm + im[b]! * cRe;
        re[b] = re[a]! - tRe;
        im[b] = im[a]! - tIm;
        re[a] = re[a]! + tRe;
        im[a] = im[a]! + tIm;
        const nRe = cRe * wRe - cIm * wIm;
        cIm = cRe * wIm + cIm * wRe;
        cRe = nRe;
      }
    }
  }
}

/**
 * One-sided amplitude spectrum of a real signal (Hann window, zero padded to
 * `size`). Bin k is at k * sampleRate / size Hz. Amplitude is scaled so that a
 * sinusoid of amplitude A yields about A.
 */
export function amplitudeSpectrum(signal: ArrayLike<number>, size: number): Float64Array {
  const n = signal.length;
  const re = new Float64Array(size);
  const im = new Float64Array(size);
  let wSum = 0;
  for (let i = 0; i < n; i++) {
    const w = 0.5 - 0.5 * Math.cos((2 * Math.PI * i) / Math.max(1, n - 1));
    re[i] = signal[i]! * w;
    wSum += w;
  }
  fft(re, im);
  const out = new Float64Array(size / 2 + 1);
  for (let k = 0; k < out.length; k++) {
    out[k] = wSum > 0 ? (2 * Math.hypot(re[k]!, im[k]!)) / wSum : 0;
  }
  return out;
}
