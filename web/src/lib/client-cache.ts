"use client";

type CacheEntry = {
  expiresAt: number;
  value?: unknown;
  promise?: Promise<unknown>;
};

const responseCache = new Map<string, CacheEntry>();

function cacheKey(url: string, token: string) {
  return `${token.slice(-16)}:${url}`;
}

export function clearClientCache() {
  responseCache.clear();
}

export async function cachedJSON<T>(
  url: string,
  token: string,
  options: { ttlMs?: number; force?: boolean } = {},
): Promise<T> {
  const { ttlMs = 45_000, force = false } = options;
  const key = cacheKey(url, token);
  const now = Date.now();
  const cached = responseCache.get(key);

  if (!force && cached?.value !== undefined && cached.expiresAt > now) {
    return cached.value as T;
  }
  if (!force && cached?.promise) {
    return cached.promise as Promise<T>;
  }

  const promise = fetch(url, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  }).then(async (response) => {
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload?.error?.message || "Request failed.");
    }
    responseCache.set(key, {
      expiresAt: Date.now() + ttlMs,
      value: payload,
    });
    return payload;
  });

  responseCache.set(key, {
    expiresAt: now + ttlMs,
    promise,
  });

  try {
    return (await promise) as T;
  } catch (error) {
    responseCache.delete(key);
    throw error;
  }
}

export function preloadJSON<T>(
  url: string,
  token: string,
  options?: { ttlMs?: number; force?: boolean },
) {
  void cachedJSON<T>(url, token, options).catch(() => {
    // Preloading is opportunistic.
  });
}
