// Types (see internal/module/users/dto.go)

/** The signed-in user's profile. */
export type Profile = {
  id: number;
  /** The display name, shown on the user's reviews. */
  name: string;
  email: string;
  role: "user" | "admin";
  /** Null without a profile photo. */
  avatar_url: string | null;
  created_at: string;
  updated_at: string;
};
