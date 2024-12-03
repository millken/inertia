import {
  NAMESPACE_SVG,
  action,
  append,
  comment,
  element,
  first_child,
  mount,
  noop,
  pop,
  prop,
  push,
  rest_props,
  set_attributes,
  snippet,
  template_effect
} from "/assets/chunk-2IRIOZ5M.js";

// ../view/Post/pjax.action.js
var win = typeof window !== "undefined" ? window : {};
var doc = win.document || {};
var elementProto = win.Element && win.Element.prototype;
var history = win.history;
var supported = !!(elementProto && history.pushState);
var origin = location ? location.protocol + "//" + location.host : "";
var cacheComponent = {};
if (supported) {
  win.addEventListener("popstate", pjaxState);
}
function pjaxClick(el) {
  const url = el.href;
  const href = el.getAttribute("href");
  if (supported && url && !href.startsWith("#") && sameWindowOrigin(el.target, url) && !el.__click) {
    el.addEventListener("click", handleClick, true);
    return {
      destroy() {
        el.removeEventListener("click", handleClick, true);
      }
    };
  }
}
function pjaxState(e) {
  if (e.state && e.state.pjaxUrl) {
    const { View, ...props } = e.state.pjaxData;
    loadAndMountComponent(e.state.pjaxUrl, View, props);
  }
}
var handleClick = async (e) => {
  e.preventDefault();
  const el = e.currentTarget;
  try {
    let response = await fetch(el.href, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        "X-PJAX": "true"
      }
    });
    if (response.ok) {
      let data = await response.json();
      const { View = void 0, ...props } = data;
      if (!View) {
        console.error("No view found");
        window.location.reload();
      } else {
        loadAndMountComponent(el.href, View, props);
        const info = {
          pjaxUrl: el.href,
          pjaxData: data
        };
        history.pushState(info, "", el.href);
        const scrollX = win.scrollX || win.pageXOffset;
        const scrollY = win.scrollY || win.pageYOffset;
        win.scrollTo(scrollX, scrollY);
      }
      if (data.redirect) {
        window.location.href = data.redirect;
      }
    } else if (response.redirected) {
      window.location.href = el.href;
    }
  } catch (error) {
    window.location.href = el.href;
  }
};
function sameWindowOrigin(target, url) {
  target = target.toLowerCase();
  return (!target || target === win.name || target === "_self" || target === "_top" && win === win.top || target === "_parent" && win === win.parent) && (url === origin || url.indexOf(origin) === 0);
}
async function loadModule(modulePath) {
  if (cacheComponent[modulePath]) {
    return cacheComponent[modulePath];
  }
  const module = await import(modulePath);
  cacheComponent[modulePath] = module;
  return module;
}
async function loadAndMountComponent(url, view, props) {
  try {
    const module = await loadModule(`/assets/${view}.js`);
    document.getElementById("app").innerHTML = "";
    const component = module.default;
    mount(component, {
      target: document.getElementById("app") || document.body,
      props
    });
  } catch (error) {
    win.location.href = url;
  }
}

// ../view/Post/Link.svelte
function Link($$anchor, $$props) {
  push($$props, true);
  let tag = prop($$props, "tag", 3, "a"), rest = rest_props($$props, [
    "$$slots",
    "$$events",
    "$$legacy",
    "tag",
    "children"
  ]);
  var fragment = comment();
  var node = first_child(fragment);
  element(node, tag, false, ($$element, $$anchor2) => {
    action($$element, ($$node) => pjaxClick($$node));
    let attributes;
    template_effect(() => attributes = set_attributes($$element, attributes, { ...rest }, void 0, $$element.namespaceURI === NAMESPACE_SVG, $$element.nodeName.includes("-")));
    var fragment_1 = comment();
    var node_1 = first_child(fragment_1);
    snippet(node_1, () => $$props.children ?? noop);
    append($$anchor2, fragment_1);
  });
  append($$anchor, fragment);
  pop();
}

export {
  Link
};
