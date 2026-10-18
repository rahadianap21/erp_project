"use client";

import React from "react";
import { useRouter } from "next/navigation";
import { ShieldAlert, ArrowLeft, Home } from "lucide-react";
import { Button } from "@/components/ui/button";

interface ForbiddenViewProps {
    isGlobalError?: boolean;
    reset?: () => void;
}

export function ForbiddenView({ isGlobalError = false, reset }: ForbiddenViewProps) {
    const router = useRouter();

    return (
        <div className="flex min-h-screen flex-col items-center justify-center bg-background p-4 text-foreground">
            <div className="w-full max-w-md text-center space-y-6">
                <div className="mx-auto flex h-20 w-20 items-center justify-center rounded-full bg-destructive/10 text-destructive animate-pulse">
                    <ShieldAlert className="h-10 w-10" />
                </div>

                <div className="space-y-2">
                    <h1 className="text-4xl font-extrabold tracking-tight sm:text-5xl">
                        403 - Forbidden
                    </h1>
                    <p className="text-lg font-semibold text-muted-foreground">
                        Akses Ditolak / Tidak Sah
                    </p>
                    <p className="text-sm text-muted-foreground/80 max-w-xs mx-auto">
                        Maaf, akun Anda tidak memiliki izin yang diperlukan untuk melihat halaman ini.
                    </p>
                </div>

                <div className="flex flex-col sm:flex-row gap-3 justify-center pt-4">
                    {reset ? (
                        <Button variant="outline" onClick={() => reset()} className="inline-flex items-center gap-2">
                            Coba Lagi
                        </Button>
                    ) : (
                        <Button
                            variant="outline"
                            onClick={() => router.back()}
                            className="inline-flex items-center gap-2"
                        >
                            <ArrowLeft className="h-4 w-4" />
                            Kembali
                        </Button>
                    )}
                    <Button
                        onClick={() => {
                            if (isGlobalError) {
                                window.location.href = "/dashboard";
                            } else {
                                router.push("/dashboard");
                            }
                        }}
                        className="inline-flex items-center gap-2"
                    >
                        <Home className="h-4 w-4" />
                        Ke Dashboard
                    </Button>
                </div>
            </div>
        </div>
    );
}
