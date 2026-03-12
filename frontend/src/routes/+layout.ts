import { browser } from '$app/environment';

export const prerender = true;
export const ssr = false;

export function load() {
  let selectedBeanId: string | null = null;
  let showPlanningChat = false;
  let showChanges = false;
  let filterText = '';

  if (browser) {
    const params = new URLSearchParams(window.location.search);
    selectedBeanId = params.get('bean');

    showPlanningChat = localStorage.getItem('beans-planning-chat') === 'true';
    showChanges = localStorage.getItem('beans-changes-pane') === 'true';
    filterText = localStorage.getItem('beans-filter-text') ?? '';
  }

  return { selectedBeanId, showPlanningChat, showChanges, filterText };
}
