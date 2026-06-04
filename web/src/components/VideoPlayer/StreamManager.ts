/**
 * StreamManager - 流复用管理器
 *
 * 核心逻辑：
 * 1. 同一个 URL 只建立一次连接
 * 2. HLS/WebRTC 均支持多 video 共享同一帧画面
 * 3. 引用计数：所有订阅者移除后才关闭连接
 *
 * HLS 复用原理：
 *   - 一个隐藏的 <video> 播放 HLS 流
 *   - 通过 video.captureStream() 获取 MediaStream
 *   - 所有可见 <video> 共享该 MediaStream，画面完全同步
 */

type Protocol = 'webrtc' | 'hls' | 'flv';

interface StreamEntry {
  id: string;
  url: string;
  protocol: Protocol;
  /** 共享的 MediaStream（captureStream 产生） */
  stream: MediaStream | null;
  /** 隐藏的源 video 元素（HLS 用） */
  sourceVideo: HTMLVideoElement | null;
  /** 播放器清理函数 */
  destroy: () => void;
  /** 引用计数 */
  refCount: number;
  /** 订阅的 video 元素 */
  subscribers: Set<HTMLVideoElement>;
  status: 'loading' | 'ready' | 'error';
  error: string | null;
  waitQueue: Array<{
    resolve: (stream: MediaStream) => void;
    reject: (error: string) => void;
  }>;
}

/** 生成流的唯一 ID（去除 token 参数的 URL 做 hash） */
function generateStreamId(url: string): string {
  const cleanUrl = url.replace(/[?&]token=[^&]+/, '');
  let hash = 0;
  for (let i = 0; i < cleanUrl.length; i++) {
    const char = cleanUrl.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash |= 0;
  }
  return 's' + Math.abs(hash).toString(36);
}

class StreamManager {
  private streams: Map<string, StreamEntry> = new Map();
  private static instance: StreamManager | null = null;

  static getInstance(): StreamManager {
    if (!StreamManager.instance) {
      StreamManager.instance = new StreamManager();
    }
    return StreamManager.instance;
  }

  /**
   * 订阅一个流，返回共享的 MediaStream。
   * 同一个 URL 只会建立一次连接，所有订阅者共享同一帧画面。
   */
  async subscribe(
    url: string,
    protocol: Protocol,
    video: HTMLVideoElement,
    options?: { webrtcTimeout?: number },
  ): Promise<{ stream: MediaStream | null; destroy: () => void }> {
    const id = generateStreamId(url);
    let entry = this.streams.get(id);

    if (entry) {
      entry.refCount++;
      entry.subscribers.add(video);
      console.log(`[StreamManager] 复用流: ${id}, 引用: ${entry.refCount}`);

      if (entry.status === 'ready' && entry.stream) {
        video.srcObject = entry.stream;
        video.play().catch(() => {});
        return { stream: entry.stream, destroy: () => this.unsubscribe(id, video) };
      }

      if (entry.status === 'loading') {
        return new Promise((resolve, reject) => {
          entry!.waitQueue.push({
            resolve: (stream) => {
              video.srcObject = stream;
              video.play().catch(() => {});
              resolve({ stream, destroy: () => this.unsubscribe(id, video) });
            },
            reject,
          });
        });
      }

      if (entry.status === 'error') {
        // 重置，重新尝试
        entry.destroy();
        this.streams.delete(id);
        return this.subscribe(url, protocol, video, options);
      }
    }

    // 创建新流
    entry = {
      id,
      url,
      protocol,
      stream: null,
      sourceVideo: null,
      destroy: () => {},
      refCount: 1,
      subscribers: new Set([video]),
      status: 'loading',
      error: null,
      waitQueue: [],
    };
    this.streams.set(id, entry);

    try {
      if (protocol === 'webrtc') {
        await this.connectWebrtc(entry, options?.webrtcTimeout || 10000);
      } else {
        await this.connectHls(entry);
      }
    } catch (error) {
      entry.status = 'error';
      entry.error = error instanceof Error ? error.message : String(error);
      entry.waitQueue.forEach((w) => w.reject(entry!.error!));
      entry.waitQueue = [];
      throw error;
    }

    // 将共享流绑定到当前 video
    if (entry.stream) {
      video.srcObject = entry.stream;
      video.play().catch(() => {});
    }

    return { stream: entry.stream, destroy: () => this.unsubscribe(id, video) };
  }

