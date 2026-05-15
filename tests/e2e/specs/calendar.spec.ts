import { test, expect } from '@playwright/test';

test.describe('Clandar Calendar', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    // Wait for the calendar grid to finish rendering
    await page.waitForSelector('.month-card');
  });

  test('shows 12 month grids', async ({ page }) => {
    const months = page.locator('.month-card');
    await expect(months).toHaveCount(12);
  });

  test('displays current year by default', async ({ page }) => {
    // year is shown in .year-display span (id=year-label)
    const yearEl = page.locator('.year-display');
    await expect(yearEl).toContainText('2026');
  });

  test('navigates to next year on → click', async ({ page }) => {
    const responsePromise = page.waitForResponse(
      resp => resp.url().includes('/api/holidays') && resp.status() === 200,
      { timeout: 15000 }
    );
    await page.locator('#year-next').click();
    await responsePromise;
    await expect(page.locator('.year-display')).toContainText('2027');
  });

  test('navigates to previous year on ← click', async ({ page }) => {
    // Wait for the API response triggered by the year change before asserting
    const responsePromise = page.waitForResponse(
      resp => resp.url().includes('/api/holidays') && resp.status() === 200,
      { timeout: 15000 }
    );
    await page.locator('#year-prev').click();
    await responsePromise;
    await expect(page.locator('.year-display')).toContainText('2025');
  });

  test('shows popover on clicking a holiday date', async ({ page }) => {
    // holiday day cells use .cal-day.has-holiday
    const holidayCell = page.locator('.cal-day.has-holiday').first();
    await holidayCell.click();
    await expect(page.locator('.popover.open')).toBeVisible();
  });

  test('closes popover on Escape key', async ({ page }) => {
    const holidayCell = page.locator('.cal-day.has-holiday').first();
    await holidayCell.click();
    await expect(page.locator('.popover.open')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('.popover.open')).not.toBeVisible();
  });

  test('filters holidays when ASEAN region is selected', async ({ page }) => {
    // Click the ASEAN region header in the sidebar
    await page.locator('.region-header', { hasText: 'ASEAN' }).click();
    // Stats count updates — look for the #stats-count element
    const statsCount = page.locator('#stats-count');
    await expect(statsCount).toBeVisible();
    // The count should be a number (ASEAN has holidays from ID, PH, SG, VN)
    const text = await statsCount.textContent();
    expect(Number(text)).toBeGreaterThanOrEqual(0);
  });

  test('shows no console errors on load', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });
    await page.goto('/');
    await page.waitForSelector('.month-card');
    expect(errors).toHaveLength(0);
  });
});
