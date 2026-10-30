import { expect, type Page } from '@playwright/test';
import { test, routes, installRecorder, tilesSettled, windowBounds, closeTileServer } from './map-raster-fixture';
import { buildMapPageFixture } from './map-page-fixture';
import { quietCounts } from './map-window-fixture';

const pois = { food: { 'food-drinks': false, 'grocery-store': false, bakery: false },
    tourism: { campsite: false, attraction: false, hotel: false, hut: true, viewpoint: false },
    amenity: { water: false, barrier: false, shelter: false, shower: false, toilets: false },
    hiking: { 'mountain-pass': false, climbing: false, summit: false },
    cycling: { 'bicycle-rental': false, 'bicycle-shop': false, 'bicycle-parking': false },
    'car-motorcycle': { garage: false, 'gas-station': false, parking: false },
    'public-transport': { 'railway-station': false, 'subway-stop': false, 'tram-stop': false, 'bus-stop': false, ferry: false } };
const mapState = { base: 'OpenFreeMap', overlays: { hiking: false, cycling: false, MTB: false, skiing: false }, pois };

async function mount(page: Page, bundle: string) {
 const origin=await routes(page, path=>path==='/api/v1/overpass/interpreter'?{json:{elements:[]}}:undefined);
 await page.goto(origin+'/map?tl_lat=47.1&tl_lon=8.1&br_lat=46.9&br_lon=7.9');
 await expect.poll(()=>page.evaluate(()=>Boolean((window as any).RealMapLibre))).toBe(true);
 await page.addScriptTag({content:bundle});
 await expect.poll(()=>page.evaluate(()=>(window as any).fixture.publications.length)).toBeGreaterThan(0);
 await page.evaluate(()=>{const w=window as any; for(const request of w.fixture.requests)request.resolve(request.channel==='map'
 ?{mapTrails:[],clusters:{type:'FeatureCollection',features:[]}}:{items:[],page:1,totalPages:1,totalHits:0});});
 await installRecorder(page); await tilesSettled(page);
}
async function resolvePending(page: Page) {
 await page.evaluate(()=>{const w=window as any; w.fixture.autoResolve=true;for(const r of w.fixture.requests) r.resolve(r.channel==='map'?{mapTrails:[],clusters:{type:'FeatureCollection',features:[]}}:{items:[],page:1,totalPages:1,totalHits:0});});
}
for(const dev of [true,false]) test.describe(`map resize side effects (${dev?'development':'production'})`,()=>{
 test.describe.configure({timeout:60000});
 test.afterAll(closeTileServer);
 let bundle:string; test.beforeAll(async()=>{bundle=await buildMapPageFixture(dev,true,true);});
 for(const action of ['window','splitter-step']) test(`raster tiles cover the resized ${action} without a search response`,async({page})=>{
  await mount(page,bundle);
  if(action.startsWith('splitter')) { await page.locator('.list-resize').press('End'); await tilesSettled(page); }
  // Predecode fixture pixels so the 80ms bound measures resize source
  // invalidation and painting independently of image decoding.
  await page.evaluate(async () => {
   const native = createImageBitmap;
   const pixels = await (await fetch('/tiles/0/0/0.png')).blob();
   const decoded = await Promise.all(Array.from({length: 1024}, () => native(pixels)));
   (window as any).RealMapLibre.addProtocol('fixture-raster', () => {
    if (!decoded.length) throw new Error('Local raster fixture exhausted');
    return Promise.resolve({data:decoded.pop()!});
   });
   (window as any).realMap.getSource('r').setTiles(['fixture-raster://tiles/{z}/{x}/{y}']);
  });
  await tilesSettled(page);
  const loadedBefore = await page.evaluate(() => (window as any).realMap.style.tileManagers.r.getRenderableIds().slice());
  await page.evaluate(() => (window as any).startFrames(1700));
  if(action==='window') await windowBounds(page,{width:1600,height:1000});
  else {
   const r=(await page.locator('.list-resize').boundingBox())!;
   await page.mouse.move(r.x+r.width/2,r.y+r.height/2);
   await page.mouse.down();
   await page.mouse.move(r.x-300,r.y+r.height/2,{steps:1});
   await page.mouse.up();
  }
  await page.waitForTimeout(1800);
  const result=await page.evaluate(()=>{const w=window as any,p=w.probe;return {changes:p.sizeChanges.length,frames:p.frames.length,late:p.frames.filter((f:any)=>f.t-(p.sizeChanges.filter((t:number)=>t<=f.t).at(-1)??0)>=80&&f.miss>0),last:w.coverage()};});
  if(action==='splitter-step') expect(result.changes,'one large resize, with no deadline restart').toBe(1);
  expect(result.frames).toBeGreaterThan(5); expect(result.late,'raster coverage within 80ms without setData').toEqual([]);expect(result.last.miss).toBe(0);
  const loadedAfter = await page.evaluate(() => (window as any).realMap.style.tileManagers.r.getRenderableIds());
  expect(loadedAfter.some((id:string) => !loadedBefore.includes(id)), 'the resize needs a new tile').toBe(true);
 });
 test('markers and popup tips follow the painted frame through resize and easing',async({page})=>{
  await mount(page,bundle); (page as any).tileDelay=600;
  await page.evaluate(()=>{
   const w=window as any,m=w.realMap,M=w.RealMapLibre,c=m.getCenter();
   const dot=document.createElement('div');dot.style.cssText='width:10px;height:10px;background:blue';
   w.marker=new M.Marker({element:dot,anchor:'center'}).setLngLat([c.lng+.02,c.lat+.01]).addTo(m);
   w.popup=new M.Popup({closeOnClick:false,closeOnMove:false,anchor:'bottom',offset:0}).setLngLat([c.lng-.02,c.lat-.01]).setText('Popup').addTo(m);
   w.errors=[];m.on('render',()=>{const root=m.getContainer().getBoundingClientRect(),r=w.marker.getElement().getBoundingClientRect(),p=m.project(w.marker.getLngLat());
    const tip=w.popup.getElement().querySelector('.maplibregl-popup-tip').getBoundingClientRect(),q=m.project(w.popup.getLngLat());
    w.errors.push(Math.max(Math.abs(r.x+r.width/2-root.x-p.x),Math.abs(r.y+r.height/2-root.y-p.y),Math.abs(tip.x+tip.width/2-root.x-q.x),Math.abs(tip.bottom-root.y-q.y)));});
  });
  await windowBounds(page,{width:1500,height:950});
  await quietCounts(page);
  await page.evaluate(()=>{const m=(window as any).realMap;m.easeTo({center:[8.02,47.01],duration:1200});});
  expect(await page.evaluate(()=>(window as any).realMap.isMoving())).toBe(true);
  await page.locator('.list-resize').press('Shift+ArrowLeft');
  await expect.poll(()=>page.evaluate(()=>(window as any).realMap.isMoving())).toBe(false);
  await page.waitForTimeout(500);
  const target = await page.evaluate(()=>(window as any).realMap.getCenter().toArray());
  expect(target[0]).toBeCloseTo(8.02, 9);
  expect(target[1]).toBeCloseTo(47.01, 9);
  const errors=await page.evaluate(()=>(window as any).errors);
  expect(errors.length).toBeGreaterThan(5);expect(Math.max(...errors)).toBeLessThanOrEqual(1);
 });
 test('context recovery survives window while WebGL is lost',async({page})=>{
  await mount(page,bundle);
  await page.evaluate(()=>{const w=window as any,m=w.realMap;w.ext=m.painter.context.gl.getExtension('WEBGL_lose_context');w.ext.loseContext();});
  await expect.poll(()=>page.evaluate(()=>(window as any).realMap.painter.context.gl.isContextLost())).toBe(true);
  await windowBounds(page,{width:1500,height:950});
  await page.waitForTimeout(1100);await page.evaluate(()=>(window as any).ext.restoreContext());
  await expect.poll(()=>page.evaluate(()=>(window as any).realMap.painter.context.gl.isContextLost())).toBe(false);
  await tilesSettled(page);
  const dimensions=await page.evaluate(()=>{const m=(window as any).realMap,c=m.getCanvas(),r=m.getContainer();return{actual:[c.width,c.height],expected:[Math.floor(r.clientWidth*devicePixelRatio),Math.floor(r.clientHeight*devicePixelRatio)]};});
  expect(dimensions.actual).toEqual(dimensions.expected);
  const image=await page.locator('.maplibregl-canvas').screenshot();
  const pixel=await page.evaluate(async data=>{const i=new Image();i.src='data:image/png;base64,'+data;await i.decode();const c=document.createElement('canvas');c.width=i.width;c.height=i.height;const x=c.getContext('2d')!;x.drawImage(i,0,0);return [...x.getImageData(c.width/2,c.height/2,1,1).data].slice(0,3);},image.toString('base64'));
  expect(pixel).toEqual([192,48,48]);
 });
 test('resize refreshes POI cells through the layer manager',async({page})=>{
  const cells=new Map<string,number>();
  const origin=await routes(page,(path,url)=>{
   if(path!=='/api/v1/overpass/interpreter')return;
   const query=url.searchParams.get('data')!;
   if(!cells.has(query))cells.set(query,cells.size+1);
   return {json:{elements:[{type:'node',id:cells.get(query),lat:47.05,lon:7.96,tags:{tourism:'alpine_hut',name:'Hut'}}]}};
  });
  await page.goto(origin+'/map?tl_lat=47.1&tl_lon=8.1&br_lat=46.9&br_lon=7.9');
  await expect.poll(()=>page.evaluate(()=>Boolean((window as any).RealMapLibre))).toBe(true);
  await page.evaluate(state=>localStorage.setItem('map-state',JSON.stringify(state)),mapState);
  await page.addScriptTag({content:bundle});
  await expect.poll(()=>page.evaluate(()=>(window as any).fixture.publications.length)).toBeGreaterThan(0);
  await expect.poll(()=>page.evaluate(()=>Boolean((window as any).realMap?.getSource('overpass')))).toBe(true);
  await resolvePending(page);
  await quietCounts(page);
  await page.evaluate(()=>(window as any).realMap.jumpTo({center:[7.999,46.999],zoom:12.6}));
  await expect.poll(()=>cells.size).toBeGreaterThan(0); await page.waitForTimeout(500); const before=cells.size;
  await windowBounds(page,{width:1600,height:1000});
  await expect.poll(()=>cells.size).toBeGreaterThan(before);
  await expect.poll(async()=>{
   const ids=await page.evaluate(async()=>((await (window as any).realMap.getSource('overpass').getData()).features??[]).map((feature:any)=>feature.properties.id));
   return new Set(ids).size===cells.size;
  }).toBe(true);
 });
});
