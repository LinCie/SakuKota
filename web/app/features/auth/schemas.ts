import { z } from "zod";

// Mirrors the backend's rules (internal/module/auth/service.go).

const email = z.email("Format email belum benar, contoh: nama@email.com.");

export const signInSchema = z.object({
  email,
  password: z.string().min(1, "Kata sandi wajib diisi."),
});

export const signUpSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, "Nama wajib diisi.")
    .max(100, "Nama maksimal 100 karakter."),
  email,
  password: z.string().min(8, "Kata sandi minimal 8 karakter."),
});