  private unsubscribe(id: string, video: HTMLVideoElement): void {
    const entry = this.streams.get(id);
    if (!entry) return;

    entry.subscribers.delete(video);
    entry.refCount--;
    video.srcObject = null;

    if (entry.refCount <= 0) {
      console.log(`[StreamManager] 销毁流: ${id}`);
      entry.destroy();
      this.streams.delete(id);
    }
  }

  // ==================== WebRTC 连接 ====================

  private async connectWebrtc(entry: StreamEntry, timeout: number): Promise<void> {
    return new Promise(async (resolve, reject) => {
      const loaded = await this.loadZLMRTCClient();
      if (!loaded) {
        entry.status = 'error';
        entry.error = 'Failed to load ZLMRTCClient.js';
        reject(new Error(entry.error));
        return;
      }

      const ZLMRTCClient = (window as any).ZLMRTCClient;
      const zlmsdpUrl = this.buildZlmWebrtcApiUrl(entry.url);
      if (!zlmsdpUrl) {
        entry.status = 'error';
        entry.error = 'Invalid WebRTC URL';
        reject(new Error(entry.error));
        return;
      }

      // 隐藏 video 用于 WebRTC 连接
      const tempVideo = document.createElement('video');
      tempVideo.style.display = 'none';
      tempVideo.muted = true;
      tempVideo.playsInline = true;
      document.body.appendChild(tempVideo);

      let resolved = false;
      let player: any = null;

      const timeoutTimer = setTimeout(() => {
        if (!resolved) {
          resolved = true;
          tempVideo.remove();
          entry.status = 'error';
          entry.error = 'WebRTC connection timeout';
          reject(new Error(entry.error));
        }
      }, timeout);

      try {
        player = new ZLMRTCClient.Endpoint({
          element: tempVideo,
          debug: false,
          zlmsdpUrl,
          recvOnly: true,
          audioEnable: false,
          videoEnable: true,
          usedatachannel: false,
        });
      } catch (e: any) {
        clearTimeout(timeoutTimer);
        tempVideo.remove();
        entry.status = 'error';
        entry.error = e.message;
        reject(e);
        return;
      }

      entry.destroy = () => {
        clearTimeout(timeoutTimer);
        if (player) { try { player.close(); } catch {} }
        tempVideo.remove();
      };

      player.on(ZLMRTCClient.Events.WEBRTC_ON_REMOTE_STREAMS, () => {
        if (resolved) return;
        resolved = true;
        clearTimeout(timeoutTimer);

        // WebRTC 直接把 video 的 stream 共享
        const stream = tempVideo.srcObject as MediaStream;
        entry.stream = stream;
        entry.status = 'ready';

        entry.waitQueue.forEach((w) => w.resolve(stream!));
        entry.waitQueue = [];
        resolve();
      });

      const handleError = (msg: string) => {
        if (resolved) return;
        resolved = true;
        clearTimeout(timeoutTimer);
        tempVideo.remove();
        entry.status = 'error';
        entry.error = msg;
        reject(new Error(msg));
      };

      player.on(ZLMRTCClient.Events.WEBRTC_ICE_CANDIDATE_ERROR, () => handleError('WebRTC ICE failed'));
      player.on(ZLMRTCClient.Events.WEBRTC_OFFER_ANWSER_EXCHANGE_FAILED, (e: any) =>
        handleError(`WebRTC SDP failed: ${e?.msg || 'unknown'}`),
      );
      player.on(ZLMRTCClient.Events.WEBRTC_ON_CONNECTION_STATE_CHANGE, (state: string) => {
        if (state === 'failed' || state === 'disconnected') {
          handleError('WebRTC connection lost');
        }
      });
    });
  }

  // ==================== HLS 连接（单源多副本） ====================

