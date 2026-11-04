/**
 * Access paths such as `project:/trunk` carry `:` and `/`, which proxies decode
 * differently (plan section 4.2). The API therefore accepts the URL-safe
 * base64 form in the path segment; these helpers mirror that contract.
 */

export function encodeAccessPath(path: string): string {
  const bytes = new TextEncoder().encode(path);
  let binary = '';

  for (const byte of bytes) {
    binary += String.fromCharCode(byte);
  }

  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/** Returns null when the segment is not valid URL-safe base64. */
export function decodeAccessPath(value: string): string | null {
  try {
    const normalised = value.replace(/-/g, '+').replace(/_/g, '/');
    const padded = normalised + '='.repeat((4 - (normalised.length % 4)) % 4);
    const binary = atob(padded);
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0));

    return new TextDecoder('utf-8', { fatal: true }).decode(bytes);
  } catch {
    return null;
  }
}
