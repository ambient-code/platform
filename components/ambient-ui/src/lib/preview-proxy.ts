import { isAllowedPreviewHost } from '@/lib/preview-hosts'

type ValidResult = { valid: true; parsed: URL }
type InvalidResult = { valid: false; reason: string }
type ValidationResult = ValidResult | InvalidResult

const ALLOWED_PROTOCOLS = new Set(['http:', 'https:'])

/**
 * Validate a preview URL: parseable, http(s), and on the trusted allowlist.
 */
export function validatePreviewUrl(url: string): ValidationResult {
  let parsed: URL
  try {
    parsed = new URL(url)
  } catch {
    return { valid: false, reason: 'URL is not valid' }
  }

  if (!ALLOWED_PROTOCOLS.has(parsed.protocol)) {
    return { valid: false, reason: `Protocol "${parsed.protocol}" is not allowed` }
  }

  if (!isAllowedPreviewHost(url)) {
    return { valid: false, reason: 'URL is not on the trusted preview hosts allowlist' }
  }

  return { valid: true, parsed }
}

/**
 * Create a copy of the given headers with frame-blocking headers removed.
 *
 * - Deletes `X-Frame-Options`
 * - Deletes `Content-Security-Policy-Report-Only`
 * - Strips `frame-ancestors` directives from `Content-Security-Policy`
 *   (removes the header entirely if no directives remain)
 */
export function stripFrameBlockingHeaders(headers: Headers): Headers {
  const result = new Headers(headers)

  result.delete('x-frame-options')
  result.delete('content-security-policy-report-only')

  const csp = result.get('content-security-policy')
  if (csp) {
    const filtered = csp
      .split(';')
      .map((d) => d.trim())
      .filter((d) => !d.startsWith('frame-ancestors'))
      .join('; ')

    if (filtered.length === 0) {
      result.delete('content-security-policy')
    } else {
      result.set('content-security-policy', filtered)
    }
  }

  return result
}

/**
 * Inject a `<base href="...">` tag and a navigation interceptor script
 * into HTML so it works correctly inside the preview proxy iframe.
 *
 * The `<base>` tag resolves relative sub-resource URLs (CSS, JS, images)
 * against the original server. The interceptor script rewrites navigations
 * (links, form submissions) to route through the preview proxy, preventing
 * the iframe from navigating directly to origins that block framing.
 */
export function injectBaseTag(html: string, baseUrl: string): string {
  const interceptScript = `<script>
(function(){
  var base="${baseUrl}";
  var proxy=window.location.origin+"/api/preview-proxy?url=";
  function resolve(u){
    try{return new URL(u,base).href}catch(e){return u}
  }
  function proxied(u){
    var abs=resolve(u);
    if(abs.startsWith("http"))return proxy+encodeURIComponent(abs);
    return u;
  }
  document.addEventListener("click",function(e){
    var a=e.target.closest("a[href]");
    if(!a)return;
    var h=a.getAttribute("href");
    if(!h||h.startsWith("#")||h.startsWith("javascript:"))return;
    e.preventDefault();
    window.location.href=proxied(h);
  },true);
  document.addEventListener("submit",function(e){
    var f=e.target;
    if(!f||!f.action)return;
    e.preventDefault();
    var action=f.getAttribute("action")||"";
    var abs=resolve(action);
    var fd=new FormData(f);
    if((f.method||"GET").toUpperCase()==="GET"){
      var target=new URL(abs);
      fd.forEach(function(v,k){target.searchParams.set(k,v)});
      window.location.href=proxy+encodeURIComponent(target.href);
    }else{
      var xhr=new XMLHttpRequest();
      xhr.open(f.method,proxy+encodeURIComponent(abs),true);
      xhr.onload=function(){
        document.open();document.write(xhr.responseText);document.close();
      };
      xhr.send(fd);
    }
  },true);
})();
</script>`

  const baseTag = /<base\s/i.test(html) ? '' : `<base href="${baseUrl}">`
  const injection = baseTag + interceptScript

  const headMatch = html.match(/<head(\s[^>]*)?>/)
  if (headMatch) {
    const insertPos = headMatch.index! + headMatch[0].length
    return html.slice(0, insertPos) + injection + html.slice(insertPos)
  }

  return `${injection}\n${html}`
}

/**
 * Build the base href from a URL.
 *
 * Takes origin + pathname, trimmed to the last `/`.
 * e.g. `https://app.example.com/dashboard/page` -> `https://app.example.com/dashboard/`
 */
export function buildBaseHref(originalUrl: URL): string {
  const pathname = originalUrl.pathname
  const trimmed = pathname.endsWith('/')
    ? pathname
    : pathname.slice(0, pathname.lastIndexOf('/') + 1)

  return `${originalUrl.origin}${trimmed}`
}
