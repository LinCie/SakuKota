import { afterEach, describe, expect, it, vi } from "vitest";

import {
  errorResponse,
  fakeBackend,
  loadApp,
  okResponse,
  routeOf,
  sessionOf,
} from "./test-helpers";

afterEach(() => {
  vi.unstubAllGlobals();
});

const bearer = (request: Request) => request.headers.get("Authorization");

describe("calling the backend", () => {
  it("sends the signed-in user's token, and nothing when signed out", async () => {
    const backend = fakeBackend(() => okResponse([]));
    const { api, session } = await loadApp();

    await api.apiGet("places");
    session.saveSession(sessionOf("token-1"));
    await api.apiGet("places");

    expect(backend.requests.map(bearer)).toEqual([null, "Bearer token-1"]);
  });

  it("returns the data of a success", async () => {
    fakeBackend(() => okResponse({ id: 7 }));
    const { api } = await loadApp();
    expect(await api.apiGet("places/7")).toEqual({
      ok: true,
      data: { id: 7 },
    });
  });

  it("returns a rejection as a result with its status and code, not a throw", async () => {
    fakeBackend(() => errorResponse(404, "PLACE_NOT_FOUND", "tidak ada"));
    const { api } = await loadApp();
    expect(await api.apiGet("places/7")).toEqual({
      ok: false,
      status: 404,
      error: { code: "PLACE_NOT_FOUND", message: "tidak ada" },
    });
  });

  it("treats a 204 as success without a body", async () => {
    fakeBackend(() => new Response(null, { status: 204 }));
    const { api } = await loadApp();
    expect(await api.apiPost("auth/logout")).toEqual({
      ok: true,
      data: undefined,
    });
  });

  it("does not retry a failed write when the connection drops", async () => {
    const backend = fakeBackend(() => {
      throw new TypeError("Failed to fetch");
    });
    const { api } = await loadApp();
    await expect(api.apiPost("places", { json: {} })).rejects.toThrow(
      /network error/,
    );
    expect(backend.requests).toHaveLength(1);
  });
});

describe("a response that isn't the backend's", () => {
  it.each([
    [
      "an HTML error page from a proxy",
      () => new Response("<html>Bad Gateway</html>", { status: 502 }),
    ],
    ["an empty 200", () => new Response("", { status: 200 })],
    ["a success without data", () => Response.json({ ok: true })],
    [
      "an error without a code",
      () => Response.json({ message: "x" }, { status: 500 }),
    ],
    [
      "an error with a numeric code",
      () => Response.json({ code: 5, message: "x" }, { status: 500 }),
    ],
  ])("throws a protocol error for %s", async (_, respond) => {
    fakeBackend(respond);
    const { api } = await loadApp();
    await expect(api.apiGet("places")).rejects.toThrow(api.ApiProtocolError);
  });

  it("accepts a success whose data is null", async () => {
    fakeBackend(() => okResponse(null));
    const { api } = await loadApp();
    expect(await api.apiGet("places")).toEqual({ ok: true, data: null });
  });
});

describe("unwrap", () => {
  it("throws an error carrying the status and code of a rejection", async () => {
    fakeBackend(() => errorResponse(409, "EMAIL_TAKEN", "sudah dipakai"));
    const { api } = await loadApp();
    const result = await api.apiPost("auth/register", { json: {} });

    const failure = (() => {
      try {
        api.unwrap(result);
      } catch (error) {
        return error;
      }
    })();
    expect(failure).toBeInstanceOf(api.ApiError);
    expect(failure).toMatchObject({
      status: 409,
      code: "EMAIL_TAKEN",
      message: "sudah dipakai",
    });
  });
});

