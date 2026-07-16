import { useEffect, useRef } from 'react';
import { WS_TOPIC } from 'constants/websocket';
import { useWebSocket } from './useWebSocket';

export interface Detection {
  x: number;
  y: number;
  w: number;
  h: number;
  confidence: number;
  labelName: string;
  labelId: number;
  trackId: number;
}

export function useThrottledInference(
  deviceId: string | undefined,
  videoRef: React.RefObject<HTMLVideoElement | null>,
  canvasRef: React.RefObject<HTMLCanvasElement | null>
) {
  const detectionsRef = useRef<Detection[]>([]);
  const lastUpdateRef = useRef<number>(0);
  const animationFrameRef = useRef<number | null>(null);

  useWebSocket({
    onMessage: (msg: any) => {
      if (msg.type === WS_TOPIC.INFERENCE && msg.payload && msg.payload.device_id === deviceId) {
        detectionsRef.current = msg.payload.detections || [];
        lastUpdateRef.current = Date.now();
      }
    },
  });

  useEffect(() => {
    const video = videoRef.current;
    const canvas = canvasRef.current;
    if (!video || !canvas || !deviceId) {
      return;
    }

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let isActive = true;

    const renderLoop = () => {
      if (!isActive) return;

      if (video.videoWidth > 0 && video.videoHeight > 0) {
        // Synchronize canvas logical resolution with layout size of the video element
        const rect = video.getBoundingClientRect();
        if (canvas.width !== rect.width || canvas.height !== rect.height) {
          canvas.width = rect.width;
          canvas.height = rect.height;
        }

        ctx.clearRect(0, 0, canvas.width, canvas.height);

        const now = Date.now();
        // Keep detections valid for up to 1 second to prevent immediate flickering
        if (now - lastUpdateRef.current < 1000 && detectionsRef.current.length > 0) {
          // Adjust for "object-fit: contain" scaling and offsetting
          const videoRatio = video.videoWidth / video.videoHeight;
          const canvasRatio = canvas.width / canvas.height;

          let drawWidth = canvas.width;
          let drawHeight = canvas.height;
          let offsetX = 0;
          let offsetY = 0;

          if (videoRatio > canvasRatio) {
            // Black bars on top and bottom
            drawHeight = canvas.width / videoRatio;
            offsetY = (canvas.height - drawHeight) / 2;
          } else {
            // Black bars on sides
            drawWidth = canvas.height * videoRatio;
            offsetX = (canvas.width - drawWidth) / 2;
          }

          const scaleX = drawWidth / video.videoWidth;
          const scaleY = drawHeight / video.videoHeight;

          detectionsRef.current.forEach((det) => {
            const x = offsetX + det.x * scaleX;
            const y = offsetY + det.y * scaleY;
            const w = det.w * scaleX;
            const h = det.h * scaleY;

            // Bounding box styling
            ctx.strokeStyle = '#00FF66'; // Premium lime green neon accent color
            ctx.lineWidth = 2;
            ctx.shadowBlur = 4;
            ctx.shadowColor = 'rgba(0, 255, 102, 0.5)';
            ctx.strokeRect(x, y, w, h);

            // Reset shadow for text / background
            ctx.shadowBlur = 0;

            // Tag style
            const labelText = `${det.labelName} (${Math.round(det.confidence * 100)}%)`;
            ctx.font = 'bold 11px Inter, system-ui, -apple-system, sans-serif';
            
            const textPadding = 6;
            const textWidth = ctx.measureText(labelText).width;
            const tagHeight = 18;
            
            const tagX = x;
            const tagY = y - tagHeight >= 0 ? y - tagHeight : y;

            ctx.fillStyle = 'rgba(0, 255, 102, 0.9)'; // Neon label tag background
            ctx.fillRect(tagX, tagY, textWidth + textPadding * 2, tagHeight);

            ctx.fillStyle = '#000000'; // Dark text for readability
            ctx.fillText(labelText, tagX + textPadding, tagY + 12);
          });
        }
      }

      animationFrameRef.current = requestAnimationFrame(renderLoop);
    };

    renderLoop();

    return () => {
      isActive = false;
      if (animationFrameRef.current !== null) {
        cancelAnimationFrame(animationFrameRef.current);
      }
      if (canvas) {
        const context = canvas.getContext('2d');
        if (context) {
          context.clearRect(0, 0, canvas.width, canvas.height);
        }
      }
    };
  }, [deviceId, videoRef, canvasRef]);
}
