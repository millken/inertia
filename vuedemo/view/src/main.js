import { createApp } from 'vue'
import {  hasView, loadView } from './viewLoader.js'

if (typeof document !== 'undefined') {
	(async function init() {
		const el = document.getElementById('app') ?? document.body;

		const raw = el?.dataset?.page ?? '{}';
		let page = {};
		if (raw && raw !== '<%data-page%>') {
			try {
				page = JSON.parse(raw) || {};
			} catch (e) {
				console.error('Failed to parse page JSON:', e);
				page = {};
			}
		}

		const { _ViEW_: _ViEW_, ...props } = page;
		const viewName = _ViEW_ ?? 'App';

		if (hasView(viewName)) {
			try {
				const mod = await loadView(viewName);
				const component = mod?.default ?? mod;
				if (!component) throw new Error('Loaded view has no component');

				createApp(component, props).mount(el);
			} catch (error) {
				console.error(`Error loading view ${viewName}`, error);
			}
		} else {
			console.error(`View ${viewName} not found`);
		}
	})();
}

