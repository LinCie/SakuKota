// Types (see internal/module/auth/dto.go)

export type User = {
  id: number;
  name: string;
  email: string;
  created_at: string;
  updated_at: string;
};

/**
 * The login/refresh response. The refresh token never reaches JavaScript: the
 * backend keeps it in an HttpOnly cookie.
 */
export type Session = {
  user: User;
  access_token: string;
};
