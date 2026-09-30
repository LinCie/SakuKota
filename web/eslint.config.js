import js from "@eslint/js";
import eslintConfigPrettier from "eslint-config-prettier/flat";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import reactX from "eslint-plugin-react-x";
import tseslint from "typescript-eslint";

const tsFiles = ["**/*.{ts,tsx}"];

// react-hooks owns the hooks rules; turn off react-x's copies of them.
const reactXHooksRulesOff = Object.fromEntries(
  Object.keys(reactHooks.rules)
    .filter((rule) => rule in reactX.rules)
    .map((rule) => [`react-x/${rule}`, "off"]),
);

export default tseslint.config(
  {
    ignores: [
      ".agents/**",
      "dist/**",
      "app/routeTree.gen.ts",
      "node_modules/**",
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    ...reactX.configs["recommended-typescript"],
    files: tsFiles,
    rules: {
      ...reactX.configs["recommended-typescript"].rules,
      ...reactXHooksRulesOff,
    },
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  },
  {
    ...reactHooks.configs.flat.recommended,
    files: tsFiles,
  },
  eslintConfigPrettier,
);
