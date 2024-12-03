import { mount, unmount } from "svelte";
let win = typeof window !== "undefined" ? window : {}
let doc = win.document || {}
let elementProto = win.Element && win.Element.prototype
let history = win.history
let supported = !!(
    elementProto && history.pushState
)
let origin = location ? (location.protocol + "//" + location.host) : ""
let cacheComponent = {};
if (supported) {
    win.addEventListener("popstate", pjaxState);
}

export function pjaxClick(el) {
    const url = el.href;
    const href = el.getAttribute("href");
    if (supported && url && !href.startsWith('#') && sameWindowOrigin(el.target, url) && !el.__click) {
        el.addEventListener('click', handleClick, true);

        return {
            destroy() {
                el.removeEventListener('click', handleClick, true);
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
const handleClick = async (e) => {
    e.preventDefault();
    const el = e.currentTarget;
    try {
        let response = await fetch(el.href, {
            method: "GET",
            headers: {
                "Content-Type": "application/json",
                "X-PJAX": "true",
            },
        });

        if (response.ok) {
            let data = await response.json();
            const { View = undefined, ...props } = data;
            if (!View) {
                console.error("No view found");
                window.location.reload();
            } else {
                loadAndMountComponent(el.href,View, props);
                const info = {
                    pjaxUrl: el.href,
                    pjaxData: data,
                };
                history.pushState(info, '', el.href);
                const scrollX = (win.scrollX || win.pageXOffset);
                const scrollY = (win.scrollY || win.pageYOffset);
                win.scrollTo(scrollX, scrollY);
            }
            // 在这里处理响应数据
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
    return (
        !target ||
        target === win.name ||
        target === "_self" ||
        (target === "_top" && win === win.top) ||
        (target === "_parent" && win === win.parent)
    ) && (
            url === origin ||
            url.indexOf(origin) === 0
        );
}
async function loadModule(modulePath) {
    if (cacheComponent[modulePath]) {
        return cacheComponent[modulePath];
    }
    const module = await import(modulePath);
    cacheComponent[modulePath] = module;
    return module;
}

async function loadAndMountComponent(url,view, props) {
    try {
        const module = await loadModule(`/assets/${view}.js`);
        document.getElementById("app").innerHTML = "";
        const component = module.default;
        mount(component, {
            target: document.getElementById("app") || document.body,
            props: props,
        });

    } catch (error) {
        win.location.href = url;
    }
}