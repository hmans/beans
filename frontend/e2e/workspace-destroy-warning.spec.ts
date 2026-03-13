import { test, expect } from './fixtures';

/**
 * Helper to create a workspace and return its name.
 * Waits for the workspace to be fully set up before returning.
 */
async function createWorkspace(page: import('@playwright/test').Page) {
  const sidebar = page.locator('nav');
  await page.getByRole('button', { name: 'Create worktree' }).click();
  await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

  const activeLabel = sidebar.locator('button.font-medium span.truncate');
  await expect(activeLabel).toBeVisible({ timeout: 5_000 });
  return (await activeLabel.textContent())!;
}

/**
 * Helper to open the destroy confirmation modal for a workspace.
 */
async function openDestroyModal(page: import('@playwright/test').Page, wsName: string) {
  const sidebar = page.locator('nav');
  const wsButton = sidebar.locator('button.font-medium').filter({ hasText: wsName });
  await wsButton.hover();

  const destroyButton = wsButton.getByRole('button', { name: 'Destroy worktree' });
  await expect(destroyButton).toBeVisible({ timeout: 2_000 });
  await destroyButton.click();
}

test.describe('Workspace destroy warnings', () => {
  test('shows no warning for clean workspace', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    const wsName = await createWorkspace(page);
    await openDestroyModal(page, wsName);

    // Modal should show the standard message without warnings
    const modal = page.locator('.fixed');
    await expect(modal.locator('p')).toContainText('Are you sure you want to destroy');
    await expect(modal.locator('p')).not.toContainText('uncommitted changes');
    await expect(modal.locator('p')).not.toContainText('unmerged commits');

    // Cancel so we don't destroy it
    await page.getByRole('button', { name: 'Cancel' }).click();
  });

  test('warns about uncommitted changes', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    const wsName = await createWorkspace(page);

    // Find the worktree path and create an uncommitted file
    const worktrees = await beans.getWorktrees();
    const wt = worktrees.find((w) => w.branch.includes(wsName));
    expect(wt).toBeTruthy();
    beans.createFileInWorktree(wt!.path, 'dirty-file.txt', 'uncommitted content');

    await openDestroyModal(page, wsName);

    // Modal should warn about uncommitted changes
    const modal = page.locator('.fixed');
    await expect(modal.locator('p')).toContainText('uncommitted changes');

    await page.getByRole('button', { name: 'Cancel' }).click();
  });

  test('warns about unmerged commits', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    const wsName = await createWorkspace(page);

    // Find the worktree path and create a commit
    const worktrees = await beans.getWorktrees();
    const wt = worktrees.find((w) => w.branch.includes(wsName));
    expect(wt).toBeTruthy();
    beans.commitInWorktree(wt!.path, 'committed-file.txt', 'committed content');

    await openDestroyModal(page, wsName);

    // Modal should warn about unmerged commits
    const modal = page.locator('.fixed');
    await expect(modal.locator('p')).toContainText('unmerged commits');

    await page.getByRole('button', { name: 'Cancel' }).click();
  });

  test('warns about both uncommitted changes and unmerged commits', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    const wsName = await createWorkspace(page);

    // Find the worktree path
    const worktrees = await beans.getWorktrees();
    const wt = worktrees.find((w) => w.branch.includes(wsName));
    expect(wt).toBeTruthy();

    // Create both a commit and an uncommitted file
    beans.commitInWorktree(wt!.path, 'committed-file.txt', 'committed content');
    beans.createFileInWorktree(wt!.path, 'dirty-file.txt', 'uncommitted content');

    // Wait for the subscription to pick up the changes
    await page.waitForTimeout(1_000);

    await openDestroyModal(page, wsName);

    // Modal should warn about both
    const modal = page.locator('.fixed');
    await expect(modal.locator('p')).toContainText('uncommitted changes');
    await expect(modal.locator('p')).toContainText('unmerged commits');

    await page.getByRole('button', { name: 'Cancel' }).click();
  });
});
