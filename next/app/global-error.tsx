"use client";

import { ForbiddenView } from "@/components/forbidden-view";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const isForbidden =
    error.message?.includes("403") ||
    error.message?.toLowerCase().includes("forbidden");

  return (
    <html lang="id">
      <body>
        {isForbidden ? (
          <ForbiddenView isGlobalError={true} reset={reset} />
        ) : (
          <div className="flex min-h-screen flex-col items-center justify-center p-4 text-center bg-background text-foreground">
            <h1 className="text-3xl font-bold mb-2">Error Fatal Aplikasi</h1>
            <p className="text-muted-foreground mb-6 text-sm">
              Sistem ERP mengalami kendala kritis.
            </p>
            <button
              onClick={() => reset()}
              className="px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm font-medium"
            >
              Segarkan Aplikasi
            </button>
          </div>
        )}
      </body>
    </html>
  );
}
