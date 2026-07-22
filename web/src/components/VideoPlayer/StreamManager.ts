/**
 * StreamManager - 流复用管理器
 *
 * 核心逻辑：
 * 1. 同一个 URL 只建立一次连接
 * 2. 所有订阅者共享同一个 WebRTC MediaStream
 * 3. 最后一个订阅者移除后关闭连接
 */

type Protocol = 'webrtc' | 'hls' | 'flv';

interface StreamEntry {
  id: string;
  url: string;
  /** 共享的 WebRTC MediaStream */
  stream: MediaStream | null;
  /** 播放器清理函数 */
  destroy: () => void;
  /** 订阅元素及其断流降级回调。 */
  subscribers: Map<HTMLVideoElement, { onDisconnect?: (error: string) => void }>;
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
    hash = ((hash << 5) - hash) + cleanUrl.charCodeAt(i);
    hash |= 0;
  }
  return `s${Math.abs(hash).toString(36)}`;
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
    video: HTMLVideoElement,
    options?: { webrtcTimeout?: number; onDisconnect?: (error: string) => void },
  ): Promise<{ stream: MediaStream | null; destroy: () => void }> {
    const id = generateStreamId(url);
    let entry = this.streams.get(id);

    if (entry) {
      entry.subscribers.set(video, { onDisconnect: options?.onDisconnect });
      console.log(`[StreamManager] 复用流: ${id}, 引用: ${entry.subscribers.size}`);

      if (entry.status === 'ready' && entry.stream) {
        video.srcObject = entry.stream;
        video.play().catch(() => { });
        return { stream: entry.stream, destroy: () => this.unsubscribe(id, video) };
      }

      if (entry.status === 'loading') {
        return new Promise((resolve, reject) => {
          entry!.waitQueue.push({
            resolve: (stream) => {
              video.srcObject = stream;
              video.play().catch(() => { });
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
        return this.subscribe(url, video, options);
      }
    }

    // 创建新流
    entry = {
      id,
      url,
      stream: null,
      destroy: () => { },
      subscribers: new Map([[video, { onDisconnect: options?.onDisconnect }]]),
      status: 'loading',
      error: null,
      waitQueue: [],
    };
    this.streams.set(id, entry);

    try {
      await this.connectWebrtc(entry, options?.webrtcTimeout || 10000);
    } catch (error) {
      entry.status = 'error';
      entry.error = error instanceof Error ? error.message : String(error);
      entry.waitQueue.forEach((w) => w.reject(entry!.error!));
      entry.waitQueue = [];
      entry.destroy();
      if (this.streams.get(id) === entry) this.streams.delete(id);
      throw error;
    }

    // 将共享流绑定到当前 video
    if (entry.stream) {
      video.srcObject = entry.stream;
      video.play().catch(() => { });
    }

    return { stream: entry.stream, destroy: () => this.unsubscribe(id, video) };
  }

  private unsubscribe(id: string, video: HTMLVideoElement): void {
    const entry = this.streams.get(id);
    if (!entry) return;

    entry.subscribers.delete(video);
    video.srcObject = null;

    if (entry.subscribers.size === 0) {
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

      const tempVideo = this.createHiddenVideo();

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
        if (player) { try { player.close(); } catch { } }
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
        if (entry.status === 'error') return;
        clearTimeout(timeoutTimer);
        entry.status = 'error';
        entry.error = msg;
        if (resolved) {
          Array.from(entry.subscribers.values()).forEach(({ onDisconnect }) => onDisconnect?.(msg));
          return;
        }
        resolved = true;
        tempVideo.remove();
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

  // ==================== 工具函数 ====================

  // display:none 会阻止部分浏览器初始化视频解码管线。
  private createHiddenVideo(): HTMLVideoElement {
    const video = document.createElement('video');
    video.style.position = 'absolute';
    video.style.width = '1px';
    video.style.height = '1px';
    video.style.opacity = '0';
    video.style.pointerEvents = 'none';
    video.muted = true;
    video.playsInline = true;
    video.setAttribute('playsinline', '');
    document.body.appendChild(video);
    return video;
  }

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
      protocol: 'webrtc',
      refs: e.subscribers.size,
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
