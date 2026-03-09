import { browser } from '$app/environment';

export const prerender = true;
export const ssr = false;

export function load() {
	let planningView: 'backlog' | 'board' = 'backlog';
	let selectedBeanId: string | null = null;

	if (browser) {
		const saved = localStorage.getItem('beans-planning-view');
		if (saved === 'backlog' || saved === 'board') {
			planningView = saved;
		}

		const params = new URLSearchParams(window.location.search);
		selectedBeanId = params.get('bean');
	}

	return { planningView, selectedBeanId };
}
