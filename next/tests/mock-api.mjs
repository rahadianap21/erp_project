import { createServer } from "node:http";

const PORT = 8091;

const TOKENS = {
  MOCK_TOKEN_WITHOUT_PERMISSIONS: [],
  PROXIED_MOCK_JWT_TOKEN: ["users:read"],
  VALID_TOKEN_UNTUK_TEST: ["users:read"],
  VALID_MOCK_TOKEN: ["dashboard:read"],
};

const USERS = [
  {
    id: 1,
    name: "Budi Santoso",
    email: "budi@company.com",
    role_name: "Staff",
  },
  {
    id: 2,
    name: "Siti Aminah",
    email: "siti@company.com",
    role_name: "Operator",
  },
];

function send(res, status, body) {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(body === undefined ? "" : JSON.stringify(body));
}

function readBody(req) {
  return new Promise((resolve) => {
    let raw = "";
    req.on("data", (chunk) => {
      raw += chunk;
    });
    req.on("end", () => {
      try {
        resolve(raw ? JSON.parse(raw) : {});
      } catch {
        resolve({});
      }
    });
  });
}

const server = createServer(async (req, res) => {
  const url = new URL(req.url ?? "/", `http://localhost:${PORT}`);
  const token = (req.headers.authorization ?? "").replace(/^Bearer\s+/i, "");

  if (url.pathname === "/health") {
    send(res, 200, { ok: true });
    return;
  }

  if (url.pathname === "/api/login") {
    await readBody(req);
    send(res, 200, {
      message: "Login berhasil",
      token: "PROXIED_MOCK_JWT_TOKEN",
      name: "Mock User",
      email: "mock@example.com",
      role_id: 2,
      role_name: "Staff",
      app: "erp",
      permissions: ["users:read"],
    });
    return;
  }

  if (url.pathname === "/api/me") {
    if (!token) {
      send(res, 401, { error: "Unauthorized" });
      return;
    }
    send(res, 200, {
      message: "Akses disetujui",
      user_id: 1,
      name: "Mock User",
      email: "mock@example.com",
      role_id: 2,
      app: "erp",
      permissions: TOKENS[token] ?? ["dashboard:read"],
    });
    return;
  }

  if (url.pathname === "/api/users") {
    if (req.method === "DELETE") {
      send(res, 403, {
        error: "Forbidden: Akses ditolak sistem granular RBAC",
      });
      return;
    }
    send(res, 200, { data: USERS, total: USERS.length });
    return;
  }

  if (url.pathname === "/api/dashboard") {
    send(res, 200, {
      data: {
        stats: {
          total_po_value: 0,
          total_po: 0,
          po_pending: 0,
          po_approved: 0,
          po_received: 0,
          total_grn: 0,
          total_products: 0,
          total_suppliers: 0,
          low_stock_count: 0,
        },
        low_stock: [],
        recent_pos: [],
        recent_grns: [],
      },
    });
    return;
  }

  send(res, 200, { data: [] });
});

server.listen(PORT, "127.0.0.1", () => {
  console.log(`Mock API berjalan di http://localhost:${PORT}`);
});
