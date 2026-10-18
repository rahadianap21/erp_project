// src/features/auth/types/auth-schema.ts
import { z } from "zod";

export const loginSchema = z.object({
    email: z
        .string()
        .trim()
        .min(1, { message: "Email wajib diisi" })
        .email({ message: "Format email tidak valid" }),
    password: z
        .string()
        .min(1, { message: "Password wajib diisi" }),
});

export type LoginInput = z.infer<typeof loginSchema>;
