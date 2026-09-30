// -----------------------------------------------------------------------------
// Types (see internal/module/auth/dto.go)
// -----------------------------------------------------------------------------

export type User = {
  id: number;
  name: string;
  email: string;
  created_at: string;
  updated_at: string;
};

export type Tokens = {
  access_token: string;
  refresh_token: string;
};

export type LoginResponse = Tokens & {
  user: User;
};
