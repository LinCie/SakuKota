import { Button as BaseButton } from "@base-ui/react/button";
import type { ComponentProps } from "react";

import { cn } from "~/shared/cn";

const variants = {
  primary: "bg-board text-paint border-board hover:bg-board-deep",
  secondary: "bg-paint text-ink border-ink/20 hover:bg-paint-deep",
  ghost: "text-ink border-transparent hover:bg-paint-deep",
  danger: "bg-chili text-paint border-chili hover:opacity-90",
} as const;

export type ButtonVariant = keyof typeof variants;

type ButtonProps = ComponentProps<typeof BaseButton> & {
  variant?: ButtonVariant;
};

/** A compact button drawn on base-ui's Button, in the admin's painted-enamel tones. */
export function Button({
  variant = "secondary",
  className,
  ...props
}: ButtonProps) {
  return (
    <BaseButton
      className={cn(
        "focus-visible:outline-cat inline-flex h-9 shrink-0 items-center justify-center gap-2 rounded-[3px] border px-3 text-sm font-semibold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-50",
        variants[variant],
        className,
      )}
      {...props}
    />
  );
}
