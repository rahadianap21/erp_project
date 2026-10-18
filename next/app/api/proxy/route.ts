import { API_BASE_URL } from "@/lib/config"
import { cookies } from 'next/headers'
import { NextResponse } from 'next/server'

export async function GET(request: Request) {
    const url = new URL(request.url)
    const target = url.searchParams.get('target') || ''
    const cookieStore = await cookies()
    const token = cookieStore.get('auth_token')?.value
    if (!token) {
        return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }
    const res = await fetch(`${API_BASE_URL}${target}`, {
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        cache: 'no-store',
    })
    const data = await res.json().catch(() => ({}))
    return NextResponse.json(data, { status: res.status })
}

export async function POST(request: Request) {
    const url = new URL(request.url)
    const target = url.searchParams.get('target') || ''
    const body = await request.text()
    const cookieStore = await cookies()
    const token = cookieStore.get('auth_token')?.value
    if (!token) {
        return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }
    const res = await fetch(`${API_BASE_URL}${target}`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body,
    })
    const data = await res.json().catch(() => ({}))
    return NextResponse.json(data, { status: res.status })
}

export async function PUT(request: Request) {
    const url = new URL(request.url)
    const target = url.searchParams.get('target') || ''
    const body = await request.text()
    const cookieStore = await cookies()
    const token = cookieStore.get('auth_token')?.value
    if (!token) {
        return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }
    const res = await fetch(`${API_BASE_URL}${target}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body,
    })
    const data = await res.json().catch(() => ({}))
    return NextResponse.json(data, { status: res.status })
}

export async function DELETE(request: Request) {
    const url = new URL(request.url)
    const target = url.searchParams.get('target') || ''
    const cookieStore = await cookies()
    const token = cookieStore.get('auth_token')?.value
    if (!token) {
        return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
    }
    const res = await fetch(`${API_BASE_URL}${target}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` },
    })
    const data = await res.json().catch(() => ({}))
    return NextResponse.json(data, { status: res.status })
}
