// Instrumentation server Next.js. Dipakai untuk menangkap error yang terjadi
// di Server Component, route handler, server action, dan proxy. Laporan
// dikirim langsung ke backend; header/cookie/query tidak pernah disertakan.
export async function onRequestError(
  error: unknown,
  request: Readonly<{ path: string; method: string }>,
  context: Readonly<{
    routerKind: string;
    routePath: string;
    routeType: string;
  }>
): Promise<void> {
  if (process.env.NEXT_RUNTIME !== "nodejs") return;

  const { reportServerError } = await import("./lib/error-tracking.server");
  await reportServerError(error, {
    path: request.path,
    method: request.method,
    routePath: context.routePath,
    routeType: context.routeType,
  });
}
