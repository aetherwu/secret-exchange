const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:5000";

function getAuth(): { openID: string; accessToken: string } | null {
  if (typeof window === "undefined") return null;
  const openID = localStorage.getItem("openID");
  const accessToken = localStorage.getItem("accessToken");
  if (!openID || !accessToken) return null;
  return { openID, accessToken };
}

function authHeader(): string {
  const auth = getAuth();
  if (!auth) return "";
  return btoa(`${auth.openID};${auth.accessToken};0x1072D`);
}

export function isLoggedIn(): boolean {
  return getAuth() !== null;
}

export function logout() {
  localStorage.removeItem("openID");
  localStorage.removeItem("accessToken");
  window.location.href = "/login";
}

export function saveAuth(openID: string, accessToken: string) {
  localStorage.setItem("openID", openID);
  localStorage.setItem("accessToken", accessToken);
}

export function getOpenID(): string | null {
  return typeof window !== "undefined"
    ? localStorage.getItem("openID")
    : null;
}

export async function api<T = Record<string, unknown>>(
  path: string,
  body?: Record<string, unknown>
): Promise<T & { errCode: number; errMsg?: string }> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };

  const token = authHeader();
  if (token) {
    headers["Authorization"] = token;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers,
    body: body ? JSON.stringify(body) : "{}",
  });

  const data = await res.json();

  if (data.errCode === 1006) {
    logout();
    throw new Error("Session expired");
  }

  return data;
}
