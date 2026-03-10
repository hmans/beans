import type { Bean } from '$lib/beans.svelte';
import { beansStore } from '$lib/beans.svelte';

class UIState {
	// Active view: 'planning' or a beanId for workspace view
	activeView = $state<'planning' | string>('planning');

	get isPlanning(): boolean {
		return this.activeView === 'planning';
	}

	setActiveView(view: 'planning' | string) {
		this.activeView = view;
		localStorage.setItem('beans-active-view', view);
	}

	// Selected bean ID (source of truth)
	selectedBeanId = $state<string | null>(null);

	// Resolved bean from store
	get currentBean(): Bean | null {
		return this.selectedBeanId ? beansStore.get(this.selectedBeanId) ?? null : null;
	}

	selectBean(bean: Bean) {
		this.selectedBeanId = bean.id;
		this.syncToUrl();
	}

	clearSelection() {
		this.selectedBeanId = null;
		this.syncToUrl();
	}

	/** Update the URL query param without navigation */
	private syncToUrl() {
		const url = new URL(window.location.href);
		if (this.selectedBeanId) {
			url.searchParams.set('bean', this.selectedBeanId);
		} else {
			url.searchParams.delete('bean');
		}
		window.history.replaceState(window.history.state, '', url);
	}

	// Planning view toggle (persisted to localStorage, initialized from layout load)
	planningView = $state<'backlog' | 'board'>('backlog');

	setPlanningView(view: 'backlog' | 'board') {
		this.planningView = view;
		localStorage.setItem('beans-planning-view', view);
	}

	// Planning chat pane (persisted to localStorage)
	showPlanningChat = $state(false);

	togglePlanningChat() {
		this.showPlanningChat = !this.showPlanningChat;
		localStorage.setItem('beans-planning-chat', this.showPlanningChat ? 'true' : 'false');
	}

	// Filter text (persisted to localStorage)
	filterText = $state('');

	setFilterText(text: string) {
		this.filterText = text;
		if (text) {
			localStorage.setItem('beans-filter-text', text);
		} else {
			localStorage.removeItem('beans-filter-text');
		}
	}

	// Form modal
	showForm = $state(false);
	editingBean = $state<Bean | null>(null);

	openCreateForm() {
		this.editingBean = null;
		this.showForm = true;
	}

	openEditForm(bean: Bean) {
		this.editingBean = bean;
		this.showForm = true;
	}

	closeForm() {
		this.showForm = false;
		this.editingBean = null;
	}
}

export const ui = new UIState();
