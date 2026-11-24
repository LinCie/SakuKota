import { z } from "zod";

import { signUpSchema } from "~/features/auth/schemas";

// The display name follows the sign-up rules (internal/domain/user.go).
export const profileSchema = z.object({ name: signUpSchema.shape.name });
