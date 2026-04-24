import { ApiErrorShape } from "@/lib/types";

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function parseResponse<T>(response: Response): Promise<T | null> {
  if (response.status === 204) {
    return null;
  }

  const text = await response.text();
  const data = text ? (JSON.parse(text) as T) : null;

  if (!response.ok) {
    const errorData = (data ?? {}) as ApiErrorShape;
    throw new ApiError(errorData.message ?? response.statusText, response.status);
  }

  return data;
}

export async function apiRequest<T>(
  input: RequestInfo,
  init?: RequestInit
): Promise<T | null> {
  const response = await fetch(input, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    },
    cache: "no-store"
  });

  return parseResponse<T>(response);
}
