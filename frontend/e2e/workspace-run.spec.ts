import { test, expect } from './fixtures';

test.describe('Workspace run experience', () => {
  test('Run button appears when run command is configured', async ({ beansWithRun, page }) => {
    await page.goto(beansWithRun.baseURL + '/');

    // Create a workspace
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });
    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    // Run button should be visible (identified by title "Run: <command>")
    await expect(page.getByTitle(/^Run:/)).toBeVisible({ timeout: 5_000 });
  });

  test('Run button is hidden when no run command is configured', async ({ beans, page }) => {
    await page.goto(beans.baseURL + '/');

    // Create a workspace
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });
    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    // Wait for workspace to load
    await expect(page.getByRole('button', { name: 'VS Code' })).toBeVisible({ timeout: 5_000 });

    // Run button should NOT be visible
    await expect(page.getByTitle(/^Run:/)).not.toBeVisible();
  });

  test('clicking Run starts the process and shows Stop button', async ({ beansWithRun, page }) => {
    await page.goto(beansWithRun.baseURL + '/');

    // Create a workspace
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });
    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    // Click the Run button
    const runButton = page.getByTitle(/^Run:/);
    await expect(runButton).toBeVisible({ timeout: 5_000 });
    await runButton.click();

    // Stop button should appear
    const stopButton = page.getByRole('button', { name: 'Stop' });
    await expect(stopButton).toBeVisible({ timeout: 5_000 });

    // Toolbar Run button should be gone (replaced by Stop)
    await expect(runButton).not.toBeVisible();

    // Run pane should be visible (only Run, not Terminal)
    await expect(page.getByRole('textbox', { name: 'Terminal input' })).toHaveCount(1, { timeout: 5_000 });
  });

  test('clicking Stop kills the process and reverts to Run button', async ({
    beansWithRun,
    page
  }) => {
    await page.goto(beansWithRun.baseURL + '/');

    // Create a workspace and start run
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });
    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    const runButton = page.getByTitle(/^Run:/);
    await expect(runButton).toBeVisible({ timeout: 5_000 });
    await runButton.click();

    // Wait for Stop button
    const stopButton = page.getByRole('button', { name: 'Stop' });
    await expect(stopButton).toBeVisible({ timeout: 5_000 });

    // Click Stop
    await stopButton.click();

    // Run button should reappear
    await expect(page.getByTitle(/^Run:/)).toBeVisible({ timeout: 5_000 });

    // Stop button should be gone
    await expect(stopButton).not.toBeVisible();
  });

  test('Open button appears when running and has correct port', async ({
    beansWithRun,
    page
  }) => {
    await page.goto(beansWithRun.baseURL + '/');

    // Create a workspace and start run
    await expect(page.getByText('Workspaces')).toBeVisible({ timeout: 10_000 });
    await page.getByRole('button', { name: 'Create worktree' }).click();
    await expect(page).toHaveURL(/\/workspace\//, { timeout: 10_000 });

    const runButton = page.getByTitle(/^Run:/);
    await expect(runButton).toBeVisible({ timeout: 5_000 });
    await runButton.click();

    // Open button should appear with a port URL
    const openButton = page.getByRole('button', { name: 'Open' });
    await expect(openButton).toBeVisible({ timeout: 5_000 });

    // Check it has a title with localhost and a port
    const title = await openButton.getAttribute('title');
    expect(title).toMatch(/http:\/\/localhost:\d+/);
  });
});
