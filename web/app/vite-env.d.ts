/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL of the rantau-go backend API; defaults to "/api" (proxied by Vite). */
  readonly VITE_API_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
