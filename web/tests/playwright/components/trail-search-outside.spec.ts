import {expect, test} from '@playwright/test';
import {buildTrailSearchFixture,mountTrailSearchFixture} from './trail-search-fixture';
for(const dev of [true,false])test.describe(`push panel outside gestures (${dev?'development':'production'})`,()=>{
 let bundle:string;test.beforeAll(async()=>{bundle=await buildTrailSearchFixture(dev);});
 test('a drag and real resize retain the panel, while a tap closes it',async({page})=>{
  await mountTrailSearchFixture(page,bundle,{layout:'push'});
  await page.getByRole('searchbox',{name:'Search trails',exact:true}).focus();
  const panel=page.locator('.search-panel');await expect(panel).toBeVisible();
  const button=page.getByTestId('outside'),r=(await button.boundingBox())!;
  await page.mouse.move(r.x+5,r.y+5);await page.mouse.down();await page.mouse.move(r.x+65,r.y+10);await page.mouse.up();
  await expect(panel).toBeVisible();
  await page.setViewportSize({width:1100,height:850});await expect(panel).toBeVisible();
  await button.click();await expect(panel).toHaveCount(0);
 });
 test('a secondary press never leaves an outside gesture pending',async({page})=>{
  await mountTrailSearchFixture(page,bundle,{layout:'push'});
  await page.getByRole('searchbox',{name:'Search trails',exact:true}).focus();
  await expect(page.locator('.search-panel')).toBeVisible();
  const node = (await page.getByTestId('outside').boundingBox())!;
  // Deliberately omit mouseup: native context menus can consume the release.
  await page.mouse.move(node.x+5,node.y+5);
  await page.mouse.down({button:'right'});
  await expect(page.locator('.search-panel')).toHaveCount(0);
 });
});
