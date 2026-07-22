/**
 * H.265 (HEVC) WebCodecs & Native Decoder Compatibility Detection
 */

export async function isH265WebCodecsSupported(): Promise<boolean> {
  if (typeof window === 'undefined' || !('VideoDecoder' in window)) {
    return false;
  }
  try {
    const support = await VideoDecoder.isConfigSupported({
      codec: 'hev1.1.6.L93.B0', // H.265 Main Profile Level 3.1
      codedWidth: 1920,
      codedHeight: 1080,
    });
    return !!support.supported;
  } catch {
    return false;
  }
}

export function isH265NativeVideoSupported(): boolean {
  if (typeof document === 'undefined') return false;
  const video = document.createElement('video');
  const type1 = video.canPlayType('video/mp4; codecs="hev1.1.6.L93.B0"');
  const type2 = video.canPlayType('video/mp4; codecs="hvc1.1.6.L93.B0"');
  return type1 === 'probably' || type1 === 'maybe' || type2 === 'probably' || type2 === 'maybe';
}
