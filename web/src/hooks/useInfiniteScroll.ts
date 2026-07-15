import { useCallback, useEffect, useRef, useState } from 'react';

interface PaginatedResult<T> {
  list: T[];
  total: number;
}

interface UseInfiniteScrollOptions {
  /** 每页条数，默认 24 */
  pageSize?: number;
  /**
   * 触发加载下一页的距离阈值（px），当滚动到距底部小于此值时触发。
   * 默认 300px，确保在卡片网格场景下提前加载以避免白屏。
   */
  threshold?: number;
}

/**
 * useInfiniteScroll 提供基于窗口滚动的无限加载能力。
 *
 * 与 usePagination 的替换关系：
 * - list: 追加式累积，而非按页替换
 * - load/reload: 首次或筛选变更时重置为第一页
 * - 内部自动监听 window scroll 并在接近底部时加载下一页
 *
 * @param fetcher 分页数据拉取函数，签名与 usePagination 一致
 */
export function useInfiniteScroll<T>(
  fetcher: (page: number, pageSize: number) => Promise<PaginatedResult<T>>,
  options?: UseInfiniteScrollOptions,
) {
  const pageSize = options?.pageSize ?? 24;
  const threshold = options?.threshold ?? 300;

  const [list, setList] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [initialLoading, setInitialLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);

  const pageRef = useRef(1);
  const hasMoreRef = useRef(true);
  const loadingRef = useRef(false);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;
  /** 请求序列号，用于丢弃过期响应（筛选条件切换场景） */
  const seqRef = useRef(0);

  /**
   * reload 重置列表并从第一页开始加载（用于筛选变更 / 手动刷新）。
   * 返回 Promise 以便调用方 catch 错误。
   */
  const reload = useCallback(async () => {
    const seq = ++seqRef.current;
    pageRef.current = 1;
    hasMoreRef.current = true;
    loadingRef.current = true;
    setInitialLoading(true);

    try {
      const data = await fetcherRef.current(1, pageSize);
      if (seq !== seqRef.current) return;
      setList(data.list || []);
      setTotal(data.total);
      hasMoreRef.current = (data.list?.length ?? 0) >= pageSize;
      setHasMore(hasMoreRef.current);
    } finally {
      if (seq === seqRef.current) {
        loadingRef.current = false;
        setInitialLoading(false);
      }
    }
  }, [pageSize]);

  /** 加载下一页，追加到现有列表 */
  const loadMore = useCallback(async () => {
    if (loadingRef.current || !hasMoreRef.current) return;
    const seq = seqRef.current;
    loadingRef.current = true;
    setLoadingMore(true);
    const nextPage = pageRef.current + 1;

    try {
      const data = await fetcherRef.current(nextPage, pageSize);
      if (seq !== seqRef.current) return;
      pageRef.current = nextPage;
      setList((prev) => [...prev, ...(data.list || [])]);
      setTotal(data.total);
      hasMoreRef.current = (data.list?.length ?? 0) >= pageSize;
      setHasMore(hasMoreRef.current);
    } finally {
      if (seq === seqRef.current) {
        loadingRef.current = false;
        setLoadingMore(false);
      }
    }
  }, [pageSize]);

  // 窗口滚动监听：接近底部时自动触发 loadMore
  useEffect(() => {
    const onScroll = () => {
      if (loadingRef.current || !hasMoreRef.current) return;
      const scrollBottom =
        document.documentElement.scrollHeight -
        window.scrollY -
        window.innerHeight;
      if (scrollBottom < threshold) {
        loadMore();
      }
    };

    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, [loadMore, threshold]);

  return {
    list,
    setList,
    total,
    initialLoading,
    loadingMore,
    hasMore,
    reload,
    loadMore,
  };
}
