import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./api";
import {
  ImageUploadError,
  imageErrorMessage,
  uploadImage,
  type MediaUpload,
} from "./image-upload";

const MB = 1024 * 1024;

/** A file of the given type and size, without allocating the bytes. */
function fakeFile(type: string, size: number) {
  const file = new File([], "upload", { type });
  Object.defineProperty(file, "size", { value: size });
  return file;
}

const upload: MediaUpload = {
  upload_key: "key-1",
  upload_url: "https://storage.example/put?sig=abc",
  method: "PUT",
  headers: { "Content-Type": "image/png", "x-amz-meta": "1" },
  expires_at: "2030-01-01T00:00:00Z",
};

function setup(response: Response | Error = new Response(null)) {
  const presign = vi.fn().mockResolvedValue(upload);
  const fetchMock = vi.fn(async () => {
    if (response instanceof Error) throw response;
    return response;
  });
  vi.stubGlobal("fetch", fetchMock);
  return { presign, fetchMock };
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("uploadImage: rejected before upload", () => {
  it.each([
    ["image/gif", 1],
    ["image/svg+xml", 1],
    ["application/pdf", 1],
    ["video/quicktime", 1],
    // Browsers report "" for extensions they don't know.
    ["", 1],
  ])("rejects %j", async (type, size) => {
    const { presign, fetchMock } = setup();
    await expect(uploadImage(fakeFile(type, size), presign)).rejects.toThrow(
      ImageUploadError,
    );
    expect(presign).not.toHaveBeenCalled();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it.each([
    ["image/jpeg", 5 * MB + 1, "Ukuran gambar maksimal 5 MB."],
    ["image/png", 10 * MB, "Ukuran gambar maksimal 5 MB."],
    ["image/webp", 5 * MB + 1, "Ukuran gambar maksimal 5 MB."],
    ["video/mp4", 50 * MB + 1, "Ukuran video maksimal 50 MB."],
    ["video/webm", 100 * MB, "Ukuran video maksimal 50 MB."],
  ])("rejects %s of %d bytes", async (type, size, message) => {
    const { presign, fetchMock } = setup();
    await expect(uploadImage(fakeFile(type, size), presign)).rejects.toThrow(
      message,
    );
    expect(presign).not.toHaveBeenCalled();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

describe("uploadImage: size limits", () => {
  it.each([
    ["image/jpeg", 5 * MB],
    ["image/png", 0],
    // Over the image limit, but videos have their own.
    ["video/mp4", 10 * MB],
    ["video/webm", 50 * MB],
  ])("accepts %s of %d bytes", async (type, size) => {
    const { presign } = setup();
    await expect(uploadImage(fakeFile(type, size), presign)).resolves.toBe(
      "key-1",
    );
  });
});

describe("uploadImage: sending", () => {
  it("presigns for the file's content type", async () => {
    const { presign } = setup();
    await uploadImage(fakeFile("image/webp", 1), presign);
    expect(presign).toHaveBeenCalledExactlyOnceWith("image/webp");
  });

  it("sends the file to the presigned URL with exactly the given method and headers", async () => {
    const { presign, fetchMock } = setup();
    const file = fakeFile("image/png", 1);
    await uploadImage(file, presign);
    expect(fetchMock).toHaveBeenCalledExactlyOnceWith(upload.upload_url, {
      method: "PUT",
      headers: upload.headers,
      body: file,
    });
  });

  it("lets a presign failure through unchanged, without uploading", async () => {
    const { fetchMock } = setup();
    const failure = new ApiError(403, { code: "FORBIDDEN", message: "no" });
    const presign = vi.fn().mockRejectedValue(failure);
    await expect(uploadImage(fakeFile("image/png", 1), presign)).rejects.toBe(
      failure,
    );
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it.each([400, 403, 500, 503])(
    "throws ImageUploadError when storage answers %d",
    async (status) => {
      const { presign } = setup(new Response(null, { status }));
      await expect(
        uploadImage(fakeFile("image/png", 1), presign),
      ).rejects.toThrow(
        new ImageUploadError("File gagal diunggah. Coba lagi."),
      );
    },
  );

  it("lets a network failure through as the original error", async () => {
    const offline = new TypeError("Failed to fetch");
    const { presign } = setup(offline);
    await expect(uploadImage(fakeFile("image/png", 1), presign)).rejects.toBe(
      offline,
    );
  });
});

describe("imageErrorMessage", () => {
  it("shows an upload error's own message", () => {
    expect(
      imageErrorMessage(new ImageUploadError("Ukuran video maksimal")),
    ).toBe("Ukuran video maksimal");
  });

  it("explains a network failure as a connection problem", () => {
    expect(imageErrorMessage(new TypeError("Failed to fetch"))).toMatch(
      /terhubung/,
    );
  });

  it("explains an expired session", () => {
    expect(
      imageErrorMessage(
        new ApiError(401, { code: "UNAUTHORIZED", message: "x" }),
      ),
    ).toMatch(/Sesimu berakhir/);
  });
});
