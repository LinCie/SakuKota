import { useForm, useSelector } from "@tanstack/react-form";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { login } from "~/features/auth/api";
import { signInSchema } from "~/features/auth/schemas";
import { saveSession } from "~/features/auth/session";
import { FormAlert } from "~/shared/components/form-alert";
import { SubmitPlate } from "~/shared/components/submit-plate";
import { TextField } from "~/shared/components/field";

type SignInPageProps = {
  /** Where to go after signing in: the page the visitor came from. */
  kembali?: string;
};

/** The sign-in form, under the auth plates. */
export function SignInPage({ kembali }: SignInPageProps) {
  const navigate = useNavigate();
  const [formError, setFormError] = useState<string | null>(null);

  const form = useForm({
    defaultValues: { email: "", password: "" },
    validators: { onSubmit: signInSchema },
    onSubmit: async ({ value }) => {
      setFormError(null);
      try {
        const result = await login(value);
        if (!result.ok) {
          setFormError(
            result.error.code === "INVALID_CREDENTIALS"
              ? "Email atau kata sandi salah."
              : "Terjadi kesalahan di server. Coba lagi.",
          );
          return;
        }
        saveSession(result.data);
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
            autoComplete="current-password"
          />
        )}
      </form.Field>

      <SubmitPlate pending={pending}>Masuk</SubmitPlate>
    </form>
  );
}
