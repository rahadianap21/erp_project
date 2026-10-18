'use client';

import { API_BASE_URL } from "@/lib/config"
import { useState, useEffect, useCallback } from 'react';
import Cookies from "js-cookie";

interface UseDebounceFetchProps {
    endpoint: string;
    delay?: number;
    initialFilters?: Record<string, string | number | boolean>;
    refreshTrigger?: number;
}

export function useDebounceFetch<T>({
    endpoint,
    delay = 500,
    initialFilters = {},
    refreshTrigger = 0
}: UseDebounceFetchProps) {
    const [search, setSearch] = useState<string>('');
    const [debouncedSearch, setDebouncedSearch] = useState<string>('');

    // 🌟 State paginasi: page & limit serta cursor-based tracking
    const [page, setPageInternal] = useState<number>(1);
    const [limit, setLimitInternal] = useState<number>(25);

    // Cursor Pagination States
    const [cursor, setCursor] = useState<number | string | null>(null);
    const [direction, setDirection] = useState<'next' | 'prev'>('next');
    const [nextCursor, setNextCursor] = useState<number | string | null>(null);
    const [prevCursor, setPrevCursor] = useState<number | string | null>(null);
    const [hasNext, setHasNext] = useState<boolean>(false);
    const [hasPrev, setHasPrev] = useState<boolean>(false);

    // State khusus penampung filter data riil (misal: role, status, kategori, dll.)
    const [filters, setFilters] = useState<Record<string, any>>(initialFilters);

    const [data, setData] = useState<T[]>([]);
    const [total, setTotal] = useState<number>(0);
    const [loading, setLoading] = useState<boolean>(true);
    const [roleOptions, setRoleOptions] = useState<string[]>([]);

    // Status/error HTTP agar halaman bisa membedakan 403 (tidak berizin) dari
    // kegagalan lain. Sebelumnya error ditelan oleh console.error sehingga
    // halaman diam-diam menampilkan tabel kosong saat pengguna tak punya akses.
    const [httpStatus, setHttpStatus] = useState<number | null>(null);
    const [error, setError] = useState<string | null>(null);

    const resetPagination = useCallback(() => {
        setCursor(null);
        setDirection('next');
        setPageInternal(1);
    }, []);

    // Efek Debounce untuk isian kolom teks pencarian
    useEffect(() => {
        const handler = setTimeout(() => {
            setDebouncedSearch(search);
            resetPagination();
        }, delay);
        return () => clearTimeout(handler);
    }, [search, delay, resetPagination]);

    // Reset pagination jika filter kustom dari luar berubah
    useEffect(() => {
        resetPagination();
    }, [filters, resetPagination]);

    const setLimit = useCallback((newLimit: number) => {
        setLimitInternal(newLimit);
        resetPagination();
    }, [resetPagination]);

    const setPage = useCallback((newPage: number | ((p: number) => number)) => {
        if (typeof newPage === 'function') {
            setPageInternal((prev) => {
                const updated = newPage(prev);
                if (updated === 1) resetPagination();
                return updated;
            });
        } else {
            if (newPage === 1) resetPagination();
            else setPageInternal(newPage);
        }
    }, [resetPagination]);

    const fetchData = useCallback(async () => {
        setLoading(true);
        setHttpStatus(null);
        setError(null);
        try {
            const queryParams = new URLSearchParams();

            queryParams.append("search", debouncedSearch);
            queryParams.append("limit", String(limit));

            // Nama parameter cursor TIDAK seragam di backend Go:
            //   - /api/products, /api/suppliers, /api/categories, /api/purchase-orders
            //     membaca "cursor_id"
            //   - /api/roles, /api/users membaca "cursor"
            // Kirim keduanya dengan nilai sama supaya hook ini tidak perlu
            // tahu endpoint mana yang memakai konvensi apa.
            if (cursor !== null && cursor !== undefined) {
                queryParams.append("cursor", String(cursor));
                queryParams.append("cursor_id", String(cursor));
                queryParams.append("direction", direction);
            }
            queryParams.append("page", String(page));

            // Masukkan filter data kustom tambahan jika ada
            Object.keys(filters).forEach((key) => {
                if (filters[key] !== undefined && filters[key] !== null) {
                    queryParams.append(key, String(filters[key]));
                }
            });

            const authToken = Cookies.get("auth_token") || "";
            const apiBaseUrl = API_BASE_URL;

            const response = await fetch(`${apiBaseUrl}${endpoint}?${queryParams.toString()}`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`
                },
                cache: 'no-store'
            });

            if (!response.ok) {
                setHttpStatus(response.status);
                throw new Error(`Server error: ${response.status}`);
            }

            const result = await response.json().catch(() => ({}));
            const rows = Array.isArray(result.data) ? result.data : [];

            setData(rows);
            setTotal(Number(result.total) || rows.length || 0);

            // Bentuk kontrak paginasi juga TIDAK seragam antar endpoint:
            //   1. /api/roles, /api/users  -> next_cursor/prev_cursor/has_next/has_prev di top-level
            //   2. /api/products, /api/suppliers, /api/categories -> objek `pagination` dengan has_more
            //   3. /api/purchase-orders -> tidak mengirim metadata sama sekali
            const pg = (result.pagination && typeof result.pagination === 'object')
                ? result.pagination
                : {};

            // Fallback (3): cursor backend adalah id baris terakhir, urutan id DESC.
            const lastRowId = rows.length ? (rows[rows.length - 1] as { id?: number | string }).id : null;
            const firstRowId = rows.length ? (rows[0] as { id?: number | string }).id : null;

            setNextCursor(
                result.next_cursor !== undefined ? result.next_cursor
                    : pg.next_cursor !== undefined ? pg.next_cursor
                        : lastRowId
            );
            setPrevCursor(
                result.prev_cursor !== undefined ? result.prev_cursor
                    : pg.prev_cursor !== undefined ? pg.prev_cursor
                        : firstRowId
            );

            // `total` lebih akurat daripada `has_more` milik backend, yang hanya
            // menebak `len(rows) >= limit` dan meleset tepat di batas kelipatan
            // (mis. total=2, limit=2 -> halaman 1 mengPromise "ada berikutnya",
            // lalu halaman 2 kosong).
            const totalRows = Number(result.total);
            const hasReliableTotal = Number.isFinite(totalRows) && totalRows >= 0;

            setHasNext(
                result.has_next !== undefined ? Boolean(result.has_next)
                    : hasReliableTotal ? page * limit < totalRows
                        : pg.has_more !== undefined ? Boolean(pg.has_more)
                            : rows.length >= limit
            );
            setHasPrev(
                result.has_prev !== undefined ? Boolean(result.has_prev || page > 1)
                    : page > 1
            );

            if (result.role_options) {
                setRoleOptions(result.role_options);
            }
        } catch (error) {
            setError(error instanceof Error ? error.message : String(error));
            console.error("Universal Fetch Error:", error);
        } finally {
            setLoading(false);
        }
    }, [endpoint, debouncedSearch, filters, cursor, direction, page, limit]);

    useEffect(() => {
        fetchData();
    }, [fetchData, refreshTrigger]);

    // Navigasi Maju (Selanjutnya)
    const handleNextPage = useCallback(() => {
        if (!hasNext || nextCursor === null || nextCursor === undefined) return;
        setCursor(nextCursor);
        setDirection('next');
        setPageInternal((p) => p + 1);
    }, [hasNext, nextCursor]);

    // Navigasi Mundur (Sebelumnya)
    // Cursor "prev" adalah id PERTAMA baris pada halaman yang sedang tampil,
    // bukan id terakhir halaman sebelumnya. Backend memakai `id > cursor`
    // diurutkan ASC lalu membalik hasilnya, sehingga id pertama halaman ini
    // menunjuk tepat ke n baris sebelum halaman ini.
    const handlePrevPage = useCallback(() => {
        if (page <= 1) return;
        setCursor(prevCursor);
        setDirection('prev');
        setPageInternal((p) => Math.max(1, p - 1));
    }, [page, prevCursor]);

    // Fungsi reset hanya mengembalikan pencarian dan saringan kategori data murni
    const handleReset = () => {
        setSearch('');
        setFilters(initialFilters);
        resetPagination();
    };

    const updateFilter = (key: string, value: any) => {
        setFilters((prev) => ({ ...prev, [key]: value }));
        resetPagination();
    };

    return {
        search,
        setSearch,
        page,
        setPage,
        limit,
        setLimit,
        cursor,
        nextCursor,
        prevCursor,
        hasNext,
        hasPrev: hasPrev || page > 1,
        handleNextPage,
        handlePrevPage,
        filters,
        updateFilter,
        setFilters,
        data,
        total,
        loading,
        httpStatus,
        error,
        handleReset,
        roleOptions,
        isFiltered: search !== '' || JSON.stringify(filters) !== JSON.stringify(initialFilters),
        refetch: fetchData
    };
}
