import { useEffect, useRef, useState, useCallback } from 'react';
import { Box, Flex, Button, Text, Slider, SliderTrack, SliderFilledTrack, SliderThumb, Select, Spinner, useToast } from '@chakra-ui/react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { terminalApi } from 'services/terminal';

interface TtyrecEntry {
  time: number;
  data: string;
}

interface PlaybackViewProps {
  sessionId: string;
  nodeId?: string;
  recording?: TtyrecEntry[];
  onClose?: () => void;
}

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}

function decodeBase64(data: string): Uint8Array {
  const binaryStr = atob(data);
  const bytes = new Uint8Array(binaryStr.length);
  for (let i = 0; i < binaryStr.length; i++) {
    bytes[i] = binaryStr.charCodeAt(i);
  }
  return bytes;
}

export default function PlaybackView({ sessionId, nodeId, recording: propRecording, onClose }: PlaybackViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const timerRef = useRef<number | null>(null);
  const speedRef = useRef(1);
  const playingRef = useRef(false);
  const lastWrittenIndexRef = useRef(0);
  const [entries, setEntries] = useState<TtyrecEntry[]>(propRecording || []);
  const [loading, setLoading] = useState(!propRecording);
  const [error, setError] = useState<string | null>(null);
  const [playing, setPlaying] = useState(false);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [speed, setSpeed] = useState(1);
  const toast = useToast();

  // Calculate total duration from last entry
  const totalTime = entries.length > 0 ? entries[entries.length - 1].time : 0;
  const currentTime = currentIndex > 0 && currentIndex <= entries.length ? entries[currentIndex - 1].time : 0;

  // Keep refs in sync
  useEffect(() => { speedRef.current = speed; }, [speed]);
  useEffect(() => { playingRef.current = playing; }, [playing]);

  // Fetch recording data if not provided
  useEffect(() => {
    if (propRecording) {
      setEntries(propRecording);
      setLoading(false);
      return;
    }
    if (!nodeId) return;

    let cancelled = false;
    const fetchRecording = async () => {
      try {
        const data = await terminalApi.getRecording(nodeId, sessionId);
        if (!cancelled) {
          const list = (Array.isArray(data) ? data : []) as TtyrecEntry[];
          setEntries(list);
          setError(null);
        }
      } catch {
        if (!cancelled) {
          setError('加载录制数据失败');
          toast({ title: '加载录制数据失败', status: 'error' });
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    fetchRecording();
    return () => { cancelled = true; };
  }, [sessionId, nodeId, propRecording, toast]);

  // Initialize xterm.js terminal
  useEffect(() => {
    if (!containerRef.current) return;

    const term = new Terminal({
      cursorBlink: false,
      cursorStyle: 'block',
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1a1b2e',
        foreground: '#cdd6f4',
        cursor: '#f5e0dc',
        selectionBackground: '#585b70',
        black: '#45475a',
        red: '#f38ba8',
        green: '#a6e3a1',
        yellow: '#f9e2af',
        blue: '#89b4fa',
        magenta: '#f5c2e7',
        cyan: '#94e2d5',
        white: '#bac2de',
        brightBlack: '#585b70',
        brightRed: '#f38ba8',
        brightGreen: '#a6e3a1',
        brightYellow: '#f9e2af',
        brightBlue: '#89b4fa',
        brightMagenta: '#f5c2e7',
        brightCyan: '#94e2d5',
        brightWhite: '#a6adc8',
      },
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    fitAddonRef.current = fitAddon;

    term.open(containerRef.current);
    fitAddon.fit();

    // Disable user input
    term.attachCustomKeyEventHandler(() => false);

    terminalRef.current = term;

    const handleResize = () => fitAddon.fit();
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      if (timerRef.current !== null) clearTimeout(timerRef.current);
      term.dispose();
      terminalRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const writeEntry = useCallback((entry: TtyrecEntry) => {
    const term = terminalRef.current;
    if (!term) return;
    try {
      term.write(decodeBase64(entry.data));
    } catch {
      // Skip entries that fail to decode
    }
  }, []);

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const scheduleNext = useCallback(() => {
    clearTimer();

    const entries_ = entries;
    const idx = currentIndex;
    const spd = speedRef.current;

    if (idx >= entries_.length) {
      setPlaying(false);
      return;
    }

    const entry = entries_[idx];
    writeEntry(entry);
    const nextIdx = idx + 1;
    setCurrentIndex(nextIdx);
    lastWrittenIndexRef.current = nextIdx;

    if (nextIdx < entries_.length) {
      const delay = (entries_[nextIdx].time - entry.time) / spd;
      const msDelay = Math.max(Math.round(delay * 1000), 10);
      timerRef.current = window.setTimeout(() => {
        if (playingRef.current) {
          scheduleNext();
        }
      }, msDelay);
    } else {
      // Reached end
      setPlaying(false);
    }
  }, [entries, currentIndex, writeEntry, clearTimer]);

  const play = useCallback(() => {
    const term = terminalRef.current;
    if (!term || entries.length === 0) return;

    if (currentIndex >= entries.length) {
      // Reached end, restart from beginning
      setCurrentIndex(0);
      term.reset();
      lastWrittenIndexRef.current = 0;
    }

    setPlaying(true);
    // scheduleNext will be triggered by the useEffect below
  }, [entries, currentIndex]);

  // Start playback when playing becomes true
  useEffect(() => {
    if (playing && entries.length > 0) {
      // Small delay to let state settle
      timerRef.current = window.setTimeout(() => {
        scheduleNext();
      }, 50);
      return () => clearTimer();
    }
  }, [playing, entries, scheduleNext, clearTimer]);

  const pause = useCallback(() => {
    clearTimer();
    setPlaying(false);
  }, [clearTimer]);

  const seek = useCallback((targetTime: number) => {
    pause();

    // Find how many entries to replay for the target time
    let idx = 0;
    for (let i = 0; i < entries.length; i++) {
      if (entries[i].time <= targetTime) {
        idx = i + 1;
      } else {
        break;
      }
    }

    // Seek forward within already-written range: only append new entries
    if (idx >= lastWrittenIndexRef.current) {
      for (let i = lastWrittenIndexRef.current; i < idx && i < entries.length; i++) {
        writeEntry(entries[i]);
      }
    } else {
      // Seek backward: rebuild terminal from start
      terminalRef.current?.reset();
      for (let i = 0; i < idx && i < entries.length; i++) {
        writeEntry(entries[i]);
      }
    }
    lastWrittenIndexRef.current = idx;
    setCurrentIndex(idx);
  }, [entries, pause, writeEntry]);

  const stop = useCallback(() => {
    clearTimer();
    setPlaying(false);
    terminalRef.current?.reset();
    setCurrentIndex(0);
    lastWrittenIndexRef.current = 0;
  }, [clearTimer]);

  const handleSpeedChange = useCallback((newSpeed: number) => {
    setSpeed(newSpeed);
    if (playing) {
      // Restart playback with new speed
      clearTimer();
      timerRef.current = window.setTimeout(() => {
        scheduleNext();
      }, 50);
    }
  }, [playing, clearTimer, scheduleNext]);

  return (
    <Box>
      <Flex mb={3} align="center" justify="space-between">
        <Flex align="center" gap={3}>
          <Text fontSize="sm" fontWeight="medium">终端回放</Text>
          {entries.length > 0 && (
            <Text fontSize="xs" color="gray.400">
              {entries.length} 条记录 · {formatTime(totalTime)}
            </Text>
          )}
        </Flex>
        {onClose && (
          <Button size="xs" variant="ghost" onClick={onClose}>关闭</Button>
        )}
      </Flex>

      <Box
        ref={containerRef}
        h="400px"
        bg="#1a1b2e"
        borderRadius="md"
        overflow="hidden"
        position="relative"
      >
        {loading && (
          <Flex position="absolute" inset={0} align="center" justify="center" bg="rgba(26,27,46,0.8)" zIndex={1}>
            <Spinner color="blue.400" />
          </Flex>
        )}
        {error && (
          <Flex position="absolute" inset={0} align="center" justify="center" bg="rgba(26,27,46,0.8)" zIndex={1}>
            <Text color="red.300">{error}</Text>
          </Flex>
        )}
        {!loading && !error && entries.length === 0 && (
          <Flex position="absolute" inset={0} align="center" justify="center" bg="rgba(26,27,46,0.8)" zIndex={1}>
            <Text color="gray.400">无录制数据</Text>
          </Flex>
        )}
      </Box>

      {/* Playback Controls */}
      {entries.length > 0 && (
        <Box mt={3}>
          {/* Progress Slider */}
          <Flex align="center" gap={3} mb={2}>
            <Text fontSize="xs" color="gray.400" w="40px" textAlign="right">
              {formatTime(currentTime)}
            </Text>
            <Slider
              flex={1}
              min={0}
              max={Math.max(totalTime, 1)}
              value={currentTime}
              step={0.1}
              onChange={seek}
              focusThumbOnChange={false}
            >
              <SliderTrack>
                <SliderFilledTrack />
              </SliderTrack>
              <SliderThumb />
            </Slider>
            <Text fontSize="xs" color="gray.400" w="40px">
              {formatTime(totalTime)}
            </Text>
          </Flex>

          {/* Buttons */}
          <Flex align="center" gap={2}>
            <Button size="sm" onClick={playing ? pause : play} isDisabled={entries.length === 0}>
              {playing ? '暂停' : '播放'}
            </Button>
            <Button size="sm" variant="outline" onClick={stop} isDisabled={entries.length === 0}>
              停止
            </Button>

            <Box flex={1} />

            <Flex align="center" gap={2}>
              <Text fontSize="xs" color="gray.400">速度:</Text>
              <Select
                size="xs"
                w="70px"
                value={speed}
                onChange={(e) => handleSpeedChange(Number(e.target.value))}
              >
                <option value={0.5}>0.5x</option>
                <option value={1}>1x</option>
                <option value={2}>2x</option>
                <option value={4}>4x</option>
              </Select>
            </Flex>
          </Flex>
        </Box>
      )}
    </Box>
  );
}
