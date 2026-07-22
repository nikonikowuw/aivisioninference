import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Box, Spinner, Center, Text, Badge } from '@chakra-ui/react';
import { streamManager } from './StreamManager';
import type { Protocol } from './StreamManager';
import { useThrottledInference } from 'hooks/useThrottledInference';

interface FallbackConfig {
  fallbackOrder: Protocol[];
  webrtcTimeout: number;
  showProtocol: boolean;
}

const DEFAULT_FALLBACK_CONFIG: FallbackConfig = {
  fallbackOrder: ['webrtc', 'hls', 'flv'],
  webrtcTimeout: 10000,
  showProtocol: false,
};

interface VideoPlayerProps {
  url: string;
  protocol?: Protocol;
  poster?: string;
  videoRef?: React.RefObject<HTMLVideoElement>;
  onError?: (error: string) => void;
  onProtocolChange?: (protocol: Protocol) => void;
  fallbackConfig?: Partial<FallbackConfig>;
  deviceId?: string;
}

// ==================== URL 工具 ====================

function detectProtocol(url: string): Protocol {
  if (url.startsWith('webrtc://')) return 'webrtc';
  if (url.includes('.m3u8')) return 'hls';
  if (url.includes('.flv')) return 'flv';
  return 'hls';
}

function isWebRTCSupported(): boolean {
  return !!(window.RTCPeerConnection);
}

/** 转换不同协议的播放 URL (WebRTC / HLS / FLV) */
function convertUrl(url: string, targetProtocol: Protocol): string | null {
  const currentProtocol = detectProtocol(url);
  if (currentProtocol === targetProtocol) return url;

  try {
    const httpUrl = url.startsWith('webrtc://') ? url.replace('webrtc://', 'http://') : url;
    const urlObj = new URL(httpUrl);
    const pathParts = urlObj.pathname.split('/').filter(Boolean);
    if (pathParts.length < 2) return null;
    const app = pathParts[0];
    let stream = pathParts[1];
    stream = stream.replace(/\.m3u8$/, '').replace(/\.flv$/, '');

    const token = urlObj.searchParams.get('token');
    const query = token ? `?token=${token}` : '';
    const host = urlObj.host;

    switch (targetProtocol) {
      case 'webrtc': return `webrtc://${host}/${app}/${stream}${query}`;
      case 'hls': return `http://${host}/${app}/${stream}/hls.m3u8${query}`;
      case 'flv': return `http://${host}/${app}/${stream}.flv${query}`;
      default: return null;
    }
  } catch { return null; }
}

const PROTOCOL_COLORS: Record<string, string> = {
  webrtc: 'green',
  hls: 'blue',
  flv: 'orange',
};

function getProtocolColor(p: Protocol): string {
  return PROTOCOL_COLORS[p] || 'gray';
}

// ==================== 主组件 ====================

