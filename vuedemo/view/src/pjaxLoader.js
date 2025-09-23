import { hasView, mountView } from './viewLoader.js'
let win = typeof window !== "undefined" ? window : {}
let doc = win.document || {}
let elementProto = win.Element && win.Element.prototype
let history = win.history
let supported = !!(
    elementProto && history.pushState
)
let origin = location ? (location.protocol + "//" + location.host) : ""
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
        const { _ViEW_, ...props } = e.state.pjaxData;
        loadAndMountComponent(e.state.pjaxUrl, _ViEW_, props);
    }
}
const handleClick = async (e) => {
    e.preventDefault();
    const el = e.currentTarget;
    try {
        // 添加随机参数防止缓存
        const url = new URL(el.href);
        url.searchParams.set('_t', Date.now());
        
        let response = await fetch(url.toString(), {
            method: "GET",
            headers: {
                "Content-Type": "application/json",
                "X-PJAX": "true",
                "X-Requested-With": "XMLHttpRequest",
            },
        });

        if (response.ok) {
            let data = await response.json();
            const { _ViEW_ = undefined, ...props } = data;
            if (!_ViEW_) {
                console.error("No view found");
                window.location.reload();
            } else {
                loadAndMountComponent(el.href, _ViEW_, props);
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
async function loadAndMountComponent(url, view, props) {
    try {
        await mountView(view, props);
    } catch (error) {
        console.error("Error loading component:", error);
        // Fallback to full navigation on failure
        try {
            window.location.href = url;
        } catch (e) {
            console.error('Failed to navigate to', url, e);
        }
    }
}