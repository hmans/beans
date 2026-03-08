import { test, expect } from './fixtures';

test.describe('Backlog sorting', () => {
	test('beans are sorted by status, then priority, then type, then title', async ({
		beans,
		backlogPage
	}) => {
		beans.create('Todo Normal Task', { status: 'todo', priority: 'normal', type: 'task' });
		beans.create('In Progress Bug', { status: 'in-progress', priority: 'normal', type: 'bug' });
		beans.create('Todo High Feature', { status: 'todo', priority: 'high', type: 'feature' });
		beans.create('Draft Idea', { status: 'draft', priority: 'low', type: 'task' });

		await backlogPage.goto(4);

		const titles = await backlogPage.getBeanTitles();
		expect(titles).toEqual([
			'In Progress Bug',
			'Todo High Feature',
			'Todo Normal Task',
			'Draft Idea'
		]);
	});

	test('list re-sorts when a bean priority changes on disk', async ({ beans, backlogPage }) => {
		const id1 = beans.create('Low Priority Task', {
			status: 'todo',
			priority: 'low',
			type: 'task'
		});
		beans.create('Normal Priority Task', { status: 'todo', priority: 'normal', type: 'task' });

		await backlogPage.goto(2);

		let titles = await backlogPage.getBeanTitles();
		expect(titles).toEqual(['Normal Priority Task', 'Low Priority Task']);

		// Change the low-priority bean to critical via CLI (filesystem change)
		beans.update(id1, { priority: 'critical' });

		// The bean should now appear first
		await expect(async () => {
			titles = await backlogPage.getBeanTitles();
			expect(titles).toEqual(['Low Priority Task', 'Normal Priority Task']);
		}).toPass({ timeout: 5_000 });
	});

	test('list re-sorts when a bean status changes on disk', async ({ beans, backlogPage }) => {
		const id1 = beans.create('A Todo Bean', { status: 'todo', type: 'task' });
		beans.create('B In Progress Bean', { status: 'in-progress', type: 'task' });

		await backlogPage.goto(2);

		let titles = await backlogPage.getBeanTitles();
		expect(titles).toEqual(['B In Progress Bean', 'A Todo Bean']);

		// Move the todo bean to in-progress
		beans.update(id1, { status: 'in-progress' });

		// Both are now in-progress, should sort by title
		await expect(async () => {
			titles = await backlogPage.getBeanTitles();
			expect(titles).toEqual(['A Todo Bean', 'B In Progress Bean']);
		}).toPass({ timeout: 5_000 });
	});

	test('new bean appears in correct sorted position', async ({ beans, backlogPage }) => {
		beans.create('Zebra Task', { status: 'todo', type: 'task' });
		beans.create('Alpha Task', { status: 'todo', type: 'task' });

		await backlogPage.goto(2);

		let titles = await backlogPage.getBeanTitles();
		expect(titles).toEqual(['Alpha Task', 'Zebra Task']);

		// Create a new bean that should sort between the two
		beans.create('Middle Task', { status: 'todo', type: 'task' });

		await expect(async () => {
			titles = await backlogPage.getBeanTitles();
			expect(titles).toEqual(['Alpha Task', 'Middle Task', 'Zebra Task']);
		}).toPass({ timeout: 5_000 });
	});

	test('deleted bean disappears from list', async ({ beans, backlogPage }) => {
		const id1 = beans.create('Bean To Delete', { status: 'todo', type: 'task' });
		beans.create('Bean To Keep', { status: 'todo', type: 'task' });

		await backlogPage.goto(2);

		// Delete the bean via CLI
		beans.run(['delete', '--force', id1]);

		await backlogPage.waitForBeanGone('Bean To Delete');
		expect(await backlogPage.count()).toBe(1);
	});
});
