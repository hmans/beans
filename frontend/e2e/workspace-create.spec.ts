import { test, expect } from './fixtures';

test.describe('Workspace creation', () => {
  test('clicking + creates a workspace and navigates to it', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');

    // Wait for the Workspaces section to appear
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    // Click the "+" button to create a workspace
    await page.getByRole('button', { name: 'Create worktree' }).click();

    // Should navigate to a workspace URL
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    // The sidebar should show the new workspace as active (has font-medium class)
    const sidebar = page.locator('nav');
    const activeWorkspace = sidebar.locator('button.font-medium', {
      has: page.locator('span.truncate')
    });
    await expect(activeWorkspace).toBeVisible({ timeout: 5_000 });

    // The agent chat composer editor should be focused
    const composer = page.locator('.composer-editor[contenteditable="true"]');
    await expect(composer).toBeFocused({ timeout: 5_000 });
  });

  test('new workspace gets an auto-generated name', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    // The active workspace label should match the adjective-animal-suffix pattern
    const sidebar = page.locator('nav');
    const activeLabel = sidebar.locator('button.font-medium span.truncate');
    await expect(activeLabel).toBeVisible({ timeout: 5_000 });
    const name = await activeLabel.textContent();
    expect(name).toMatch(/^[a-z]+-[a-z]+-[a-z0-9]{4}$/);
  });

  test('creating multiple workspaces gives each a unique name', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    // Create first workspace
    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    const sidebar = page.locator('nav');

    // Get first workspace name (skip "main" — it's the first workspace item)
    const workspaceLabels = sidebar.locator('button:has(span.truncate) span.truncate');
    const firstName = await workspaceLabels.nth(1).textContent();

    // Create second workspace
    await page.getByRole('button', { name: 'Create worktree' }).click();

    // Wait for a second workspace to appear
    await expect(workspaceLabels).toHaveCount(3, { timeout: 10_000 }); // main + 2 new

    // Collect all non-main workspace names (order may vary due to LastActiveAt sorting)
    const allNames: string[] = [];
    for (let i = 0; i < 3; i++) {
      const name = await workspaceLabels.nth(i).textContent();
      if (name && name !== 'main') allNames.push(name);
    }

    expect(allNames).toHaveLength(2);
    expect(allNames[0]).not.toBe(allNames[1]);
  });

  test('destroy worktree removes it from sidebar', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    // Create a workspace
    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    // Get the workspace name
    const sidebar = page.locator('nav');
    const activeLabel = sidebar.locator('button.font-medium span.truncate');
    await expect(activeLabel).toBeVisible({ timeout: 5_000 });
    const wsName = await activeLabel.textContent();

    // Hover over the workspace card to reveal the destroy button
    const wsCard = sidebar.locator('div.rounded-md').filter({ hasText: wsName! });
    await wsCard.hover();

    // Click the destroy button (archive icon)
    const destroyButton = wsCard.getByRole('button', { name: 'Destroy worktree' });
    await expect(destroyButton).toBeVisible({ timeout: 2_000 });
    await destroyButton.click();

    // Confirm the destruction in the modal
    const confirmButton = page.getByRole('button', { name: 'Destroy', exact: true });
    await expect(confirmButton).toBeVisible({ timeout: 2_000 });
    await confirmButton.click();

    // Should navigate away from the workspace
    await expect(page).not.toHaveURL(/\/workspace\//, { timeout: 10_000 });

    // The workspace should be gone from the sidebar
    await expect(sidebar.getByText(wsName!)).not.toBeVisible({ timeout: 5_000 });
  });

  test('navigating back to planning after creating workspace works', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    // Create a workspace
    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    // Click Planning to navigate back
    await page.getByRole('button', { name: 'Planning' }).click();
    await expect(page).toHaveURL(/\/planning/, { timeout: 5_000 });

    // The workspace should still be listed in sidebar
    const sidebar = page.locator('nav');
    const workspaceLabels = sidebar.locator('button:has(span.truncate) span.truncate');
    // main + the new workspace = 2
    await expect(workspaceLabels).toHaveCount(2, { timeout: 5_000 });
  });

  test('workspace shows main in sidebar as first item', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });

    // "main" should be the first workspace
    const sidebar = page.locator('nav');
    const firstWorkspace = sidebar.locator('button:has(span.truncate) span.truncate').first();
    await expect(firstWorkspace).toHaveText('main');
  });
});
