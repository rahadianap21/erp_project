import { API_BASE_URL } from "@/lib/api"
import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'

export async function POST(request: Request) {
    const { email, password } = await request.json()

    const res = await fetch(`${API_BASE_URL}/api/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
    })

    const data = await res.json().catch(() => ({}))

    if (!res.ok) {
        return NextResponse.json({ error: data.error || 'Login gagal' }, { status: res.status })
    }

    const cookieStore = await cookies()
    cookieStore.set('auth_token', data.token, {
        path: '/',
        httpOnly: true,
        secure: process.env.NODE_ENV === 'production',
        sameSite: 'lax',
        maxAge: 60 * 60 * 24,
    })

    return NextResponse.json({ message: 'Login berhasil', name: data.name, permissions: data.permissions, app: data.app })
}