const VideoPlayer: React.FC<VideoPlayerProps> = ({
  url,
  protocol: forcedProtocol,
  poster,
  videoRef: externalVideoRef,
  onError,
  onProtocolChange,
  fallbackConfig,
  deviceId,
}) => {
  const config = React.useMemo(() => ({ ...DEFAULT_FALLBACK_CONFIG, ...fallbackConfig }), [fallbackConfig]);
  const internalVideoRef = useRef<HTMLVideoElement>(null);
  const videoRef = externalVideoRef || internalVideoRef;
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const cleanupRef = useRef<(() => void) | null>(null);
  const mountedRef = useRef(true);

  useThrottledInference(deviceId, videoRef, canvasRef);

  const [currentProtocol, setCurrentProtocol] = useState<Protocol | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [fallbackInfo, setFallbackInfo] = useState<string | null>(null);

  const cleanup = useCallback(() => {
    cleanupRef.current?.();
    cleanupRef.current = null;
  }, []);

  const tryNextProtocolRef = useRef<(attemptIndex: number, originalUrl: string) => void>(() => {});

  const tryPlay = useCallback(
    async (targetProtocol: Protocol, originalUrl: string, attemptIndex: number) => {
      const video = videoRef.current;
      if (!video || !mountedRef.current) return;

      cleanup();

      const targetUrl = convertUrl(originalUrl, targetProtocol);
      if (!targetUrl) {
        tryNextProtocolRef.current(attemptIndex + 1, originalUrl);
        return;
      }

      setCurrentProtocol(targetProtocol);
      setError(null);
      setFallbackInfo(attemptIndex > 0 ? `降级自 ${config.fallbackOrder[attemptIndex - 1]}` : null);
      onProtocolChange?.(targetProtocol);

      if (targetProtocol === 'flv') {
        // FLV 不支持共享，独立播放
        let destroyed = false;
        import('flv.js').then(({ default: Flv }) => {
          if (destroyed || !mountedRef.current) return;
          if (Flv.isSupported()) {
            const flv = Flv.createPlayer({ type: 'flv', url: targetUrl });
            flv.attachMediaElement(video);
            flv.load();
            video.onplaying = () => {
              if (mountedRef.current) setLoading(false);
            };
            flv.on(Flv.Events.ERROR, (errType: string, errDetail: string) => {
              console.warn(`[VideoPlayer] flv 失败 (${errType}): ${errDetail}`);
              if (!destroyed && mountedRef.current) {
                tryNextProtocolRef.current(attemptIndex + 1, originalUrl);
              }
            });
            const playResult = flv.play();
            if (playResult) playResult.catch(() => {});
            cleanupRef.current = () => {
              destroyed = true;
              try {
                flv.pause();
                flv.unload();
                flv.detachMediaElement();
                flv.destroy();
              } catch {}
            };
          } else {
            tryNextProtocolRef.current(attemptIndex + 1, originalUrl);
          }
        }).catch(() => {
          if (!destroyed) tryNextProtocolRef.current(attemptIndex + 1, originalUrl);
        });
        return;
      }

      if (targetProtocol === 'hls') {
        // HLS 直接绑定可见 video，避免 hidden video -> captureStream -> visible video
        // 造成额外的解码、合成和固定 30 FPS 重采样。
        let destroyed = false;
        let hlsInstance: import('hls.js').default | null = null;
        let mediaRecoveryAttempted = false;

        cleanupRef.current = () => {
          destroyed = true;
          if (hlsInstance) {
            hlsInstance.destroy();
            hlsInstance = null;
          }
          video.removeAttribute('src');
          video.load();
        };

        import('hls.js').then(({ default: Hls }) => {
          if (destroyed || !mountedRef.current) return;

          const fallback = (detail: string) => {
            if (destroyed || !mountedRef.current) return;
            console.warn(`[VideoPlayer] hls 失败: ${detail}`);
            tryNextProtocolRef.current(attemptIndex + 1, originalUrl);
          };

          if (Hls.isSupported()) {
            hlsInstance = new Hls({
              liveSyncDurationCount: 1,
              liveMaxLatencyDurationCount: 3,
              maxLiveSyncPlaybackRate: 1.5,
            });
            hlsInstance.attachMedia(video);
            hlsInstance.on(Hls.Events.MEDIA_ATTACHED, () => hlsInstance?.loadSource(targetUrl));
            hlsInstance.on(Hls.Events.MANIFEST_PARSED, () => {
              video.play().catch(() => {});
            });
            hlsInstance.on(Hls.Events.ERROR, (_event, data) => {
              if (!data.fatal) return;
              if (data.type === Hls.ErrorTypes.MEDIA_ERROR && !mediaRecoveryAttempted) {
                mediaRecoveryAttempted = true;
                hlsInstance?.recoverMediaError();
                return;
              }
              fallback(data.details || data.type || 'unknown');
            });
          } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
            video.src = targetUrl;
            video.play().catch(() => {});
            video.onerror = () => fallback(video.error?.message || 'native playback error');
          } else {
            fallback('unsupported');
          }

          video.onplaying = () => {
            if (mountedRef.current) setLoading(false);
          };
        }).catch(() => {
          if (!destroyed) tryNextProtocolRef.current(attemptIndex + 1, originalUrl);
        });
        return;
      }

      // WebRTC 走 StreamManager 复用连接。
      if (targetProtocol === 'webrtc' && !isWebRTCSupported()) {
        tryNextProtocolRef.current(attemptIndex + 1, originalUrl);
        return;
      }

      try {
        const result = await streamManager.subscribe(targetUrl, video, {
          webrtcTimeout: config.webrtcTimeout,
          onDisconnect: () => tryNextProtocolRef.current(attemptIndex + 1, originalUrl),
        });

        if (!mountedRef.current) {
          result.destroy();
          return;
        }

        setLoading(false);
        cleanupRef.current = result.destroy;
      } catch (err) {
        console.warn(`[VideoPlayer] ${targetProtocol} 失败:`, err);
        tryNextProtocolRef.current(attemptIndex + 1, originalUrl);
      }
    },
    [cleanup, config, onProtocolChange],
  );

  const tryNextProtocol = useCallback(
    (attemptIndex: number, originalUrl: string) => {
      if (attemptIndex >= config.fallbackOrder.length) {
        const finalError = `所有协议均失败: ${config.fallbackOrder.join(' → ')}`;
        setError(finalError);
        setLoading(false);
        onError?.(finalError);
        return;
      }
      tryPlay(config.fallbackOrder[attemptIndex], originalUrl, attemptIndex);
    },
    [config, onError, tryPlay],
  );

  useEffect(() => {
    tryNextProtocolRef.current = tryNextProtocol;
  }, [tryNextProtocol]);

  useEffect(() => {
    if (!url) return;

    mountedRef.current = true;
    setLoading(true);
    setError(null);
    setFallbackInfo(null);

    const startProtocol = forcedProtocol || detectProtocol(url);
    const startIndex = config.fallbackOrder.indexOf(startProtocol);
    const effectiveIndex = startIndex >= 0 ? startIndex : 0;

    tryPlay(config.fallbackOrder[effectiveIndex], url, effectiveIndex);

    return () => {
      mountedRef.current = false;
      cleanup();
    };
  }, [url, forcedProtocol]);

  return (
    <Box position="relative" w="100%" h="100%" bg="black" borderRadius="md" overflow="hidden">
      <video
        ref={videoRef}
        style={{ width: '100%', height: '100%', objectFit: 'contain' }}
        controls
        poster={poster}
        autoPlay
        playsInline
        muted
      />
      <canvas
        ref={canvasRef}
        style={{
          position: 'absolute',
          top: 0,
          left: 0,
          width: '100%',
          height: '100%',
          pointerEvents: 'none',
          zIndex: 1,
        }}
      />
      {loading && !error && (
        <Center position="absolute" top="0" left="0" w="100%" h="100%">
          <Spinner color="white" size="xl" />
        </Center>
      )}
      {error && (
        <Center position="absolute" top="0" left="0" w="100%" h="100%">
          <Text color="red.300" fontSize="sm" textAlign="center" px={4}>{error}</Text>
        </Center>
      )}
      {config.showProtocol && currentProtocol && !error && (
        <Badge position="absolute" top={2} right={2}
          colorScheme={getProtocolColor(currentProtocol)}
          fontSize="xs">{currentProtocol.toUpperCase()}</Badge>
      )}
      {fallbackInfo && !error && (
        <Badge position="absolute" top={2} left={2} colorScheme="yellow" fontSize="xs">{fallbackInfo}</Badge>
      )}
    </Box>
  );
};

export default VideoPlayer;
