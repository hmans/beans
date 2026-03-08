import { test, expect } from './fixtures';

test.describe('Board sorting', () => {
	test('beans appear in the correct column by status', async ({ beans, boardPage }) => {
		beans.create('Draft Bean', { status: 'draft', type: 'task' });
		beans.create('Todo Bean', { status: 'todo', type: 'task' });
		beans.create('Active Bean', { status: 'in-progress', type: 'task' });
		beans.create('Done Bean', { status: 'completed', type: 'task' });

		await boardPage.goto();

		// Wait for all beans to appear in their columns
		await boardPage.waitForBeanInColumn('Draft Bean', 'draft');
		await boardPage.waitForBeanInColumn('Todo Bean', 'todo');
		await boardPage.waitForBeanInColumn('Active Bean', 'in-progress');
		await boardPage.waitForBeanInColumn('Done Bean', 'completed');

		expect(await boardPage.getColumnTitles('draft')).toEqual(['Draft Bean']);
		expect(await boardPage.getColumnTitles('todo')).toEqual(['Todo Bean']);
		expect(await boardPage.getColumnTitles('in-progress')).toEqual(['Active Bean']);
		expect(await boardPage.getColumnTitles('completed')).toEqual(['Done Bean']);
	});

	test('beans within a column are sorted by priority', async ({ beans, boardPage }) => {
		beans.create('Low Task', { status: 'todo', priority: 'low', type: 'task' });
		beans.create('Critical Task', { status: 'todo', priority: 'critical', type: 'task' });
		beans.create('Normal Task', { status: 'todo', priority: 'normal', type: 'task' });
		beans.create('High Task', { status: 'todo', priority: 'high', type: 'task' });

		await boardPage.goto();
		await boardPage.waitForColumnCount('todo', 4);

		const titles = await boardPage.getColumnTitles('todo');
		expect(titles).toEqual(['Critical Task', 'High Task', 'Normal Task', 'Low Task']);
	});

	test('bean moves to new column when status changes on disk', async ({ beans, boardPage }) => {
		const id = beans.create('Moving Bean', { status: 'todo', type: 'task' });

		await boardPage.goto();
		await boardPage.waitForBeanInColumn('Moving Bean', 'todo');

		// Change status via CLI
		beans.update(id, { status: 'in-progress' });

		// Wait for it to appear in the new column
		await boardPage.waitForBeanInColumn('Moving Bean', 'in-progress');
		await boardPage.waitForBeanNotInColumn('Moving Bean', 'todo');
	});

	test('column re-sorts when bean priority changes on disk', async ({ beans, boardPage }) => {
		const id = beans.create('Will Be Critical', {
			status: 'todo',
			priority: 'low',
			type: 'task'
		});
		beans.create('Normal Priority', { status: 'todo', priority: 'normal', type: 'task' });

		await boardPage.goto();
		await boardPage.waitForColumnCount('todo', 2);

		// Initially: Normal sorts before Low
		let titles = await boardPage.getColumnTitles('todo');
		expect(titles).toEqual(['Normal Priority', 'Will Be Critical']);

		// Promote to critical
		beans.update(id, { priority: 'critical' });

		// Should re-sort: critical before normal
		await expect(async () => {
			titles = await boardPage.getColumnTitles('todo');
			expect(titles).toEqual(['Will Be Critical', 'Normal Priority']);
		}).toPass({ timeout: 5_000 });
	});
});
