import { test, expect } from './fixtures';

test.describe('Bean properties', () => {
  test('detail view lists custom properties sorted by key', async ({
    beans,
    backlogPage,
    page
  }) => {
    const id = beans.create('Properties Bean', { status: 'todo', type: 'task' });
    beans.run(['update', id, '--property', 'zeta=last', '--property', 'github_issue=#135']);

    await backlogPage.goto(1);
    await backlogPage.selectBean('Properties Bean');

    const props = page.getByTestId('bean-properties');
    await expect(props).toBeVisible({ timeout: 10_000 });
    await expect(props.locator('dt')).toHaveText(['github_issue', 'zeta']);
    await expect(props.locator('dd')).toHaveText(['#135', 'last']);
  });

  test('detail view hides the section without properties', async ({ beans, backlogPage, page }) => {
    const id = beans.create('Plain Bean', { status: 'todo', type: 'task' });

    await backlogPage.goto(1);
    await backlogPage.selectBean('Plain Bean');

    await expect(page.getByText(id).first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByTestId('bean-properties')).toHaveCount(0);
  });
});
