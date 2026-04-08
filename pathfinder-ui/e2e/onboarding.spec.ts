import { test, expect } from '@playwright/test';

test.describe('Onboarding flow', () => {
  test('shows step 1 - goal type selection', async ({ page }) => {
    await page.goto('/onboarding');
    // Step 1 should ask about goal type
    await expect(page.locator('body')).toContainText(/goal|what|primary/i);
    // Should show goal type options
    const typeOptions = page.locator('button, [role="radio"]').filter({ hasText: /career|health|education|personal/i });
    await expect(typeOptions.first()).toBeVisible();
  });

  test('can select a goal type and advance to next step', async ({ page }) => {
    await page.goto('/onboarding');
    // Select a goal type from the type buttons
    await page.locator('button[type="button"]').filter({ hasText: /^career$/i }).click();
    // The Next button is enabled only after selecting a type
    const nextButton = page.getByRole('button', { name: /^next$/i });
    await expect(nextButton).toBeEnabled();
    await nextButton.click();
    // Step 2: goal title input should now be visible
    await expect(page.getByLabel(/Goal Title/i)).toBeVisible();
  });
});
