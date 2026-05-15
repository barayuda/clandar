import { test, expect } from '@playwright/test';

test.describe('Clandar API', () => {
  test('GET /health returns ok', async ({ request }) => {
    const res = await request.get('/health');
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.status).toBe('ok');
  });

  test('GET /api/regions returns 4 regions', async ({ request }) => {
    const res = await request.get('/api/regions');
    expect(res.ok()).toBeTruthy();
    const body = await res.json();
    expect(body.regions).toHaveLength(4);
  });

  test('GET /api/countries returns 20 countries', async ({ request }) => {
    const res = await request.get('/api/countries');
    const body = await res.json();
    expect(body.count).toBe(20);
  });

  test('GET /api/holidays?country=AU&year=2026 returns holidays', async ({ request }) => {
    const res = await request.get('/api/holidays?country=AU&year=2026');
    const body = await res.json();
    expect(body.count).toBeGreaterThan(0);
    expect(body.holidays[0].country_code).toBe('AU');
  });

  test('GET /api/holidays with invalid year returns 400', async ({ request }) => {
    const res = await request.get('/api/holidays?year=1800');
    expect(res.status()).toBe(400);
    const body = await res.json();
    expect(body.error).toContain('2000');
  });

  test('GET /api/holidays with invalid type returns 400', async ({ request }) => {
    const res = await request.get('/api/holidays?type=garbage');
    expect(res.status()).toBe(400);
  });
});
