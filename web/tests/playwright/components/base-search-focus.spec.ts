import { devices, expect, test as base } from '@playwright/test';
import { buildTrailModalFixture } from './trail-modal-fixture';

const test = base.extend({
 launchOptions: async ({launchOptions}, use) => {
  const ignored = Array.isArray(launchOptions.ignoreDefaultArgs) ? launchOptions.ignoreDefaultArgs : [];
  await use({...launchOptions, ignoreDefaultArgs:[...ignored,'--hide-scrollbars']});
 },
});

const source = `<script>
import Combo from '#lib/components/base/combobox.svelte';
import Search from '#lib/components/base/search.svelte';
import Dropdown from '#lib/components/base/dropdown.svelte';
let icon = $state('');
let tags = $state([{text:'alpha', value:'alpha'}]);
let text = $state('');
let tagQuery = $state('');
let searchItems = $state([]);
const choices = ['car','caravan','car side'].map(text => ({text,value:text}));
let pinned = $state(false);
</script>
<Dropdown items={[{text:'Menu item',value:'item'}]} />
<button data-testid="pin" onclick={() => pinned = !pinned}>Pinned</button>
<section data-testid="icon"><Combo bind:value={icon} items={choices.filter(i=>i.text.includes(icon))} dismissOnEscape={pinned}/></section>
<button data-testid="after-icon">After icon</button>
<section data-testid="search"><Search bind:value={text} items={searchItems} timeBetweenUpdates={0} onupdate={q=>searchItems=q ? choices.filter(i=>i.text.includes(q)) : []} dismissOnEscape={pinned}/></section>
<button data-testid="after-search">After search</button>
<section data-testid="tags"><Combo multiple chips bind:value={tags} items={choices.filter(i=>i.text.includes(tagQuery))} onupdate={q=>tagQuery=q} dismissOnEscape={pinned}/></section>
<input aria-label="Description" />
`;
for (const dev of [true,false]) test.describe(`base search focus (${dev?'development':'production'})`, () => {
 let bundle: string;
 test.beforeAll(async()=>{bundle=await buildTrailModalFixture(dev,source);});
 test.beforeEach(async({page})=>{
  await page.route('http://focus.test/**',r=>r.fulfill({contentType:'text/html',body:'<html><body></body></html>'}));
  await page.goto('http://focus.test/'); await page.addScriptTag({content:bundle});
 });
 for (const kind of ['icon','search']) test(`${kind}: leaving a default field closes the options`,async({page})=>{
  const field=page.getByTestId(kind), input=field.locator('input');
  await input.fill('car'); await expect(field.locator('li').first()).toBeVisible();
  await input.press('Tab'); await expect(field.locator('ul')).toHaveCount(0);
  for(let i=0;i<2 && !(await page.getByTestId(`after-${kind}`).evaluate(n=>n===document.activeElement));i++) await page.keyboard.press('Tab');
  await expect(page.getByTestId(`after-${kind}`)).toBeFocused();
 });
 for(const outside of ['pointer','focus']) test(`pinned chips return focus to the input and dismiss on outside ${outside}`,async({page})=>{
  await page.getByTestId('pin').click();
  const field=page.getByTestId('tags'), input=field.locator('input');
  await input.fill('car'); await expect(field.locator('li').first()).toBeVisible();
  await field.getByRole('button',{name:'Close',exact:true}).click();
  await expect(input).toBeFocused(); await expect(field.locator('ul')).toHaveCount(0);
  await input.click(); await expect(field.locator('li').first()).toBeVisible();
  if(outside==='pointer') await page.getByRole('textbox',{name:'Description'}).click();
  else await page.getByRole('textbox',{name:'Description'}).focus();
  await expect(field.locator('ul')).toHaveCount(0);
 });
 for (const kind of ['icon','search']) test(`${kind}: outside focus closes pinned options after their focused element disappears`, async ({page}) => {
  await page.getByTestId('pin').click();
  const field = page.getByTestId(kind);
  await field.locator('input').fill('car');
  await field.locator('li').first().focus();
  // Firefox does not dispatch focusout when a focused option is removed.
  await field.evaluate(node => {
   node.addEventListener('focusout', event => event.stopImmediatePropagation(), {capture:true});
   document.activeElement?.remove();
  });
  await expect(field.locator('ul')).toBeVisible();
  await page.getByRole('textbox', {name:'Description'}).click();
  await expect(field.locator('ul')).toHaveCount(0);
 });
 test('a secondary click on an option never selects it', async ({page}) => {
  const field = page.getByTestId('icon'), input = field.locator('input');
  await input.fill('car');
  await field.locator('li').nth(1).click({button:'right'});
  await expect(input).toHaveValue('car');
 });
 test('removing a keyboard-selected pinned icon cannot leave stale suggestions', async ({page}) => {
  await page.getByTestId('pin').click();
  const field = page.getByTestId('icon'), input = field.locator('input');
  await input.fill('car');
  await field.locator('li').nth(1).focus();
  // Model Firefox omitting focusout when the keyboard-selected option is removed.
  await field.evaluate(node => node.addEventListener('focusout', event => {
   if ((event.target as Element).closest('li')) event.stopImmediatePropagation();
  }, {capture:true}));
  await page.keyboard.press('Enter');
  await expect(input).toHaveValue('caravan');
  await page.getByRole('textbox', {name:'Description'}).focus();
  await expect(field.locator('ul')).toHaveCount(0);
 });
 for (const method of ['pointer','keyboard']) test(`default chips reopen suggestions after ${method} removal and typing`, async ({page}) => {
  const field = page.getByTestId('tags'), input = field.locator('input');
  await input.fill('car');
  await expect(field.locator('li')).toHaveCount(3);
  const close = field.getByRole('button', {name:'Close',exact:true});
  if (method === 'pointer') {
   await close.click();
   await input.click();
  } else {
   await close.focus();
   await page.keyboard.press('Enter');
   for (let i=0; i<12 && !(await input.evaluate(node=>node===document.activeElement)); i++) await page.keyboard.press('Tab');
   await expect(input).toBeFocused();
  }
  await input.press('End');
  await page.keyboard.type('a');
  await expect(field.locator('li')).toHaveCount(1);
  await expect(field.locator('li')).toHaveText('caravan');
 });
 for (const kind of ['icon','search','tags']) test(`${kind}: a scrollbar pointer press without focus transfer preserves further typing`, async ({page,browserName}) => {
  await page.addStyleTag({content:'html { overflow-y:scroll; scrollbar-gutter:stable } body { min-height:3000px } html::-webkit-scrollbar { width:16px }'});
  const field = page.getByTestId(kind), input = field.locator('input');
  await input.fill('car');
  await expect(field.locator('li')).toHaveCount(3);
  const bar = await page.evaluate(() => ({width:innerWidth-document.documentElement.clientWidth,x:innerWidth-2}));
  expect(await page.evaluate(()=>document.documentElement.scrollHeight)).toBeGreaterThan(await page.evaluate(()=>innerHeight));
  // Headless Firefox hides its scrollbar; Chromium must expose the real gutter.
  if (browserName === 'chromium') expect(bar.width).toBeGreaterThan(0);
  if (browserName === 'chromium') {
   await page.mouse.move(bar.x,20);
   await page.mouse.down();
   await page.mouse.up();
  } else {
   // Headless Firefox has no native gutter. Keep its no-focus-transfer model.
   await page.evaluate(x => document.documentElement.dispatchEvent(new PointerEvent('pointerdown', {
    bubbles:true,button:0,buttons:1,clientX:x,clientY:20,pointerType:'mouse',
   })),bar.x);
  }
  await expect(input).toBeFocused();
  await expect(field.locator('li')).toHaveCount(3);
  await page.keyboard.type('a');
  await expect(field.locator('li')).toHaveCount(1);
  await expect(field.locator('li')).toHaveText('caravan');
 });
 test.describe('Pixel 7 default fields', () => {
  test.use({viewport:devices['Pixel 7'].viewport,deviceScaleFactor:devices['Pixel 7'].deviceScaleFactor,userAgent:devices['Pixel 7'].userAgent,hasTouch:true,isMobile:true});
  test.skip(({browserName})=>browserName!=='chromium','CDP touch is Chromium-only');
  test('default chips reopen suggestions after tap removal and typing', async ({page}) => {
   const field = page.getByTestId('tags'), input = field.locator('input');
   await input.tap();
   await input.fill('car');
   await expect(field.locator('li')).toHaveCount(3);
   await field.getByRole('button', {name:'Close',exact:true}).tap();
   await expect(field.getByRole('button', {name:'Close',exact:true})).toHaveCount(0);
   await input.tap();
   await expect(field.locator('li')).toHaveCount(3);
   await input.press('End');
   await page.keyboard.type('a');
   await expect(field.locator('li')).toHaveCount(1);
   await expect(field.locator('li')).toHaveText('caravan');
  });
  for (const kind of ['icon','search','tags']) test(`${kind}: a touch swipe outside preserves suggestions while typing`, async ({page,browserName}) => {
   test.skip(browserName !== 'chromium', 'CDP touch is Chromium-only');
   await page.addStyleTag({content:'body { min-height:3000px }'});
   const field = page.getByTestId(kind), input = field.locator('input');
   await input.fill('car');
   await expect(field.locator('li')).toHaveCount(3);
   const cdp = await page.context().newCDPSession(page);
   try {
    await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{x:12,y:750}]});
    for (let i=1; i<=6; i++) {
     await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[{x:12,y:750-i*40}]});
     await page.evaluate(() => new Promise(requestAnimationFrame));
    }
    await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
   } finally { await cdp.detach(); }
   await expect.poll(() => page.evaluate(() => scrollY)).toBeGreaterThan(0);
   await expect(input).toBeFocused();
   await expect(field.locator('li')).toHaveCount(3);
   await page.keyboard.type('a');
   await expect(field.locator('li')).toHaveCount(1);
   await expect(field.locator('li')).toHaveText('caravan');
  });
 });
});
