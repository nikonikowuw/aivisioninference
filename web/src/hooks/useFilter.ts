import { useCallback, useEffect, useRef, useState } from 'react';

export interface FilterState {
  [key: string]: string | undefined;
}

interface UseFilterOptions {
  initialValues?: FilterState;
  debounceMs?: number;
}

const isEmpty = (f: FilterState) => !Object.values(f).some(Boolean);

/**
 * useFilter 管理列表搜索筛选状态
 *
 * 【核心功能】提供筛选状态管理、单字段更新、整体重置、搜索触发计数器。
 * searchTrigger 每次搜索时递增，供页面监听触发数据重新加载。
 * 筛选条件变化后自动 debounce 触发搜索（默认 300ms）。
 * 空筛选条件下跳过 debounce，由页面自身 load 处理初始加载。
 *
 * @param options - 初始筛选值配置
 * @returns filters 状态、setFilter 单字段更新、resetFilters 重置、searchTrigger 搜索触发计数器
 */
export function useFilter(options?: UseFilterOptions) {
  const [filters, setFilters] = useState<FilterState>(options?.initialValues ?? {});
  const [searchTrigger, setSearchTrigger] = useState(0);
  const initialValuesRef = useRef(options?.initialValues ?? {});
  const debounceMs = options?.debounceMs ?? 300;
  const mountedRef = useRef(false);

  // 同步 initialValuesRef，确保父组件传入新值时 reset 能恢复到最新初始值
  useEffect(() => {
    initialValuesRef.current = options?.initialValues ?? {};
  }, [options?.initialValues]);

  const setFilter = useCallback((key: string, value: string | undefined) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  }, []);

  // 重置 filters 到初始值，由 debounce effect 自动触发搜索
  const resetFilters = useCallback(() => {
    setFilters({ ...initialValuesRef.current });
  }, []);

  // 筛选条件变化后 debounce 自动触发搜索
  // 首次挂载跳过，由页面 useEffect([searchTrigger]) 的初始 searchTrigger=0 触发首次加载
  useEffect(() => {
    if (!mountedRef.current) {
      mountedRef.current = true;
      return;
    }
    if (isEmpty(filters)) {
      setSearchTrigger(prev => prev + 1);
      return;
    }
    const timer = setTimeout(() => {
      setSearchTrigger(prev => prev + 1);
    }, debounceMs);
    return () => clearTimeout(timer);
  }, [filters, debounceMs]);

  return { filters, setFilter, resetFilters, searchTrigger };
}
