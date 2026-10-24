// API_BASE_URL hanya untuk jalur yang memanggil backend secara langsung
// (route handler server + kasus lama yang belum dimigrasi). Halaman client
// seharusnya memakai lib/api-client yang lewat /api/proxy.
export const API_BASE_URL =
  process.env.API_URL ??
  process.env.NEXT_PUBLIC_API_URL ??
  "http://localhost:8090";
