import { mountView } from './viewLoader.js'

if (typeof document !== 'undefined') {
	(async function init() {
		const el = document.getElementById('app') ?? document.body;

		const raw = el?.dataset?.page ?? '{}';
		let page = {};
		if (raw && raw !== '<inertia>data-page</inertia>') {
			try {
				page = JSON.parse(raw) || {};
			} catch (e) {
				console.error('Failed to parse page JSON:', e);
				page = {};
			}
		}

		const { _ViEW_: _ViEW_, ...props } = page;
		const viewName = _ViEW_ ?? 'App';

		try {
			await mountView(viewName, props, el);
		} catch (error) {
			console.error(`Error loading view ${viewName}`, error);
		}
	})();
}

