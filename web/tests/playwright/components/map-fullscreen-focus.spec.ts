import { discardHiddenScrollPosition } from './map-scroll-fixture';
import { expect, test } from '@playwright/test';
import { buildMapPageFixture, mountMapPageFixture } from './map-page-fixture';
import { setBrowserFullscreen } from '../browser_fullscreen';
for(const dev of [true,false])test.describe(`fullscreen list restoration (${dev?'development':'production'})`,()=>{
 let bundle:string;test.beforeAll(async()=>{bundle=await buildMapPageFixture(dev,true);});
 for(const focus of ['result','body'])test(`restores scrolled results after a fullscreen cycle with ${focus} focus`,async({page})=>{
  await mountMapPageFixture(page,bundle);
  await page.evaluate(()=>{const items=Array.from({length:35},(_,i)=>({id:'fixture-'+i,author:'tester',name:'Trail '+i}));for(const r of(window as any).fixture.requests)r.resolve(r.channel==='map'?{mapTrails:[],clusters:{type:'FeatureCollection',features:[]}}:{items,page:1,totalPages:1,totalHits:items.length});});
  await page.addStyleTag({content:'.trail-result { min-height:180px }'});
  const list=page.locator('.trail-results'),result=page.locator('.trail-result a').nth(25);await result.focus();
  if(focus==='body')await result.evaluate(n=>(n as HTMLElement).blur());
  const before=await list.evaluate(n=>n.scrollTop);
  expect(before).toBeGreaterThan(500);
  await discardHiddenScrollPosition(page);
  await setBrowserFullscreen(page,true);
  await setBrowserFullscreen(page,false);
  await expect.poll(()=>list.evaluate(n=>n.scrollTop)).toBeCloseTo(before,0);
  if(focus==='result')await expect(result).toBeFocused();
  else {
   // Start another session from body after scrolling away from the previous trigger.
   await result.focus();
   await setBrowserFullscreen(page,true);await setBrowserFullscreen(page,false);
   await result.evaluate(node=>(node as HTMLElement).blur());
   await list.evaluate(node=>node.scrollTop=800);
   await expect.poll(()=>list.evaluate(node=>node.scrollTop)).toBe(800);
   await setBrowserFullscreen(page,true);await setBrowserFullscreen(page,false);
   await expect.poll(()=>list.evaluate(node=>node.scrollTop)).toBe(800);
   await expect(page.getByTestId('fixture-search')).toBeFocused();
  }
 });
});
