import { useForm, useSelector } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { login, register } from "~/features/auth/api";
import { signUpSchema } from "~/features/auth/schemas";
import { saveSession } from "~/features/auth/session";
import { FormAlert } from "~/shared/components/form-alert";
import { SubmitPlate } from "~/shared/components/submit-plate";
import { TextField } from "~/shared/components/field";

type SignUpPageProps = {
  /** Where to go after signing in: the page the visitor came from. */
  kembali?: string;
};

/** The sign-up form, under the auth plates. */
export function SignUpPage({ kembali }: SignUpPageProps) {
  const navigate = useNavigate();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm({
    defaultValues: { name: "", email: "", password: "" },
    validators: { onSubmit: signUpSchema },
    onSubmit: async ({ value }) => {
      setFormError(null);
      try {
        const created = await register(value);
        if (!created.ok) {
          setFormError(
            created.error.code === "EMAIL_TAKEN"
              ? "Email ini sudah terdaftar. Silakan masuk."
              : "Terjadi kesalahan di server. Coba lagi.",
          );
          return;
        }

        // Registering returns no tokens, so sign in with the same details.
        const signedIn = await login({
          email: value.email,
          password: value.password,
        });
        if (!signedIn.ok) {
          await navigate({ to: "/masuk", search: { kembali } });
          return;
        }
        saveSession(signedIn.data);
        await navigate({ href: kembali ?? "/" });
      } catch {
        setFormError("Tidak bisa terhubung ke server. Coba lagi.");
      }
    },
  });
  const pending = useSelector(form.store, (state) => state.isSubmitting);

  return (
    <form
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
      className="flex flex-col gap-4"
    >
      {formError ? <FormAlert>{formError}</FormAlert> : null}

      <form.Field name="name">
        {(field) => (
          <TextField field={field} label="Nama" autoComplete="name" />
        )}
      </form.Field>
      <form.Field name="email">
        {(field) => (
          <TextField
            field={field}
            label="Email"
            type="email"
            autoComplete="email"
          />
        )}
      </form.Field>
      <form.Field name="password">
        {(field) => (
          <TextField
            field={field}
            label="Kata sandi"
            type="password"
            autoComplete="new-password"
            hint="Minimal 8 karakter."
          />
        )}
      </form.Field>

      <SubmitPlate pending={pending}>Buat akun</SubmitPlate>
    </form>
  );
}
