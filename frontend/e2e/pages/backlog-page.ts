import { expect, type Locator, type Page } from '@playwright/test';

/**
 * Page object for the backlog (list) view at /.
 */
export class BacklogPage {
	readonly beanItems: Locator;

	constructor(private page: Page) {
		this.beanItems = page.locator('.bean-item');
	}

	/**
	 * Navigate to the backlog page and wait for beans to load.
	 * @param expectedCount If provided, wait until exactly this many beans are visible.
	 */
	async goto(expectedCount?: number) {
		await this.page.goto('/');
		if (expectedCount !== undefined && expectedCount > 0) {
			await expect(this.beanItems).toHaveCount(expectedCount, { timeout: 10_000 });
		} else if (expectedCount === undefined) {
			// Just wait for at least one bean
			await this.page.waitForSelector('.bean-item', { timeout: 10_000 });
		}
		// If expectedCount === 0, no waiting needed
	}

	/** Get all visible bean titles in display order. */
	async getBeanTitles(): Promise<string[]> {
		const titles = await this.beanItems.locator('button > div > span.text-sm').allTextContents();
		return titles.map((t) => t.trim());
	}

	/** Get all visible bean statuses in display order. */
	async getBeanStatuses(): Promise<string[]> {
		const statuses = await this.beanItems
			.locator('button > div > span.rounded-full')
			.allTextContents();
		return statuses.map((s) => s.trim());
	}

	/** Click on a bean by its title. */
	async selectBean(title: string) {
		await this.beanItems.filter({ hasText: title }).first().locator('button').click();
	}

	/** Wait for a specific bean to appear. */
	async waitForBean(title: string) {
		await this.page.locator('.bean-item', { hasText: title }).waitFor({ timeout: 10_000 });
	}

	/** Wait for a bean to disappear from the list. */
	async waitForBeanGone(title: string) {
		await this.page
			.locator('.bean-item', { hasText: title })
			.waitFor({ state: 'detached', timeout: 10_000 });
	}

	/** Get the count of visible beans. */
	async count(): Promise<number> {
		return this.beanItems.count();
	}
}
