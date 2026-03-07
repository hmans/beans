import type { Bean } from '$lib/beans.svelte';
import { beansStore } from '$lib/beans.svelte';

class UIState {
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

	/** Read initial selection from URL on page load */
	loadFromUrl() {
		const params = new URLSearchParams(window.location.search);
		const beanId = params.get('bean');
		if (beanId) {
			this.selectedBeanId = beanId;
		}
	}

	// Draggable pane
	paneWidth = $state(350);
	isDragging = $state(false);

	startDrag(e: MouseEvent) {
		this.isDragging = true;
		e.preventDefault();
	}

	onDrag(e: MouseEvent) {
		if (!this.isDragging) return;
		this.paneWidth = Math.max(200, Math.min(600, e.clientX));
	}

	stopDrag() {
		if (this.isDragging) {
			this.isDragging = false;
			localStorage.setItem('beans-pane-width', this.paneWidth.toString());
		}
	}

	loadPaneWidth() {
		const saved = localStorage.getItem('beans-pane-width');
		if (saved) {
			this.paneWidth = Math.max(200, Math.min(600, parseInt(saved, 10)));
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
