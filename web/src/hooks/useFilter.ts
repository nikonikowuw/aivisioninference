import { useCallback, useRef, useState } from 'react';

export interface FilterState {
  [key: string]: string | undefined;
}

interface UseFilterOptions {
  initialValues?: FilterState;
}

/**
 * useFilter 管理列表搜索筛选状态
 *
 * 【核心功能】提供筛选状态管理、单字段更新、整体重置、搜索触发计数器。
 * searchTrigger 每次搜索时递增，供页面监听触发数据重新加载。
 *
 * @param options - 初始筛选值配置
 * @returns filters 状态、setFilter 单字段更新、resetFilters 重置、searchTrigger 搜索触发计数器、handleSearch 触发搜索
 */
export function useFilter(options?: UseFilterOptions) {
  const [filters, setFilters] = useState<FilterState>(options?.initialValues ?? {});
  const [searchTrigger, setSearchTrigger] = useState(0);
  const initialValuesRef = useRef(options?.initialValues ?? {});

  const setFilter = useCallback((key: string, value: string | undefined) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  }, []);

  const resetFilters = useCallback(() => {
    setFilters({ ...initialValuesRef.current });
    setSearchTrigger(prev => prev + 1);
  }, []);

  const handleSearch = useCallback(() => {
    setSearchTrigger(prev => prev + 1);
  }, []);

  return { filters, setFilter, resetFilters, searchTrigger, handleSearch };
}