describe("when the access token has expired", () => {
  /** A backend where "old" is expired, refreshing yields "new", and "new" works. */
  function backendWithExpiredToken(refreshResponse?: () => Response) {
    return fakeBackend((request) => {
      switch (routeOf(request)) {
        case "auth/refresh":
          return refreshResponse?.() ?? okResponse(sessionOf("new"));
        case "places":
          return bearer(request) === "Bearer new"
            ? okResponse({ id: 1 })
            : errorResponse(401, "UNAUTHORIZED");
        default:
          return errorResponse(404, "NOT_FOUND");
      }
    });
  }

  it("renews the token and repeats the request, even a write", async () => {
    const backend = backendWithExpiredToken();
    const { api, session } = await loadApp();
    session.saveSession(sessionOf("old"));

    const result = await api.apiPost("places", { json: { name: "Warung" } });

    expect(result).toEqual({ ok: true, data: { id: 1 } });
    expect(backend.to("places").map(bearer)).toEqual([
      "Bearer old",
      "Bearer new",
    ]);
    expect(session.getAccessToken()).toBe("new");
  });

  it("sends the same body again on the repeat", async () => {
    const bodies: string[] = [];
    fakeBackend(async (request) => {
      if (routeOf(request) === "auth/refresh") {
        return okResponse(sessionOf("new"));
      }
      bodies.push(await request.text());
      return bearer(request) === "Bearer new"
        ? okResponse({})
        : errorResponse(401, "UNAUTHORIZED");
    });
    const { api, session } = await loadApp();
    session.saveSession(sessionOf("old"));

    await api.apiPost("places", { json: { name: "Warung" } });

    expect(bodies).toEqual(['{"name":"Warung"}', '{"name":"Warung"}']);
  });

  it("renews once for several requests failing together", async () => {
    const backend = backendWithExpiredToken();
    const { api, session } = await loadApp();
    session.saveSession(sessionOf("old"));

    const results = await Promise.all([
      api.apiPost("places", { json: {} }),
      api.apiPost("places", { json: {} }),
      api.apiPost("places", { json: {} }),
    ]);

    expect(results.every((result) => result.ok)).toBe(true);
    expect(backend.to("auth/refresh")).toHaveLength(1);
  });

  it("gives up with the 401 when the session can't be renewed, and signs the user out", async () => {
    const backend = backendWithExpiredToken(() =>
      errorResponse(401, "INVALID_REFRESH_TOKEN"),
    );
    const { api, session } = await loadApp();
    session.saveSession(sessionOf("old"));

    const result = await api.apiPost("places", { json: {} });

    expect(result).toMatchObject({ ok: false, status: 401 });
    expect(backend.to("places")).toHaveLength(1);
    expect(session.getSession()).toBeNull();
  });

  it("keeps the user signed in when the renewal only failed to connect", async () => {
    fakeBackend((request) => {
      if (routeOf(request) === "auth/refresh") throw new TypeError("offline");
      return errorResponse(401, "UNAUTHORIZED");
    });
    const { api, session } = await loadApp();
    session.saveSession(sessionOf("old"));

    const result = await api.apiPost("places", { json: {} });

    expect(result).toMatchObject({ ok: false, status: 401 });
    expect(session.getAccessToken()).toBe("old");
  });

  it("does not loop when the renewed token is rejected too", async () => {
    const backend = fakeBackend((request) =>
      routeOf(request) === "auth/refresh"
        ? okResponse(sessionOf("new"))
        : errorResponse(401, "UNAUTHORIZED"),
    );
    const { api, session } = await loadApp();
    session.saveSession(sessionOf("old"));

    const result = await api.apiPost("places", { json: {} });

    expect(result).toMatchObject({ ok: false, status: 401 });
    expect(backend.to("places")).toHaveLength(2);
    expect(backend.to("auth/refresh")).toHaveLength(1);
  });
});

describe("a 401 that isn't an expired token", () => {
  it("is not retried for a visitor who isn't signed in", async () => {
    const backend = fakeBackend(() => errorResponse(401, "UNAUTHORIZED"));
    const { api } = await loadApp();

    const result = await api.apiPost("places", { json: {} });

    expect(result).toMatchObject({ ok: false, status: 401 });
    expect(backend.requests).toHaveLength(1);
  });

  it("is not retried for a wrong password, even while signed in", async () => {
    const backend = fakeBackend(() =>
      errorResponse(401, "INVALID_CREDENTIALS"),
    );
    const { api, session } = await loadApp();
    session.saveSession(sessionOf("token"));

    const result = await api.apiPost("auth/login", { json: {} });

    expect(result).toMatchObject({
      ok: false,
      status: 401,
      error: { code: "INVALID_CREDENTIALS" },
    });
    expect(backend.requests).toHaveLength(1);
  });

  it("is not retried for a refused renewal, which would recurse", async () => {
    const backend = fakeBackend(() =>
      errorResponse(401, "INVALID_REFRESH_TOKEN"),
    );
    const { session } = await loadApp();
    session.saveSession(sessionOf("token"));

    await session.refreshSession();

    expect(backend.to("auth/refresh")).toHaveLength(1);
  });

  it("is passed on for a 403, which renewing can't fix", async () => {
    const backend = fakeBackend(() => errorResponse(403, "FORBIDDEN"));
    const { api, session } = await loadApp();
    session.saveSession(sessionOf("token"));

    const result = await api.apiPost("places", { json: {} });

    expect(result).toMatchObject({ ok: false, status: 403 });
    expect(backend.requests).toHaveLength(1);
  });
});
