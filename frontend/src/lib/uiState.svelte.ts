import { goto } from '$app/navigation';
import type { Bean } from '$lib/beans.svelte';
import { beansStore } from '$lib/beans.svelte';

class UIState {
  // Active view: 'planning' or a beanId for workspace view (derived from URL)
  activeView = $state<'planning' | string>('planning');

  // Planning sub-view (derived from URL)
  planningView = $state<'backlog' | 'board'>('backlog');

  get isPlanning(): boolean {
    return this.activeView === 'planning';
  }

  /** Sync UIState from URL path. Called reactively from layout on every navigation. */
  syncFromUrl(pathname: string) {
    const workspaceMatch = pathname.match(/^\/workspace\/([^/]+)/);
    if (workspaceMatch) {
      this.activeView = workspaceMatch[1];
      return;
    }

    this.activeView = 'planning';
    this.planningView = pathname === '/planning/board' ? 'board' : 'backlog';
  }

  /** Navigate to a view via URL routing. */
  navigateTo(view: 'planning' | string) {
    if (view === 'planning') {
      goto(`/planning${this.planningView === 'board' ? '/board' : ''}`);
    } else {
      goto(`/workspace/${view}`);
    }
  }

  /** Navigate to a planning sub-view. */
  navigateToPlanningView(view: 'backlog' | 'board') {
    goto(view === 'board' ? '/planning/board' : '/planning');
  }

  // Selected bean ID (source of truth)
  selectedBeanId = $state<string | null>(null);

  // Resolved bean from store
  get currentBean(): Bean | null {
    return this.selectedBeanId ? (beansStore.get(this.selectedBeanId) ?? null) : null;
  }

  selectBean(bean: Bean) {
    this.selectedBeanId = bean.id;
    this.syncSelectedBeanToUrl();
  }

  clearSelection() {
    this.selectedBeanId = null;
    this.syncSelectedBeanToUrl();
  }

  /** Update the URL query param without navigation */
  private syncSelectedBeanToUrl() {
    const url = new URL(window.location.href);
    if (this.selectedBeanId) {
      url.searchParams.set('bean', this.selectedBeanId);
    } else {
      url.searchParams.delete('bean');
    }
    window.history.replaceState(window.history.state, '', url);
  }

  // Planning chat pane (persisted to localStorage)
  showPlanningChat = $state(false);

  togglePlanningChat() {
    this.showPlanningChat = !this.showPlanningChat;
    localStorage.setItem('beans-planning-chat', this.showPlanningChat ? 'true' : 'false');
  }

  // Changes pane (persisted to localStorage)
  showChanges = $state(false);

  toggleChanges() {
    this.showChanges = !this.showChanges;
    localStorage.setItem('beans-changes-pane', this.showChanges ? 'true' : 'false');
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
