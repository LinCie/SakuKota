import { afterEach, describe, expect, it, vi } from "vitest";

import {
  errorResponse,
  fakeBackend,
  fakeJwt,
  gate,
  loadApp,
  okResponse,
  sessionOf,
} from "~/shared/test-helpers";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("signing in and out", () => {
  it("starts signed out", async () => {
    const { session } = await loadApp();
    expect(session.getSession()).toBeNull();
    expect(session.getAccessToken()).toBeUndefined();
  });

  it("tells every listener when the user signs in or out", async () => {
    const { session } = await loadApp();
    const seen: (string | undefined)[] = [];
    session.subscribeSession(() => seen.push(session.getAccessToken()));

    session.saveSession(sessionOf("t1"));
    session.clearSession();

    expect(seen).toEqual(["t1", undefined]);
  });

  it("stops telling a listener that unsubscribed", async () => {
    const { session } = await loadApp();
    const listener = vi.fn();
    const unsubscribe = session.subscribeSession(listener);

    unsubscribe();
    session.saveSession(sessionOf("t1"));

    expect(listener).not.toHaveBeenCalled();
  });
});

describe("whether the user sees admin tools", () => {
  const withToken = (token: string) => ({ ...sessionOf(token) });

  it("is true for an admin's token", async () => {
    const { session } = await loadApp();
    expect(
      session.isAdmin(withToken(fakeJwt({ sub: "1", role: "admin" }))),
    ).toBe(true);
  });

  it("is true for an admin whose name has non-ASCII characters", async () => {
    const { session } = await loadApp();
    expect(
      session.isAdmin(
        withToken(fakeJwt({ name: "Ñandú 日本", role: "admin" })),
      ),
    ).toBe(true);
  });

  it.each([
    ["an ordinary user", { role: "user" }],
    ["a different casing", { role: "Admin" }],
    ["a role list", { role: ["admin"] }],
    ["no role", { sub: "1" }],
    ["a payload that is null", null],
    ["a payload that is a string", "admin"],
    ["a payload that is a number", 1],
  ])("is false for %s", async (_, claims) => {
    const { session } = await loadApp();
    expect(session.isAdmin(withToken(fakeJwt(claims)))).toBe(false);
  });

  it.each([
    ["no session at all", null],
    ["an empty token", sessionOf("")],
    ["a token without a payload", sessionOf("header-only")],
    ["a payload that isn't base64", sessionOf("a.%%%.c")],
    ["a payload that isn't JSON", sessionOf("a.bm90IGpzb24.c")],
  ])("is false, without throwing, for %s", async (_, current) => {
    const { session } = await loadApp();
    expect(session.isAdmin(current)).toBe(false);
  });

  it("reads base64url payloads, whose '-' and '_' differ from base64", async () => {
    const { session } = await loadApp();
    // Characters that encode to '-' and '_' in base64url.
    const claims = { role: "admin", pad: "???>>>~~~" };
    const token = fakeJwt(claims);
    expect(token.split(".")[1]).toMatch(/[-_]/);
    expect(session.isAdmin(withToken(token))).toBe(true);
  });
});

describe("restoring the session when the page loads", () => {
  it("signs the user in when the refresh cookie is still good", async () => {
    fakeBackend(() => okResponse(sessionOf("restored")));
    const { session } = await loadApp();

    expect(await session.restoreSession()).toBe(true);
    expect(session.getAccessToken()).toBe("restored");
  });

  it("leaves a visitor signed out when there is no valid cookie", async () => {
    fakeBackend(() => errorResponse(401, "INVALID_REFRESH_TOKEN"));
    const { session } = await loadApp();

    expect(await session.restoreSession()).toBe(false);
    expect(session.getSession()).toBeNull();
  });

  it("asks the backend only once, however many route guards wait on it", async () => {
    const backend = fakeBackend(() => okResponse(sessionOf("restored")));
    const { session } = await loadApp();

    await Promise.all([
      session.restoreSession(),
      session.restoreSession(),
      session.restoreSession(),
    ]);
    await session.restoreSession();

    expect(backend.requests).toHaveLength(1);
  });
});

