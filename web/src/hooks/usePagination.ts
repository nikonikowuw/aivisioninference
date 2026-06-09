import { useCallback, useRef, useState } from 'react';
import { useToast } from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';

interface PaginatedResult<T> {
  list: T[];
  total: number;
}

interface LoadOptions {
  page?: number;
  pageSize?: number;
  silent?: boolean;
}

export function usePagination<T>(
  fetcher: (page: number, pageSize: number) => Promise<PaginatedResult<T>>,
) {
  const [list, setList] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [initialLoading, setInitialLoading] = useState(true);
  const [pageLoading, setPageLoading] = useState(false);
  const requestIdRef = useRef(0);
  const initialRef = useRef(true);
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;
  const pageRef = useRef(page);
  pageRef.current = page;
  const pageSizeRef = useRef(pageSize);
  pageSizeRef.current = pageSize;
  const toast = useToast();
  const { t } = useTranslation();

  const load = useCallback(async (opts?: LoadOptions) => {
    const id = ++requestIdRef.current;
    const p = opts?.page ?? pageRef.current;
    const ps = opts?.pageSize ?? pageSizeRef.current;
    const silent = opts?.silent ?? false;
    
    if (!silent) setPageLoading(true);
    
    try {
      const data = await fetcherRef.current(p, ps);
      if (id !== requestIdRef.current) return;
      setList(data.list || []);
      setTotal(data.total);
      if (opts?.page !== undefined) setPage(opts.page);
      if (opts?.pageSize !== undefined) setPageSize(opts.pageSize);
    } catch {
      if (id !== requestIdRef.current) return;
      toast({ title: t('common:message.loadFailed'), status: 'error' });
    } finally {
      if (id === requestIdRef.current) {
        if (!silent) setPageLoading(false);
        if (initialRef.current) {
          initialRef.current = false;
          setInitialLoading(false);
        }
      }
    }
  }, [toast, t]);

  const changePage = useCallback((p: number) => {
    load({ page: p });
  }, [load]);

  const changePageSize = useCallback((s: number) => {
    load({ page: 1, pageSize: s });
  }, [load]);

  return {
    list,
    total,
    page,
    pageSize,
    initialLoading,
    pageLoading,
    load,
    changePage,
    changePageSize,
    setList,
  };
}
