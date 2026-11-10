// Base URL of the controller API.
//
// The controller serves the built frontend, so the API is on the page's own
// origin. That also works when the controller sits behind an HTTPS reverse
// proxy on 443. `yarn dev` runs Vite on another port, so in development the
// API is a controller on :8888 of the same host. VITE_API_BASE_URL overrides
// both.
export const apiBasePath: string = import.meta.env.VITE_API_BASE_URL
  || (import.meta.env.DEV
    ? `${window.location.protocol}//${window.location.hostname}:8888`
    : window.location.origin)
