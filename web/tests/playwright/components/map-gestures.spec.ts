import { expect, devices } from '@playwright/test';
import { windowTest as test, mount, counts, quietCounts, bounds, settle } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';

async function exactPublishedArea(page: import('@playwright/test').Page) {
 const visible=await bounds(page),published=await page.evaluate(()=>(window as any).fixture.publications.at(-1)[0]),params=new URLSearchParams(await page.evaluate(()=>location.search));
 for(const corner of ['northEast','southWest'] as const)for(const axis of ['lat','lng'] as const)expect(published[corner][axis]).toBeCloseTo(visible[corner][axis],10);
 for(const [key,value] of Object.entries({tl_lat:visible.northEast.lat,tl_lon:visible.northEast.lng,br_lat:visible.southWest.lat,br_lon:visible.southWest.lng}))expect(Number(params.get(key))).toBeCloseTo(value,10);
}

for(const dev of [true,false])test.describe(`gesture releases (${dev?'development':'production'})`,()=>{
 let bundle:string;test.beforeAll(async()=>{bundle=await buildMapPageFixture(dev,true,true);});
 for(const pointer of ['mouse','touch'])for(const trigger of ['blur','hidden','released mouse']){
  if(pointer==='touch'&&trigger==='released mouse')continue;
  test(`${pointer} map drag publishes its final area on ${trigger}`,async({browser,page:defaultPage})=>{
   const context=pointer==='touch'?await browser.newContext({...devices['Pixel 7'],viewport:{width:1000,height:600}}):undefined;
   const page=context?await context.newPage():defaultPage;
   try{
    await mount(page,bundle);await page.addStyleTag({content:"output { display:none }"});await settle(page);const before=await quietCounts(page),cdp=await page.context().newCDPSession(page);
    const r=(await page.locator('.maplibregl-canvas').boundingBox())!,x=r.x+r.width/2,y=r.y+r.height/2;
    await page.evaluate(()=>{const m=(window as any).realMap;m.dragPan.disable();m.dragPan.enable({maxSpeed:0});});
    if(pointer==='mouse'){await page.mouse.move(x,y);await page.mouse.down();await page.mouse.move(x+95,y+20,{steps:8});}
    else{await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{x,y}]});for(let i=1;i<=8;i++)await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[{x:x+i*12,y:y+i*2}]});}
    await expect.poll(()=>page.evaluate(()=>(window as any).realMap.isMoving())).toBe(true);
    if(trigger==='blur')await page.evaluate(()=>dispatchEvent(new Event('blur')));
    else if(trigger==='hidden')await page.evaluate(()=>{Object.defineProperty(document,'hidden',{configurable:true,get:()=>true});document.dispatchEvent(new Event('visibilitychange'));});
    else await page.evaluate(()=>dispatchEvent(new MouseEvent('mousemove',{buttons:0})));
    if(pointer==='mouse')await page.mouse.up();else await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
    if(trigger==='hidden')await page.evaluate(()=>{delete(document as any).hidden;document.dispatchEvent(new Event('visibilitychange'));});
    const after=await quietCounts(page);expect(after).toMatchObject({results:before.results+1,map:before.map+1,publications:before.publications+1});
    const area=await bounds(page),last=await page.evaluate(()=>(window as any).fixture.publications.at(-1)[0]);
    for(const corner of ['northEast','southWest'] as const)for(const axis of ['lat','lng'] as const)expect(last[corner][axis]).toBeCloseTo(area[corner][axis],10);expect(after.url).not.toBe(before.url);await cdp.detach();
   }finally{await context?.close();}
  });
 }
 for(const handle of ['list-resize','overlay-resize','overlay-move'])for(const foreign of ['contextmenu','mousemove'])test(`${handle}: touch drag survives a foreign ${foreign}`,async({browser})=>{
  const context=await browser.newContext({...devices['Pixel 7'],viewport:{width:1100,height:700}}),page=await context.newPage();
  try{
   await mount(page,bundle);await page.addStyleTag({content:"output { display:none }"});await settle(page);if(handle!=='list-resize'){await page.locator('.expand-results').click();await settle(page);}
   const before=await quietCounts(page),cdp=await page.context().newCDPSession(page),node=page.locator('.'+handle),r=(await node.boundingBox())!,x=r.x+r.width/2,y=r.y+r.height/2;
   await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{x,y}]});await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[{x:x-25,y:y-20}]});await page.waitForTimeout(600);
   if(foreign==='contextmenu')expect(await node.evaluate(n=>n.dispatchEvent(new MouseEvent('contextmenu',{bubbles:true,cancelable:true})))).toBe(false);
   else await page.evaluate(()=>dispatchEvent(new MouseEvent('mousemove',{buttons:0})));
   const middle=await node.boundingBox();
   await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[{x:x-100,y:y-70}]});
   await expect.poll(async()=> (await node.boundingBox())?.x).not.toBe(middle?.x);
   await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
   const after=await quietCounts(page);expect(after.results).toBe(before.results+(handle==='list-resize'?1:0));expect(after.publications).toBe(before.publications+(handle==='list-resize'?1:0));
   await cdp.detach();
  }finally{await context.close();}
 });
 test('a window resize stops the preceding drag and keeps its final exact bounds',async({page})=>{
  await mount(page,bundle);
  await page.addStyleTag({content:"output { display:none }"});
  await settle(page);
  const before=await quietCounts(page),r=(await page.locator('.maplibregl-canvas').boundingBox())!,x=r.x+r.width*.4,y=r.y+r.height*.5;
  const original=await bounds(page);
  await page.evaluate(()=>{const m=(window as any).realMap;m.dragPan.disable();m.dragPan.enable({maxSpeed:0});});
  await page.clock.install();
  await page.clock.pauseAt(new Date(Date.now()+1000));
  await page.mouse.move(x,y);await page.mouse.down();
  await page.mouse.move(x+20,y+10);await page.clock.runFor(16);
  const session = await page.context().newCDPSession(page);
  const initialWidth = await page.evaluate(()=>innerWidth);
  await page.evaluate(()=>{(window as any).dragResizeEvents=0;addEventListener('resize',()=>++(window as any).dragResizeEvents);});
  for(let i=1;i<=12;i++){
   await page.mouse.move(x+20+i*7,y+10+i*2);
   await page.clock.runFor(16);
   await session.send('Emulation.setDeviceMetricsOverride',{width:initialWidth+i*4,height:800+i*2,deviceScaleFactor:1,mobile:false});
   await expect.poll(()=>page.evaluate(()=>(window as any).dragResizeEvents)).toBe(i);
   await page.clock.runFor(16);
  }
  expect(await page.evaluate(()=>innerWidth)).toBe(initialWidth+48);
  const beforeFinalDrag=await bounds(page);
  // The 200-ms series must actually end before the pointer is released.
  for(let i=1;i<=18;i++){
   await page.mouse.move(x+104+i*3,y+34+i);
   await page.clock.runFor(16);
  }
  expect(await bounds(page)).toEqual(beforeFinalDrag);
  await page.mouse.up();await page.clock.runFor(1000);
  const after=await quietCounts(page);
  expect(after.publications).toBe(before.publications+1);
  expect(after.results).toBe(before.results+1);
  await exactPublishedArea(page);
  expect(after.url).not.toBe(before.url);
  expect(await bounds(page)).not.toEqual(original);
  await session.detach();
 });

 test('two window series during a drag publish only once after release in the second series',async({page})=>{
  await mount(page,bundle);await page.addStyleTag({content:'output { display:none }'});await settle(page);
  const before=await quietCounts(page),r=(await page.locator('.maplibregl-canvas').boundingBox())!,x=r.x+r.width/2,y=r.y+r.height/2;
  await page.evaluate(()=>{const m=(window as any).realMap;m.dragPan.disable();m.dragPan.enable({maxSpeed:0});});
  await page.clock.install();await page.clock.pauseAt(new Date(Date.now()+1000));
  const session=await page.context().newCDPSession(page),initial=await page.evaluate(()=>({width:innerWidth,height:innerHeight}));
  await page.evaluate(()=>{(window as any).seriesEvents=0;addEventListener('resize',()=>++(window as any).seriesEvents);});
  try{
   await page.mouse.move(x,y);await page.mouse.down();await page.mouse.move(x+30,y+10);await page.clock.runFor(16);
   await session.send('Emulation.setDeviceMetricsOverride',{width:initial.width+30,height:initial.height+10,deviceScaleFactor:1,mobile:false});
   await expect.poll(()=>page.evaluate(()=>(window as any).seriesEvents)).toBe(1);
   // R13 stops motion preceding the resize; a newly started drag belongs to the user.
   await page.mouse.up();await page.mouse.move(x,y);await page.mouse.down();await page.mouse.move(x+30,y+10);
   await page.clock.runFor(260);
   expect((await counts(page)).publications).toBe(before.publications);
   await page.mouse.move(x+65,y+20);await page.clock.runFor(16);
   await session.send('Emulation.setDeviceMetricsOverride',{width:initial.width+60,height:initial.height+20,deviceScaleFactor:1,mobile:false});
   await expect.poll(()=>page.evaluate(()=>(window as any).seriesEvents)).toBe(2);await page.clock.runFor(16);
   await page.mouse.move(x+95,y+30);await page.clock.runFor(16);await page.mouse.up();await page.clock.runFor(260);
   // A button-free move after the completed second series must not replay its deferred publication.
   await page.mouse.move(x+105,y+35);await page.clock.runFor(1000);
   const after=await quietCounts(page);expect(after.publications).toBe(before.publications+1);expect(after.results).toBe(before.results+1);
   await exactPublishedArea(page);
  }finally{await session.detach();}
 });
});
