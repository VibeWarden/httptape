const API_URL = import.meta.env.VITE_API_URL || "http://localhost:3001";

export interface Product {
  id: number;
  name: string;
  price: number;
  description: string;
  stock?: number;
  category?: string;
}

export interface Cart {
  cart_id: string;
  items: unknown[];
  created_at: string;
}

export interface UserProfile {
  name: string;
  email: string;
  phone: string;
  card: {
    number: string;
    expiry: string;
    cvv: string;
  };
  address: string;
}

// DataSource mirrors httptape's SourceState. The source for the *current* page
// is driven by the SSE health stream (see useHealthStream.ts), not derived from
// individual responses. The X-Httptape-Source header is still set per-request
// by the proxy if you want to inspect it directly.
export type DataSource = "upstream" | "l1-cache" | "l2-cache";

export interface ApiResponse<T> {
  data: T;
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<ApiResponse<T>> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`);
  }
  const data: T = await res.json();
  return { data };
}

export function getProducts(): Promise<ApiResponse<Product[]>> {
  return apiFetch<Product[]>("/api/products");
}

export function getProduct(id: number): Promise<ApiResponse<Product>> {
  return apiFetch<Product>(`/api/products/${id}`);
}

export function createCart(): Promise<ApiResponse<Cart>> {
  return apiFetch<Cart>("/api/cart", { method: "POST" });
}

export function getProfile(): Promise<ApiResponse<UserProfile>> {
  return apiFetch<UserProfile>("/api/profile");
}
