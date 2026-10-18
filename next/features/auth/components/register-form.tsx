"use client"

import { useRouter } from "next/navigation"
import { Controller, useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { AnimatePresence, motion } from "framer-motion"
import { LoaderCircle } from "lucide-react"
import Cookies from 'js-cookie'

import { Button } from "@/components/ui/button"
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { toast } from "@/components/ui/toast"
import { loginSchema, type LoginInput } from "../types/auth-schema"

const listVariants = {
    hidden: { opacity: 0 },
    visible: {
        opacity: 1,
        transition: {
            staggerChildren: 0.08,
        },
    },
}

const itemVariants = {
    hidden: { opacity: 0, y: 12 },
    visible: { opacity: 1, y: 0 },
}

export default function AuthForm() {
    const router = useRouter()

    // Form handler untuk Login
    const loginForm = useForm<LoginInput>({
        resolver: zodResolver(loginSchema),
        mode: "onTouched",
        defaultValues: { email: "", password: "" },
    })

    async function onLoginSubmit(data: LoginInput) {
        try {
            // Sesi diset 15 menit, nanti akan diperpanjang otomatis oleh SessionExtender jika aktif
            const fifteenMinutes = new Date(new Date().getTime() + 15 * 60 * 1000);

            const response = await fetch(`/api/auth/bff-login`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(data),
            })

            const result = await response.json().catch(() => ({}))

            console.log("ISI RESPONS DARI GO BACKEND:", result.permissions);

            if (!response.ok) {
                throw new Error(result.error ?? "Login gagal. Pastikan email dan password benar.")
            }

            // Aplikasi tujuan ditentukan backend dari role pengguna.
            const app = result.app === 'pos' ? 'pos' : 'erp'

            // 1. Simpan Utama Token JWT
            

            // 1b. Simpan aplikasi pengguna; dipakai proxy (routing) untuk gating.
            Cookies.set('user_app', app, { expires: fifteenMinutes, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/' })

            // 1c. Penanda sesi yang bisa dibaca JavaScript. Token JWT sendiri
            // kini disimpan di cookie httpOnly oleh route handler BFF, sehingga
            // halaman tidak bisa lagi memakai `auth_token` sebagai penanda login.
            Cookies.set('session_active', '1', { expires: fifteenMinutes, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/' })

            // 2. Simpan Data Pengguna (Email & Nama)
            Cookies.set('user_email', data.email, { expires: fifteenMinutes, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/' })
            if (result.name) {
                Cookies.set('user_name', result.name, { expires: fifteenMinutes, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/' })
            }

            const allowedPermissions = result.permissions || result.Permissions;

            // 3. FIX: Cek dan simpan role_id dari objek 'result' (bukan data input form)
            // if (result.role_id !== undefined) {
            //     Cookies.set('role_id', result.role_id.toString(), { expires: fifteenMinutes, secure: process.env.NODE_ENV === 'production', sameSite: 'lax', path: '/' })
            // }

            // 4. Simpan Permissions
            if (allowedPermissions && Array.isArray(allowedPermissions)) {
                console.log("Menyimpan permissions ke cookie:", allowedPermissions);
                Cookies.set('user_permissions', JSON.stringify(allowedPermissions), {
                    expires: fifteenMinutes,
                    secure: process.env.NODE_ENV === 'production',
                    sameSite: 'lax',
                    path: '/'
                })
            } else {
                console.warn("⚠️ Gagal menyimpan kuki: Data permissions dari backend kosong atau bukan array!");
            }

            toast.add({
                type: "success",
                title: "Login Berhasil",
                description: result.message ?? "Selamat datang kembali di Axion ERP.",
                timeout: 3000,
            })

            loginForm.reset()

            // Menggunakan window.location agar Next.js proxy langsung membaca cookie terbaru secara bersih
            window.location.replace(app === 'pos' ? '/pos' : '/dashboard')
        } catch (error) {
            toast.add({
                type: "error",
                title: "Login Gagal",
                description: error instanceof Error ? error.message : "Email atau password salah.",
                timeout: 3000,
            })
        }
    }


    return (
        <div className="flex min-h-svh items-center justify-center bg-muted/40 p-4 sm:p-6 lg:p-8">
            <div className="w-full max-w-[420px] rounded-2xl border border-border/80 bg-card p-6 shadow-sm sm:p-8">
                {/* Brand Logo & Welcome Title */}
                <div className="mb-6 flex flex-col items-center justify-center text-center">
                    <div className="mb-3 grid size-10 place-items-center rounded-xl bg-primary text-base font-bold text-primary-foreground shadow-xs">
                        A
                    </div>
                    <h1 className="text-xl font-semibold tracking-tight text-foreground">Welcome</h1>
                </div>

                {/* Login Form */}
                <AnimatePresence mode="wait">
                    <motion.div
                        key="login-tab"
                        initial="hidden"
                        animate="visible"
                        exit="hidden"
                        variants={listVariants}
                        className="space-y-4"
                    >
                        <form onSubmit={loginForm.handleSubmit(onLoginSubmit)} noValidate className="space-y-4">
                            <FieldGroup className="space-y-2">
                                <motion.div variants={itemVariants} transition={{ type: "tween" }}>
                                    <Controller
                                        control={loginForm.control}
                                        name="email"
                                        render={({ field, fieldState }) => (
                                            <Field data-invalid={fieldState.invalid}>
                                                <FieldLabel htmlFor="login-email">Email address</FieldLabel>
                                                <Input
                                                    {...field}
                                                    id="login-email"
                                                    type="email"
                                                    placeholder="you@company.com"
                                                    autoComplete="email"
                                                    aria-invalid={fieldState.invalid}
                                                />
                                                {fieldState.error && <FieldError>{fieldState.error.message}</FieldError>}
                                            </Field>
                                        )}
                                    />
                                </motion.div>

                                <motion.div variants={itemVariants} transition={{ type: "tween" }}>
                                    <Controller
                                        control={loginForm.control}
                                        name="password"
                                        render={({ field, fieldState }) => (
                                            <Field data-invalid={fieldState.invalid}>
                                                <div className="flex items-center justify-between">
                                                    <FieldLabel htmlFor="login-password">Password</FieldLabel>
                                                </div>
                                                <Input
                                                    {...field}
                                                    id="login-password"
                                                    type="password"
                                                    placeholder="Enter your password"
                                                    autoComplete="current-password"
                                                    aria-invalid={fieldState.invalid}
                                                />
                                                {fieldState.error && <FieldError>{fieldState.error.message}</FieldError>}
                                            </Field>
                                        )}
                                    />
                                </motion.div>
                            </FieldGroup>

                            <motion.div variants={itemVariants} transition={{ type: "tween" }} className="pt-1">
                                <Button
                                    type="submit"
                                    disabled={loginForm.formState.isSubmitting}
                                    className="w-full cursor-pointer"
                                >
                                    {loginForm.formState.isSubmitting ? (
                                        <>
                                            <LoaderCircle data-icon="inline-start" className="animate-spin" />
                                            Signing in...
                                        </>
                                    ) : (
                                        "Sign in"
                                    )}
                                </Button>
                            </motion.div>
                        </form>
                    </motion.div>
                </AnimatePresence>
            </div>
        </div>
    )
}