# API Layer

> How the frontend talks to the backend gateway.

## Routing

All requests go to same-origin `/api/v1/...`.
- **Development:** Vite proxies them to `INKWORDS_GATEWAY_ORIGIN` (`frontend/vite.config.ts`).
- **Production:** `frontend/nginx.conf` routes them to the owning service.
- `src/gatewayRouting.test.ts` checks that every route prefix is routed.

`src/services/apiRoutes.ts` is the single `as const` route registry. It is grouped by the owning backend service (`coreApi`, `llmStream`, `parserService`, `exportService`, `reviewService`) and wraps path parameters in `encodeURIComponent`. Never hard-code a URL string in a service, hook or component.

When you add an endpoint:
1. Add it under the right service group in `apiRoutes.ts`.
2. If it's a new prefix, update `nginx.conf` and the Vite proxy; `gatewayRouting.test.ts` will fail otherwise.
3. Add a typed function to the domain service module.

## `src/services/apiClient.ts`

| Helper | Use |
|--------|-----|
| `requestEnvelope<T>(url, options)` | Default for JSON APIs. Unwraps `{code, message, error, data}` and requires `code === 200`. |
| `requestJson<T>(url, options)` | Endpoints that don't return the envelope |
| `requestBlob(url, options)` | File downloads |
| `assertApiResponse(response, fallback)` | Status handling shared with SSE (`services/sse.ts`) |

Options:
- `json`: request body, serialized for you and given `Content-Type`.
- `fallbackMessage`: a Chinese message shown when the server sends none.
- `fetchImpl`: lets tests inject a fake `fetch`.
- `token`: overrides the stored token.

Error behaviour:
- **401:** clears the token through `authTokenStore` and throws `AUTH_EXPIRED_MESSAGE` ("登录已过期，请重新登录").
- **502/503/504:** throws `GATEWAY_UNAVAILABLE_MESSAGE`.
- **Other non-OK responses:** throw `new Error(payload.message || payload.error || fallbackMessage)`.

Errors are plain `Error` objects whose message is ready to show. There is no typed error class.

## Service modules

One module per backend domain: `blog.ts`, `review.ts`, `user.ts`, `auth.ts`, `project.ts`, `projectCourse.ts`, `generationTasks.ts`, `exportTasks.ts`, `sidebarExport.ts`. They come in two styles: an object literal (`export const reviewService = { ... }`) or standalone functions (`generationTasks.ts`, `exportTasks.ts`). Follow whichever style the module already uses.

```ts
// src/services/review.ts
export const reviewService = {
  getToday() {
    return requestEnvelope<ReviewCardResponse>(apiRoutes.reviewService.today, {
      fallbackMessage: '请求复习接口失败',
    })
  },
}
```

Rules:
- **Every call passes a Chinese `fallbackMessage`.**
- **Retryable task mutations send an idempotency key.** Pattern: `exportTasks.ts` sends `export-pdf:${blogID}`. An explicit user retry gets a new key.
- **Components don't import services directly.** They go through a hook or store. Importing *types* from services is fine.
- **Show failures to the user** with a toast or an inline `role="alert"`. Don't just `console.error`.

## Known debt (do not copy)

- `services/projectCourse.ts` declares its own `ApiResponse<T>` and calls `requestJson` without checking `code`. New code should use `requestEnvelope`.
- `pages/Editor.tsx`, `pages/Dashboard.tsx`, `pages/Login.tsx` and `components/Sidebar.tsx` import services directly.
- Errors are swallowed with only `console.error` in 18 places, including `blogStore.fetchBlogs` / `updateBlog` and `Dashboard.tsx`.
