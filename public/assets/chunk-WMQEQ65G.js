// ../node_modules/svelte/src/version.js
var PUBLIC_VERSION = "5";

// ../node_modules/svelte/src/internal/disclose-version.js
if (typeof window !== "undefined")
  (window.__svelte ||= { v: /* @__PURE__ */ new Set() }).v.add(PUBLIC_VERSION);