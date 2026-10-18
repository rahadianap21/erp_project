import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
  {
    // `react-hooks/set-state-in-effect` DIMATIKAN SEMENTARA.
    //
    // Rule ini berasal dari preset React Compiler di
    // eslint-plugin-react-hooks@7.1.1. Perbaikan resmi untuk setState yang
    // terjadi SETELAH `await` (pola data-loading yang umum dan sah) baru ada
    // di kanal kanari 7.1.1-canary-*; versi kanari juga mengaktifkan rule baru
    // `react-hooks/purity` sehingga belum bisa diadopsi sebagai ganti bersih.
    //
    // Dari 50 temuan sisa Tahap 5, mayoritas adalah setState pasca-`await`
    // (false positive). Sisa yang genuine (setLoading sinkron, sinkronisasi
    // state saat mode edit dari props, reset/clear state) dicatat di
    // "Backlog" RENCANA-REFORMASI-STRUKTUR.md untuk dikerjakan terpisah
    // per-file dengan uji e2e.
    rules: {
      "react-hooks/set-state-in-effect": "off",
    },
  },
]);

export default eslintConfig;
