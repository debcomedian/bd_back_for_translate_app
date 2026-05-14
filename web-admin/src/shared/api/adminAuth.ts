import { api } from "./client";
import type { AuthResponse } from "../../types";

export async function adminLogin(payload: {
  login: string;
  password: string;
}): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>("/admin/login", payload);
  return data;
}
