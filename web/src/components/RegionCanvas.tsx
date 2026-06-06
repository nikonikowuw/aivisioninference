import {
  Box,
  Button,
  ButtonGroup,
  HStack,
  VStack,
  Text,
  IconButton,
  Tooltip,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Badge,
} from '@chakra-ui/react';
import { useRef, useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { DeleteIcon } from '@chakra-ui/icons';
import type { ROIRegion, MarkRegion, LineRegion } from 'services/api';

type RegionKind = 'roi' | 'mark' | 'line';
type DrawMode = 'select' | 'polygon' | 'rect' | 'line';

interface Point {
  x: number;
  y: number;
}

interface RegionCanvasProps {
  videoRef: React.RefObject<HTMLVideoElement>;
  roiRegions: ROIRegion[];
  markRegions: MarkRegion[];
  lineRegions: LineRegion[];
  onChange: (data: { roi: ROIRegion[]; mark: MarkRegion[]; line: LineRegion[] }) => void;
}

const KIND_CONFIG: Record<RegionKind, { labelKey: string; color: string; fill: string }> = {
  roi: { labelKey: 'canvas.roiLabel', color: '#48bb78', fill: 'rgba(72,187,120,0.12)' },
  mark: { labelKey: 'canvas.markLabel', color: '#4299e1', fill: 'rgba(66,153,225,0.12)' },
  line: { labelKey: 'canvas.lineLabel', color: '#ed8936', fill: 'rgba(237,137,54,0.3)' },
};

export default function RegionCanvas({ videoRef, roiRegions, markRegions, lineRegions, onChange }: RegionCanvasProps) {
  const { t } = useTranslation(['modules/ai-tasks']);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const [activeTab, setActiveTab] = useState<RegionKind>('roi');
  const [drawMode, setDrawMode] = useState<DrawMode>('select');
  const [currentPoints, setCurrentPoints] = useState<Point[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [isDrawing, setIsDrawing] = useState(false);
  const [canvasSize, setCanvasSize] = useState({ width: 640, height: 360 });
  const [rectStart, setRectStart] = useState<Point | null>(null);
  const [rectEnd, setRectEnd] = useState<Point | null>(null);
  const [lineStart, setLineStart] = useState<Point | null>(null);
  const [lineEnd, setLineEnd] = useState<Point | null>(null);
  const [mousePos, setMousePos] = useState<Point | null>(null);
  const [lineDirection, setLineDirection] = useState<'in' | 'out' | 'both'>('both');

  // 当前 tab 对应的区域列表
  const currentRegions = activeTab === 'roi' ? roiRegions : activeTab === 'mark' ? markRegions : lineRegions;
  const cfg = KIND_CONFIG[activeTab];

  // 推荐绘制模式
  useEffect(() => {
    setDrawMode('select');
    setCurrentPoints([]);
    setSelectedId(null);
  }, [activeTab]);

  // canvas 尺寸
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const ro = new ResizeObserver((entries) => {
      for (const e of entries) {
        const w = e.contentRect.width;
        setCanvasSize({ width: w, height: Math.round(w * 9 / 16) });
      }
    });
    ro.observe(container);
    return () => ro.disconnect();
  }, []);

  // 绘制循环
  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    const video = videoRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // 背景
    if (video && video.readyState >= 2) {
      ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    } else {
      ctx.fillStyle = '#1a1a2e';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      ctx.fillStyle = '#ffffff40';
      ctx.font = '14px sans-serif';
      ctx.textAlign = 'center';
      ctx.fillText(t('canvas.waitingStream'), canvas.width / 2, canvas.height / 2);
    }

    // 绘制所有区域（三类）
    const drawAll = (kind: RegionKind, regions: (ROIRegion | MarkRegion | LineRegion)[]) => {
      const c = KIND_CONFIG[kind];
      regions.forEach((r) => {
        const isActive = r.id === selectedId && kind === activeTab;
        if (kind === 'line') {
          drawLineRegion(ctx, r as LineRegion, c, isActive);
        } else {
          drawShapeRegion(ctx, r as ROIRegion, c, isActive);
        }
      });
    };
    drawAll('roi', roiRegions);
    drawAll('mark', markRegions);
    drawAll('line', lineRegions);

    // 正在绘制的形状
    if (drawMode === 'polygon' && currentPoints.length > 0) {
      drawPolygonInProgress(ctx, currentPoints, cfg.color);
    }
    if (drawMode === 'rect' && rectStart && rectEnd) {
      drawRectInProgress(ctx, rectStart, rectEnd, cfg.color);
    }
    if (drawMode === 'line' && lineStart && lineEnd) {
      drawLineInProgress(ctx, lineStart, lineEnd, cfg.color);
    }

    requestAnimationFrame(draw);
  }, [roiRegions, markRegions, lineRegions, selectedId, activeTab, drawMode, currentPoints, rectStart, rectEnd, lineStart, lineEnd, mousePos, videoRef, cfg, t]);

  useEffect(() => {
    const id = requestAnimationFrame(draw);
    return () => cancelAnimationFrame(id);
  }, [draw]);

  // 绘制多边形/矩形
  const drawShapeRegion = (ctx: CanvasRenderingContext2D, region: ROIRegion, c: typeof KIND_CONFIG.roi, isSelected: boolean) => {
    const w = canvasSize.width;
    const h = canvasSize.height;
    ctx.strokeStyle = isSelected ? '#fff' : c.color;
    ctx.lineWidth = isSelected ? 3 : 2;
    ctx.fillStyle = isSelected ? c.fill.replace('0.12', '0.25') : c.fill;

    if (region.type === 'polygon' && region.points) {
      ctx.beginPath();
      region.points.forEach((p, i) => {
        const px = p[0] * w, py = p[1] * h;
        i === 0 ? ctx.moveTo(px, py) : ctx.lineTo(px, py);
      });
      ctx.closePath();
      ctx.fill();
      ctx.stroke();
      if (isSelected) {
        region.points.forEach((p) => {
          ctx.fillStyle = c.color;
          ctx.beginPath();
          ctx.arc(p[0] * w, p[1] * h, 5, 0, Math.PI * 2);
          ctx.fill();
        });
      }
    } else if (region.type === 'rect' && region.x !== undefined) {
      const x = region.x * w, y = region.y! * h, rw = region.width! * w, rh = region.height! * h;
      ctx.fillRect(x, y, rw, rh);
      ctx.strokeRect(x, y, rw, rh);
    }

    if (region.label) {
      const lx = region.type === 'polygon' && region.points ? region.points[0][0] * w : (region.x || 0) * w;
      const ly = region.type === 'polygon' && region.points ? region.points[0][1] * h - 8 : (region.y || 0) * h - 8;
      ctx.fillStyle = c.color;
      ctx.font = '12px sans-serif';
      ctx.fillText(region.label, lx, ly);
    }
  };

  // 绘制线段
  const drawLineRegion = (ctx: CanvasRenderingContext2D, line: LineRegion, c: typeof KIND_CONFIG.roi, isSelected: boolean) => {
    const w = canvasSize.width, h = canvasSize.height;
    ctx.strokeStyle = isSelected ? '#fff' : c.color;
    ctx.lineWidth = isSelected ? 4 : 3;
    ctx.beginPath();
    ctx.moveTo(line.start[0] * w, line.start[1] * h);
    ctx.lineTo(line.end[0] * w, line.end[1] * h);
    ctx.stroke();

    // 箭头
    const mx = (line.start[0] + line.end[0]) / 2 * w;
    const my = (line.start[1] + line.end[1]) / 2 * h;
    const angle = Math.atan2((line.end[1] - line.start[1]) * h, (line.end[0] - line.start[0]) * w);
    const arrowLen = 12;
    ctx.fillStyle = c.color;
    if (line.direction === 'both' || line.direction === 'in') {
      ctx.beginPath();
      ctx.moveTo(mx + arrowLen * Math.cos(angle), my + arrowLen * Math.sin(angle));
      ctx.lineTo(mx + arrowLen * Math.cos(angle + 2.5), my + arrowLen * Math.sin(angle + 2.5));
      ctx.lineTo(mx + arrowLen * Math.cos(angle - 2.5), my + arrowLen * Math.sin(angle - 2.5));
      ctx.closePath();
      ctx.fill();
    }
    if (line.direction === 'both' || line.direction === 'out') {
      const ra = angle + Math.PI;
      ctx.beginPath();
      ctx.moveTo(mx + arrowLen * Math.cos(ra), my + arrowLen * Math.sin(ra));
      ctx.lineTo(mx + arrowLen * Math.cos(ra + 2.5), my + arrowLen * Math.sin(ra + 2.5));
      ctx.lineTo(mx + arrowLen * Math.cos(ra - 2.5), my + arrowLen * Math.sin(ra - 2.5));
      ctx.closePath();
      ctx.fill();
    }

    // 端点
    [{ x: line.start[0] * w, y: line.start[1] * h }, { x: line.end[0] * w, y: line.end[1] * h }].forEach((p) => {
      ctx.fillStyle = c.color;
      ctx.beginPath();
      ctx.arc(p.x, p.y, 5, 0, Math.PI * 2);
      ctx.fill();
    });

    if (line.label) {
      ctx.fillStyle = c.color;
      ctx.font = '12px sans-serif';
      ctx.fillText(line.label, line.start[0] * w, line.start[1] * h - 10);
    }
  };

  // 绘制进行中的多边形（线跟随鼠标）
  const drawPolygonInProgress = (ctx: CanvasRenderingContext2D, points: Point[], color: string) => {
    const w = canvasSize.width, h = canvasSize.height;

    // 已确定的边
    ctx.strokeStyle = color;
    ctx.lineWidth = 2;
    ctx.beginPath();
    points.forEach((p, i) => { i === 0 ? ctx.moveTo(p.x, p.y) : ctx.lineTo(p.x, p.y); });
    ctx.stroke();

    // 跟随鼠标的线
    if (mousePos && points.length > 0) {
      ctx.setLineDash([6, 4]);
      ctx.beginPath();
      ctx.moveTo(points[points.length - 1].x, points[points.length - 1].y);
      ctx.lineTo(mousePos.x, mousePos.y);
      ctx.stroke();
      ctx.setLineDash([]);
    }

    // 顶点
    points.forEach((p) => {
      ctx.fillStyle = color;
      ctx.beginPath();
      ctx.arc(p.x, p.y, 5, 0, Math.PI * 2);
      ctx.fill();
    });

    // 首顶点高亮（闭合提示）
    if (points.length >= 3 && mousePos) {
      const first = points[0];
      const dist = Math.hypot(mousePos.x - first.x, mousePos.y - first.y);
      if (dist < 15) {
        ctx.strokeStyle = '#fff';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.arc(first.x, first.y, 12, 0, Math.PI * 2);
        ctx.stroke();
        ctx.fillStyle = color;
        ctx.font = '11px sans-serif';
        ctx.fillText(t('canvas.clickToClose'), first.x + 14, first.y + 4);
      } else {
        ctx.fillStyle = color;
        ctx.font = '12px sans-serif';
        ctx.fillText(t('canvas.clickToContinue'), first.x, first.y - 10);
      }
    } else if (points.length > 0) {
      ctx.fillStyle = color;
      ctx.font = '12px sans-serif';
      ctx.fillText(t('canvas.clickToAddVertex'), points[0].x, points[0].y - 10);
    }
  };

  const drawRectInProgress = (ctx: CanvasRenderingContext2D, s: Point, e: Point, color: string) => {
    ctx.strokeStyle = color;
    ctx.lineWidth = 2;
    ctx.setLineDash([5, 5]);
    ctx.fillStyle = color.replace(')', ',0.1)').replace('rgb', 'rgba');
    const x = Math.min(s.x, e.x), y = Math.min(s.y, e.y), w = Math.abs(e.x - s.x), h = Math.abs(e.y - s.y);
    ctx.fillRect(x, y, w, h);
    ctx.strokeRect(x, y, w, h);
    ctx.setLineDash([]);
  };

  const drawLineInProgress = (ctx: CanvasRenderingContext2D, s: Point, e: Point, color: string) => {
    ctx.strokeStyle = color;
    ctx.lineWidth = 3;
    ctx.setLineDash([6, 4]);
    ctx.beginPath();
    ctx.moveTo(s.x, s.y);
    ctx.lineTo(e.x, e.y);
    ctx.stroke();
    ctx.setLineDash([]);
    [s, e].forEach((p) => {
      ctx.fillStyle = color;
      ctx.beginPath();
      ctx.arc(p.x, p.y, 5, 0, Math.PI * 2);
      ctx.fill();
    });
  };

  const norm = (x: number, y: number): Point => ({ x: x / canvasSize.width, y: y / canvasSize.height });
  const getCanvasPoint = (e: React.MouseEvent): Point => {
    const c = canvasRef.current;
    if (!c) return { x: 0, y: 0 };
    const r = c.getBoundingClientRect();
    return { x: e.clientX - r.left, y: e.clientY - r.top };
  };

  // 点击/拖拽处理
  const handleMouseDown = (e: React.MouseEvent) => {
    const pt = getCanvasPoint(e);
    if (drawMode === 'select') {
      const hit = findHitRegion(pt);
      setSelectedId(hit?.id || null);
      // 同步选中绊线的方向
      if (hit) {
        const line = lineRegions.find(l => l.id === hit.id);
        if (line) setLineDirection(line.direction);
      }
      return;
    }
    if (drawMode === 'rect') {
      setRectStart(pt);
      setRectEnd(pt);
      setIsDrawing(true);
    }
    if (drawMode === 'line') {
      if (!lineStart) {
        setLineStart(pt);
        setLineEnd(pt);
        setIsDrawing(true);
      }
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    const pt = getCanvasPoint(e);
    setMousePos(pt);
    if (drawMode === 'rect' && isDrawing && rectStart) setRectEnd(pt);
    if (drawMode === 'line' && isDrawing && lineStart) setLineEnd(pt);
  };

  const handleMouseUp = (e: React.MouseEvent) => {
    const pt = getCanvasPoint(e);
    if (drawMode === 'rect' && isDrawing && rectStart) {
      setIsDrawing(false);
      const s = norm(rectStart.x, rectStart.y);
      const en = norm(pt.x, pt.y);
      if (Math.abs(en.x - s.x) > 0.02 && Math.abs(en.y - s.y) > 0.02) {
        addRegion({
          id: `${activeTab}_${Date.now()}`,
          type: 'rect',
          x: Math.min(s.x, en.x),
          y: Math.min(s.y, en.y),
          width: Math.abs(en.x - s.x),
          height: Math.abs(en.y - s.y),
          label: `${t(cfg.labelKey)} ${currentRegions.length + 1}`,
        });
      }
      setRectStart(null);
      setRectEnd(null);
    }
    if (drawMode === 'line' && isDrawing && lineStart) {
      setIsDrawing(false);
      const s = norm(lineStart.x, lineStart.y);
      const en = norm(pt.x, pt.y);
      const dist = Math.hypot(en.x - s.x, en.y - s.y);
      if (dist > 0.03) {
        const newLine: LineRegion = {
          id: `line_${Date.now()}`,
          start: [s.x, s.y],
          end: [en.x, en.y],
          direction: lineDirection,
          label: `${t('canvas.linePrefix')} ${lineRegions.length + 1}`,
        };
        onChange({ roi: roiRegions, mark: markRegions, line: [...lineRegions, newLine] });
      }
      setLineStart(null);
      setLineEnd(null);
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    if (drawMode === 'polygon') {
      const pt = getCanvasPoint(e);
      // 检查是否靠近首顶点（闭合）
      if (currentPoints.length >= 3) {
        const first = currentPoints[0];
        const dist = Math.hypot(pt.x - first.x, pt.y - first.y);
        if (dist < 15) {
          // 闭合多边形
          const normalizedPoints = currentPoints.map(p => [norm(p.x, p.y).x, norm(p.x, p.y).y]);
          addRegion({
            id: `${activeTab}_${Date.now()}`,
            type: 'polygon',
            points: normalizedPoints,
            label: `${t(cfg.labelKey)} ${currentRegions.length + 1}`,
          });
          setCurrentPoints([]);
          return;
        }
      }
      setCurrentPoints([...currentPoints, pt]);
    }
  };

  const addRegion = (region: ROIRegion) => {
    if (activeTab === 'roi') onChange({ roi: [...roiRegions, region], mark: markRegions, line: lineRegions });
    else if (activeTab === 'mark') onChange({ roi: roiRegions, mark: [...markRegions, region], line: lineRegions });
  };

  const deleteRegion = (id: string) => {
    if (activeTab === 'roi') onChange({ roi: roiRegions.filter(r => r.id !== id), mark: markRegions, line: lineRegions });
    else if (activeTab === 'mark') onChange({ roi: roiRegions, mark: markRegions.filter(r => r.id !== id), line: lineRegions });
    else onChange({ roi: roiRegions, mark: markRegions, line: lineRegions.filter(r => r.id !== id) });
    if (selectedId === id) setSelectedId(null);
  };

  const updateLineDirection = (id: string, direction: 'in' | 'out' | 'both') => {
    onChange({
      roi: roiRegions,
      mark: markRegions,
      line: lineRegions.map(l => l.id === id ? { ...l, direction } : l),
    });
  };

  const findHitRegion = (pt: Point): { id: string } | null => {
    const w = canvasSize.width, h = canvasSize.height;
    const allRegions = [
      ...roiRegions.map(r => ({ ...r, _kind: 'roi' as const })),
      ...markRegions.map(r => ({ ...r, _kind: 'mark' as const })),
    ];
    for (const r of [...allRegions].reverse()) {
      if (r.type === 'polygon' && r.points) {
        const px = pt.x / w, py = pt.y / h;
        let inside = false;
        for (let i = 0, j = r.points.length - 1; i < r.points.length; j = i++) {
          const [xi, yi] = r.points[i], [xj, yj] = r.points[j];
          if ((yi > py) !== (yj > py) && px < (xj - xi) * (py - yi) / (yj - yi) + xi) inside = !inside;
        }
        if (inside) return r;
      }
      if (r.type === 'rect' && r.x !== undefined) {
        const px = pt.x / w, py = pt.y / h;
        if (px >= r.x && px <= r.x + r.width! && py >= r.y! && py <= r.y! + r.height!) return r;
      }
    }
    // 线段命中检测
    for (const l of [...lineRegions].reverse()) {
      const sx = l.start[0] * w, sy = l.start[1] * h, ex = l.end[0] * w, ey = l.end[1] * h;
      const dx = ex - sx, dy = ey - sy;
      const len2 = dx * dx + dy * dy;
      const t = Math.max(0, Math.min(1, ((pt.x - sx) * dx + (pt.y - sy) * dy) / len2));
      const projX = sx + t * dx, projY = sy + t * dy;
      if (Math.hypot(pt.x - projX, pt.y - projY) < 10) return l;
    }
    return null;
  };

  // 键盘
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.key === 'Delete' || e.key === 'Backspace') && selectedId) {
        e.preventDefault();
        deleteRegion(selectedId);
      }
      if (e.key === 'Escape') {
        setCurrentPoints([]);
        setRectStart(null);
        setRectEnd(null);
        setLineStart(null);
        setLineEnd(null);
        setIsDrawing(false);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [selectedId, roiRegions, markRegions, lineRegions]);

  const drawModeOptions: { mode: DrawMode; label: string; icon: string }[] = activeTab === 'line'
    ? [
        { mode: 'select', label: t('canvas.select'), icon: '◇' },
        { mode: 'line', label: t('canvas.drawLine'), icon: '╱' },
      ]
    : [
        { mode: 'select', label: t('canvas.select'), icon: '◇' },
        { mode: 'rect', label: t('canvas.drawRect'), icon: '▭' },
        { mode: 'polygon', label: t('canvas.drawPolygon'), icon: '⬡' },
      ];

  const regionsList = activeTab === 'line'
    ? lineRegions.map(l => ({ id: l.id, label: l.label || t('canvas.lineLabel'), sub: `${l.direction === 'in' ? t('canvas.directionIn') : l.direction === 'out' ? t('canvas.directionOut') : t('canvas.directionBoth')}` }))
    : currentRegions.map(r => ({ id: r.id, label: r.label || t(cfg.labelKey), sub: r.type === 'polygon' ? t('canvas.drawPolygon') : t('canvas.drawRect') }));

  return (
    <VStack spacing={3} align="stretch">
      <Tabs variant="soft-rounded" size="sm" index={['roi', 'mark', 'line'].indexOf(activeTab)} onChange={(i) => setActiveTab(['roi', 'mark', 'line'][i] as RegionKind)}>
        <TabList>
          <Tab>
            <HStack spacing={1}>
              <Box w={2} h={2} borderRadius="full" bg={KIND_CONFIG.roi.color} />
              <Text>{t('canvas.roiLabel')}</Text>
              {roiRegions.length > 0 && <Badge colorScheme="green" fontSize="xs">{roiRegions.length}</Badge>}
            </HStack>
          </Tab>
          <Tab>
            <HStack spacing={1}>
              <Box w={2} h={2} borderRadius="full" bg={KIND_CONFIG.mark.color} />
              <Text>{t('canvas.markLabel')}</Text>
              {markRegions.length > 0 && <Badge colorScheme="blue" fontSize="xs">{markRegions.length}</Badge>}
            </HStack>
          </Tab>
          <Tab>
            <HStack spacing={1}>
              <Box w={2} h={2} borderRadius="full" bg={KIND_CONFIG.line.color} />
              <Text>{t('canvas.lineLabel')}</Text>
              {lineRegions.length > 0 && <Badge colorScheme="orange" fontSize="xs">{lineRegions.length}</Badge>}
            </HStack>
          </Tab>
        </TabList>
      </Tabs>

      {/* 工具栏 */}
      <HStack spacing={2} flexWrap="wrap">
        <ButtonGroup size="sm" isAttached variant="outline">
          {drawModeOptions.map((opt) => (
            <Tooltip key={opt.mode} label={opt.label}>
              <Button isActive={drawMode === opt.mode} onClick={() => { setDrawMode(opt.mode); setCurrentPoints([]); }}>
                {opt.icon}
              </Button>
            </Tooltip>
          ))}
        </ButtonGroup>
        {activeTab === 'line' && (
          <ButtonGroup size="sm" isAttached variant="outline">
            <Tooltip label={t('canvas.directionBoth')}>
              <Button
                isActive={lineDirection === 'both'}
                onClick={() => {
                  setLineDirection('both');
                  if (selectedId) updateLineDirection(selectedId, 'both');
                }}
                colorScheme={lineDirection === 'both' ? 'orange' : undefined}
              >
                {t('canvas.directionBoth')}
              </Button>
            </Tooltip>
            <Tooltip label={t('canvas.directionIn')}>
              <Button
                isActive={lineDirection === 'in'}
                onClick={() => {
                  setLineDirection('in');
                  if (selectedId) updateLineDirection(selectedId, 'in');
                }}
                colorScheme={lineDirection === 'in' ? 'orange' : undefined}
              >
                {t('canvas.directionIn')}
              </Button>
            </Tooltip>
            <Tooltip label={t('canvas.directionOut')}>
              <Button
                isActive={lineDirection === 'out'}
                onClick={() => {
                  setLineDirection('out');
                  if (selectedId) updateLineDirection(selectedId, 'out');
                }}
                colorScheme={lineDirection === 'out' ? 'orange' : undefined}
              >
                {t('canvas.directionOut')}
              </Button>
            </Tooltip>
          </ButtonGroup>
        )}
        {selectedId && (
          <Tooltip label={t('canvas.deleteSelected')}>
            <IconButton aria-label={t('canvas.deleteSelected')} icon={<DeleteIcon />} size="sm" colorScheme="red" variant="ghost" onClick={() => deleteRegion(selectedId)} />
          </Tooltip>
        )}
        <Text fontSize="xs" color="gray.500" ml={2}>
          {drawMode === 'polygon' && t('canvas.clickToAddVertex')}
          {drawMode === 'rect' && t('canvas.drawRect')}
          {drawMode === 'line' && t('canvas.drawLine')}
          {drawMode === 'select' && t('canvas.select')}
        </Text>
        <Box flex={1} />
        {currentRegions.length > 0 && (
          <Tooltip label={`${t('canvas.deleteSelected')} ${t(cfg.labelKey)}`}>
            <Button
              size="xs"
              variant="ghost"
              colorScheme="red"
              onClick={() => {
                if (activeTab === 'roi') onChange({ roi: [], mark: markRegions, line: lineRegions });
                else if (activeTab === 'mark') onChange({ roi: roiRegions, mark: [], line: lineRegions });
                else onChange({ roi: roiRegions, mark: markRegions, line: [] });
                setSelectedId(null);
              }}
            >
              {t('canvas.deleteSelected')} {t(cfg.labelKey)}
            </Button>
          </Tooltip>
        )}
        {(roiRegions.length + markRegions.length + lineRegions.length) > 0 && (
          <Tooltip label={t('canvas.clearAll')}>
            <Button
              size="xs"
              variant="ghost"
              colorScheme="red"
              onClick={() => {
                onChange({ roi: [], mark: [], line: [] });
                setSelectedId(null);
              }}
            >
              {t('canvas.clearAll')}
            </Button>
          </Tooltip>
        )}
      </HStack>

      {/* 画布 */}
      <Box ref={containerRef} position="relative" border="1px solid" borderColor="gray.200" borderRadius="md" overflow="hidden">
        <canvas
          ref={canvasRef}
          width={canvasSize.width}
          height={canvasSize.height}
          style={{ width: '100%', height: 'auto', cursor: drawMode === 'select' ? 'default' : 'crosshair' }}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onClick={handleClick}
        />
      </Box>

      {/* 区域列表 */}
      {regionsList.length > 0 && (
        <VStack spacing={1} align="stretch">
          <Text fontSize="xs" fontWeight="600" color="gray.500">{t(cfg.labelKey)}：</Text>
          {regionsList.map((r) => (
            <HStack
              key={r.id} spacing={2} p={2}
              bg={selectedId === r.id ? 'blue.50' : 'gray.50'}
              borderRadius="md" cursor="pointer"
              onClick={() => setSelectedId(r.id)}
            >
              <Box w={2} h={2} borderRadius="full" bg={cfg.color} />
              <Text fontSize="sm" flex={1}>{r.label}</Text>
              <Text fontSize="xs" color="gray.500">{r.sub}</Text>
              <IconButton
                aria-label={t('canvas.deleteSelected')} icon={<DeleteIcon />} size="xs" variant="ghost" colorScheme="red"
                onClick={(e) => { e.stopPropagation(); deleteRegion(r.id); }}
              />
            </HStack>
          ))}
        </VStack>
      )}
    </VStack>
  );
}
