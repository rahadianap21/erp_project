import { cookies } from "next/headers";

export async function getSessionData() {
  const cookieStore = await cookies();
  const permissionsCookie = cookieStore.get("user_permissions")?.value;
  let permissions: string[] = [];
  if (permissionsCookie) {
    try {
      const parsed = JSON.parse(decodeURIComponent(permissionsCookie));
      if (Array.isArray(parsed)) {
        permissions = parsed;
      }
    } catch {
      permissions = [];
    }
  }

  const storedEmail = cookieStore.get("user_email")?.value;
  const storedName = cookieStore.get("user_name")?.value;
  let user = {
    name: storedName || "Axion User",
    email: storedEmail || "user@axion.erp",
    avatar: "",
  };
  if (!storedName && storedEmail) {
    const username = storedEmail.split("@")[0];
    const formattedName = username
      .replace(/[-_.]/g, " ")
      .replace(/\b\w/g, (c) => c.toUpperCase());
    user = {
      name: formattedName || "Admin User",
      email: storedEmail,
      avatar: "",
    };
  }

  return { permissions, user };
}
