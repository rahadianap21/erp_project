export async function apiGet(path: string, init?: RequestInit) {
    const res = await fetch(`/api/proxy?target=${encodeURIComponent(path)}`, {
        method: 'GET',
        credentials: 'include',
        cache: 'no-store',
        ...init,
    })
    return res
}

export async function apiPost(path: string, body: any, init?: RequestInit) {
    const res = await fetch(`/api/proxy?target=${encodeURIComponent(path)}`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
        body: JSON.stringify(body),
        ...init,
    })
    return res
}

export async function apiPut(path: string, body: any, init?: RequestInit) {
    const res = await fetch(`/api/proxy?target=${encodeURIComponent(path)}`, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', ...(init?.headers || {}) },
        body: JSON.stringify(body),
        ...init,
    })
    return res
}

export async function apiDelete(path: string, init?: RequestInit) {
    const res = await fetch(`/api/proxy?target=${encodeURIComponent(path)}`, {
        method: 'DELETE',
        credentials: 'include',
        ...init,
    })
    return res
}
