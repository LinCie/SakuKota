# Agent Instructions

## Styling

- Use Tailwind CSS utility classes for all styling.
- Never write raw CSS, including stylesheet rules or inline styles. The one exception is shadcn/ui's theme blocks in `app/app.css` (`:root` variables, `@theme inline`, `@layer base`, and its imports), which follow shadcn's convention.
- Reusable primitives come from shadcn/ui and live in `app/shared/components/ui/`.
