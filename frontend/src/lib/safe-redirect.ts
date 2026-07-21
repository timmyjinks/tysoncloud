/**
 * The only search param that gets fed into a URL/redirect: keep it to
 * same-origin relative paths so a crafted `?redirect=` can't send someone
 * to an external site after they sign in.
 */
export function safeRedirectTarget(target: string | undefined, fallback = "/dashboard") {
  if (!target) return fallback;
  if (!target.startsWith("/") || target.startsWith("//")) return fallback;
  return target;
}
