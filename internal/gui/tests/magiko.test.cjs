// Real embedded-page scripts, isolated API fixtures, no live account access.
const assert = require('node:assert/strict');
const fs = require('node:fs/promises');
const path = require('node:path');
const { test } = require('node:test');
const { chromium, webkit } = require('playwright');
const assets = path.resolve(__dirname, '../assets');

for (const engine of ['chromium', 'webkit']) {
  test(`${engine}: Magiko preferences and explicit skill import`, async t => {
    const browser = await (engine === 'webkit' ? webkit.launch() : chromium.launch({channel:'chromium'}));
    t.after(() => browser.close());
    for (const lang of ['en', 'zh', 'zh-TW', 'ja', 'de']) {
      await t.test(lang, async () => {
        const page = await browser.newPage({viewport:{width:960,height:900}});
        t.after(() => page.close());
        const errors = [], posts = [];
        let preferences = {enabled:false,text:''};
        page.on('pageerror', e => errors.push(e.message));
        await page.route('**/*', async route => {
          const req = route.request(), url = new URL(req.url());
          const json = data => route.fulfill({json:data});
          if (url.pathname === '/boot.js') return route.fulfill({contentType:'text/javascript',body:`window.bootPrefs={lang:"${lang}",theme:"light",web:true,magiko:true};`});
          if (url.pathname === '/wails/runtime.js') return route.fulfill({contentType:'text/javascript',body:'export const Window = {};'});
          if (url.pathname === '/api/state') return json({agents:[],profiles:[],settings:{lang,theme:'light'}});
          if (url.pathname === '/api/providers') return json({providers:[],presets:[],excluded:[],gateway:{running:true,window:true},plugins:[]});
          if (url.pathname === '/api/groups') return json({groups:[],pools:[]});
          if (url.pathname === '/api/magiko') return json({preferences,skills:['protocol-reverse-engineering','reverse-engineering-api'],configDir:'/fixture/.config/magiko',maxText:12000});
          if (url.pathname === '/api/magiko/preferences') {
            posts.push(url.pathname); preferences=req.postDataJSON(); return json(preferences);
          }
          if (url.pathname === '/api/magiko/skills') { posts.push(url.pathname); return json({installed:['protocol-reverse-engineering','reverse-engineering-api']}); }
          if (url.pathname.startsWith('/api/')) return json({});
          const file = path.join(assets, url.pathname === '/' ? 'index.html' : url.pathname);
          const contentType = {'.html':'text/html','.js':'text/javascript','.css':'text/css','.svg':'image/svg+xml','.png':'image/png'}[path.extname(file)];
          try { await route.fulfill({body:(url.pathname === "/" ? (await fs.readFile(file,"utf8")).replace("<title>magpie</title>","<title>Magiko</title>").replace("<span>magpie</span>","<span>Magiko</span>") : await fs.readFile(file)),contentType}); }
          catch { await route.fulfill({status:404,body:''}); }
        });
        await page.goto('http://magiko.test/?mode=window&view=magiko');
        await page.locator('#mk-config').waitFor({state:'visible'});
        await page.waitForFunction(() => document.getElementById('mk-config').textContent.includes('/fixture/'));
        assert.equal(await page.locator('#mk-enabled').isChecked(),false);
        await page.locator('#mk-text').fill('保留已有 instructions 与 tools。');
        await page.locator('#mk-enabled').check();
        await page.locator('#mk-save').click();
        await page.waitForFunction(() => /Saved|已保存/.test(document.getElementById('mk-status').textContent));
        assert.deepEqual(preferences,{enabled:true,text:'保留已有 instructions 与 tools。'});
        await page.reload();
        await page.waitForFunction(() => document.getElementById('mk-text').value.includes('instructions'));
        assert.equal(await page.locator('#mk-enabled').isChecked(),true);
        // Confirmation cancellation must not import anything.
        const cancel = dialog => dialog.dismiss(); page.on('dialog',cancel);
        await page.locator('#mk-import').click();
        assert.equal(posts.filter(p=>p.endsWith('/skills')).length,0);
        page.off('dialog',cancel); page.once('dialog',dialog=>dialog.accept());
        await page.locator('#mk-import').click();
        await page.waitForFunction(() => /Library: 2|Library：2/.test(document.getElementById('mk-status').textContent));
        assert.equal(posts.filter(p=>p.endsWith('/skills')).length,1);
        // Shell navigation honors the workspace's dirty state in both engines.
        await page.locator('#mk-text').fill('unsaved navigation');
        await page.locator('#nav [data-view="providers"]').click();
        await page.locator('dialog.action-confirm').waitFor({state:'visible'});
        await page.locator('dialog.action-confirm button').first().click();
        assert.equal(await page.locator('#mk-text').inputValue(),'unsaved navigation');
        await page.locator('#nav [data-view="providers"]').click();
        await page.locator('dialog.action-confirm button.primary').click();
        await page.locator('#view-providers').waitFor({state:'visible'});
        await page.locator('#nav [data-view="magiko"]').click();
        await page.waitForFunction(() => document.getElementById('mk-text').value.includes('instructions'));
        // Discard reloads actual saved state, and oversize input never posts.
        await page.locator('#mk-text').fill('unsaved');
        await page.locator('#mk-reset').click();
        await page.waitForFunction(() => document.getElementById('mk-text').value.includes('instructions'));
        const n=posts.length;
        await page.locator('#mk-text').fill('中'.repeat(4001));
        await page.locator('#mk-save').click();
        await page.waitForFunction(() => /12,000/.test(document.getElementById('mk-status').textContent));
        assert.equal(posts.length,n);
        // Narrow layout stays within the viewport and accepts keyboard focus.
        await page.setViewportSize({width:580,height:900});
        await page.locator('#mk-reset').click();
        await page.locator('#mk-text').focus();
        assert.equal(await page.locator('#mk-text').evaluate(el=>el===document.activeElement),true);
        assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1));
        assert.deepEqual(errors,[]);
        if (process.env.ARTIFACT_DIR && lang==='zh') {
          await page.setViewportSize({width:1100,height:1000});
          await page.waitForFunction(() => !document.documentElement.classList.contains("nav-morph"));
          await fs.mkdir(process.env.ARTIFACT_DIR,{recursive:true});
          await page.screenshot({path:path.join(process.env.ARTIFACT_DIR,`magiko-${engine}.png`),fullPage:true,animations:"disabled"});
        }
        await page.close();
      });
    }
  });
}