  private async connectHls(entry: StreamEntry): Promise<void> {
    return new Promise((resolve, reject) => {
      // 创建隐藏 video 播放 HLS
      const hiddenVideo = document.createElement('video');
      hiddenVideo.style.display = 'none';
      hiddenVideo.muted = true;
      hiddenVideo.playsInline = true;
      hiddenVideo.setAttribute('playsinline', '');
      document.body.appendChild(hiddenVideo);
      entry.sourceVideo = hiddenVideo;

      let hlsInstance: any = null;
      let resolved = false;

      entry.destroy = () => {
        if (hlsInstance) {
          hlsInstance.destroy();
          hlsInstance = null;
        }
        hiddenVideo.remove();
      };

      // 加载 HLS.js（如果原生支持则直接播放）
      import('hls.js').then(({ default: Hls }) => {
        if (Hls.isSupported()) {
          hlsInstance = new Hls();
          hlsInstance.loadSource(entry.url);
          hlsInstance.attachMedia(hiddenVideo);

          hlsInstance.on(Hls.Events.MANIFEST_PARSED, () => {
            hiddenVideo.play().then(() => {
              // HLS 开始播放后，用 captureStream 获取 MediaStream
              this.shareViaCaptureStream(entry, hiddenVideo, resolved, resolve, reject);
              resolved = true;
            }).catch((e) => {
              if (!resolved) {
                resolved = true;
                reject(e);
              }
            });
          });

          hlsInstance.on(Hls.Events.ERROR, (_event: any, data: any) => {
            if (data.fatal && !resolved) {
              resolved = true;
              reject(new Error('HLS playback failed'));
            }
          });
        } else if (hiddenVideo.canPlayType('application/vnd.apple.mpegurl')) {
          // Safari 原生 HLS
          hiddenVideo.src = entry.url;
          hiddenVideo.addEventListener('loadedmetadata', () => {
            hiddenVideo.play().then(() => {
              this.shareViaCaptureStream(entry, hiddenVideo, resolved, resolve, reject);
              resolved = true;
            }).catch((e) => {
              if (!resolved) {
                resolved = true;
                reject(e);
              }
            });
          });
          hiddenVideo.addEventListener('error', () => {
            if (!resolved) {
              resolved = true;
              reject(new Error('HLS native playback failed'));
            }
          });
        } else {
          resolved = true;
          reject(new Error('HLS not supported'));
        }
      }).catch(() => {
        if (!resolved) {
          resolved = true;
          reject(new Error('Failed to load HLS library'));
        }
      });
    });
  }

  /**
   * 通过 captureStream() 将隐藏 video 的输出共享给所有订阅者
   */
  private shareViaCaptureStream(
    entry: StreamEntry,
    hiddenVideo: HTMLVideoElement,
    resolved: boolean,
    resolve: (value: void) => void,
    reject: (err: Error) => void,
  ): void {
    try {
      // captureStream 帧率参数
      const captureStream = (hiddenVideo as any).captureStream
        ? (hiddenVideo as any).captureStream(30)
        : (hiddenVideo as any).mozCaptureStream
          ? (hiddenVideo as any).mozCaptureStream(30)
          : null;

      if (captureStream) {
        entry.stream = captureStream;
      } else {
        // 不支持 captureStream 时，直接返回 null
        // 各分屏仍然独立播放（降级行为）
        console.warn('[StreamManager] captureStream not supported, fallback to independent playback');
      }

      entry.status = 'ready';

      // 通知已存在的订阅者
      entry.subscribers.forEach((subVideo) => {
        if (entry.stream) {
          subVideo.srcObject = entry.stream;
          subVideo.play().catch(() => {});
        }
      });

      entry.waitQueue.forEach((w) => w.resolve(entry.stream!));
      entry.waitQueue = [];
      resolve();
    } catch (e: any) {
      entry.stream = null;
      entry.status = 'ready';
      entry.waitQueue.forEach((w) => w.resolve(null as any));
      entry.waitQueue = [];
      resolve(); // 不阻塞，降级为独立播放
    }
  }

  // ==================== 工具函数 ====================

  private loadZLMRTCClient(): Promise<boolean> {
    if ((window as any).ZLMRTCClient) return Promise.resolve(true);
    return new Promise((resolve) => {
      const script = document.createElement('script');
      script.src = '/ZLMRTCClient.js';
      script.onload = () => resolve(true);
      script.onerror = () => resolve(false);
      document.head.appendChild(script);
    });
  }

  private buildZlmWebrtcApiUrl(webrtcUrl: string): string | null {
    try {
      const urlObj = new URL(webrtcUrl.replace('webrtc://', 'http://'));
      const pathParts = urlObj.pathname.split('/').filter(Boolean);
      if (pathParts.length < 2) return null;
      const params = new URLSearchParams({
        app: pathParts[0],
        stream: pathParts[1],
        type: 'play',
      });
      const token = urlObj.searchParams.get('token');
      if (token) params.set('token', token);
      return `${urlObj.protocol}//${urlObj.host}/index/api/webrtc?${params.toString()}`;
    } catch {
      return null;
    }
  }

  getStats(): Array<{ id: string; url: string; protocol: Protocol; refs: number; status: string }> {
    return Array.from(this.streams.values()).map((e) => ({
      id: e.id,
      url: e.url,
      protocol: e.protocol,
      refs: e.refCount,
      status: e.status,
    }));
  }

  destroyAll(): void {
    this.streams.forEach((entry) => entry.destroy());
    this.streams.clear();
  }
}

export const streamManager = StreamManager.getInstance();
export type { Protocol };
