import { expect, devices } from '@playwright/test';
import { windowTest as test, mount, quietCounts, bounds, settle } from './map-window-fixture';
import { buildMapPageFixture } from './map-page-fixture';

for(const dev of [true,false])test.describe(`gesture releases (${dev?'development':'production'})`,()=>{
 let bundle:string;test.beforeAll(async()=>{bundle=await buildMapPageFixture(dev,true,true);});
 for(const {pointer, trigger} of [{pointer:'mouse', trigger:'blur'}, {pointer:'touch', trigger:'hidden'}]){
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
    if(pointer==='mouse')await page.mouse.up();else await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
    if(trigger==='hidden')await page.evaluate(()=>{delete(document as any).hidden;document.dispatchEvent(new Event('visibilitychange'));});
    const after=await quietCounts(page);expect(after).toMatchObject({results:before.results+1,map:before.map+1,publications:before.publications+1});
    const area=await bounds(page),last=await page.evaluate(()=>(window as any).fixture.publications.at(-1)[0]);
    for(const corner of ['northEast','southWest'] as const)for(const axis of ['lat','lng'] as const)expect(last[corner][axis]).toBeCloseTo(area[corner][axis],10);expect(after.url).not.toBe(before.url);await cdp.detach();
   }finally{await context?.close();}
  });
 }
 test('touch drags on split and overview handles survive foreign mouse events', async ({browser}) => {
  const context=await browser.newContext({...devices['Pixel 7'],viewport:{width:1100,height:700}}),page=await context.newPage();
  const cdp=await context.newCDPSession(page);
  try{
   await mount(page,bundle);await page.addStyleTag({content:"output { display:none }"});await settle(page);
   for(const handle of ['list-resize','overlay-resize','overlay-move']){
    if(handle==='overlay-resize'){await page.locator('.expand-results').click();await settle(page);}
    const before=await quietCounts(page),node=page.locator('.'+handle),r=(await node.boundingBox())!,x=r.x+r.width/2,y=r.y+r.height/2;
    await cdp.send('Input.dispatchTouchEvent',{type:'touchStart',touchPoints:[{x,y}]});
    await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[{x:x-25,y:y-20}]});
    await page.waitForTimeout(600);
    expect(await node.evaluate(n=>n.dispatchEvent(new MouseEvent('contextmenu',{bubbles:true,cancelable:true})))).toBe(false);
    await page.evaluate(()=>dispatchEvent(new MouseEvent('mousemove',{buttons:0})));
    const middle=await node.boundingBox();
    await cdp.send('Input.dispatchTouchEvent',{type:'touchMove',touchPoints:[{x:x-100,y:y-70}]});
    await expect.poll(async()=> (await node.boundingBox())?.x).not.toBe(middle?.x);
    await cdp.send('Input.dispatchTouchEvent',{type:'touchEnd',touchPoints:[]});
    const after=await quietCounts(page);
    expect(after.results).toBe(before.results+(handle==='list-resize'?1:0));
    expect(after.publications).toBe(before.publications+(handle==='list-resize'?1:0));
   }
  }finally{await cdp.detach();await context.close();}
 });
});