describe("renewing the session", () => {
  it("replaces the token and reports success", async () => {
    fakeBackend(() => okResponse(sessionOf("new")));
    const { session } = await loadApp();
    session.saveSession(sessionOf("old"));

    expect(await session.refreshSession()).toBe(true);
    expect(session.getAccessToken()).toBe("new");
  });

  it("signs the user out when the cookie is expired, revoked or used", async () => {
    fakeBackend(() => errorResponse(401, "INVALID_REFRESH_TOKEN"));
    const { session } = await loadApp();
    session.saveSession(sessionOf("old"));

    expect(await session.refreshSession()).toBe(false);
    expect(session.getSession()).toBeNull();
  });

  it("keeps the user signed in when the server errors", async () => {
    fakeBackend(() => errorResponse(500, "INTERNAL_ERROR"));
    const { session } = await loadApp();
    session.saveSession(sessionOf("old"));

    expect(await session.refreshSession()).toBe(false);
    expect(session.getAccessToken()).toBe("old");
  });

  it("keeps the user signed in when the connection drops", async () => {
    fakeBackend(() => {
      throw new TypeError("offline");
    });
    const { session } = await loadApp();
    session.saveSession(sessionOf("old"));

    expect(await session.refreshSession()).toBe(false);
    expect(session.getAccessToken()).toBe("old");
  });

  it("keeps the user signed in when the server answers with something that isn't the API", async () => {
    fakeBackend(
      () => new Response("<html>Bad Gateway</html>", { status: 502 }),
    );
    const { session } = await loadApp();
    session.saveSession(sessionOf("old"));

    expect(await session.refreshSession()).toBe(false);
    expect(session.getAccessToken()).toBe("old");
  });

  it("sends one request for overlapping renewals, since refresh tokens are single-use", async () => {
    const backend = fakeBackend(() => okResponse(sessionOf("new")));
    const { session } = await loadApp();

    const results = await Promise.all([
      session.refreshSession(),
      session.refreshSession(),
    ]);

    expect(results).toEqual([true, true]);
    expect(backend.requests).toHaveLength(1);
  });

  it("renews again later, once the first renewal has finished", async () => {
    const backend = fakeBackend(() => okResponse(sessionOf("new")));
    const { session } = await loadApp();

    await session.refreshSession();
    await session.refreshSession();

    expect(backend.requests).toHaveLength(2);
  });

  it("can be retried after a failure", async () => {
    let online = false;
    fakeBackend(() => {
      if (!online) throw new TypeError("offline");
      return okResponse(sessionOf("new"));
    });
    const { session } = await loadApp();

    expect(await session.refreshSession()).toBe(false);
    online = true;
    expect(await session.refreshSession()).toBe(true);
  });

  it("does not sign the user back in after they signed out mid-renewal", async () => {
    const release = gate<Response>();
    const backend = fakeBackend(() => release.promise);
    const { session } = await loadApp();
    session.saveSession(sessionOf("old"));

    const renewal = session.refreshSession();
    await vi.waitFor(() => expect(backend.requests).toHaveLength(1));
    session.clearSession();
    release.open(okResponse(sessionOf("new")));

    expect(await renewal).toBe(false);
    expect(session.getSession()).toBeNull();
  });

  it("does not overwrite a new sign-in made mid-renewal", async () => {
    const release = gate<Response>();
    const backend = fakeBackend(() => release.promise);
    const { session } = await loadApp();
    session.saveSession(sessionOf("old"));

    const renewal = session.refreshSession();
    await vi.waitFor(() => expect(backend.requests).toHaveLength(1));
    session.saveSession(sessionOf("signed-in-again"));
    release.open(okResponse(sessionOf("stale")));

    expect(await renewal).toBe(false);
    expect(session.getAccessToken()).toBe("signed-in-again");
  });
});
