import test from 'node:test';
import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {readFile} from 'node:fs/promises';
import {resolve,extname} from 'node:path';
import {chromium} from '../out/ui-tools/node_modules/playwright/index.mjs';
const root=resolve('frontend');
const types={'.html':'text/html','.js':'text/javascript','.css':'text/css','.svg':'image/svg+xml'};
const runtime="export const Call={ByName:(name,...args)=>window.__bridge(name.split('.').at(-1),...args)}";
async function setup(t,{installed='',available='0.2.0',releaseStatus='available',selfUpdate=false,launchFails=false,holdCheck=false}={}){
 const server=createServer(async(req,res)=>{
  try{
   const pathname=new URL(req.url,'http://localhost').pathname;
   if(pathname==='/wails/runtime.js'){res.writeHead(200,{'Content-Type':'text/javascript'});res.end(runtime);return}
   const file=resolve(root,'.'+(pathname==='/'?'/index.html':pathname));
   if(!file.startsWith(root+'/')){res.writeHead(403);res.end();return}
   const data=await readFile(file);res.writeHead(200,{'Content-Type':types[extname(file)]||'text/plain'});res.end(data);
  }catch{res.writeHead(404);res.end()}
 });
 await new Promise(r=>server.listen(0,'127.0.0.1',r));const browser=await chromium.launch({headless:true});
 t.after(async()=>{await browser.close();await new Promise(r=>server.close(r))});
 const page=await browser.newPage({viewport:{width:420,height:300}});const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(({installed,available,releaseStatus,selfUpdate,launchFails,holdCheck})=>{
  const state={installed,available,releaseStatus,selfUpdate,launchFails,holdCheck,channel:'stable',pending:false,calls:[],progress:null,finish:null,held:null};
  window.fixture=state;
  const local=()=>[{id:'companion',name:'Companion',description:'代码管理',releaseStatus:'development',installed:''},{id:'observer',name:'Observer',description:'程序运行观察',installed:state.installed,previous:state.installed?'0.1.9':'',releaseStatus:'unknown',available:''}];
  const checked=()=>local().map(a=>a.id==='companion'?a:{...a,releaseStatus:state.releaseStatus,available:state.releaseStatus==='available'?state.available:'',update:!!a.installed&&a.installed!==state.available&&state.releaseStatus==='available',note:state.releaseStatus==='error'?'软件源 HTTP 503':''});
  const self=()=>({version:'0.1.2',releaseStatus:'unknown',pending:state.pending,pendingVersion:state.pending?'0.1.3':''});
  window.__bridge=async(method,...args)=>{
   state.calls.push([method,...args]);
   if(method==='List')return local();
   if(method==='SelfLocal')return self();
   if(method==='Check'){
    const result=checked();
    if(state.holdCheck){state.holdCheck=false;return new Promise(r=>{state.held=()=>r(result)})}
    return result;
   }
   if(method==='SelfInfo')return {...self(),releaseStatus:'available',available:'0.1.3',update:state.selfUpdate&&!state.pending};
   if(method==='Channel')return state.channel;
   if(method==='SetChannel'){state.channel=args[0];state.releaseStatus='available';return}
   if(method==='Progress')return state.progress||{};
   if(method==='InstallVersion'){state.installed=args[1];return}
   if(method==='Launch'){if(state.launchFails)throw Error('启动失败');return}
   if(method==='UpdateSelfVersion'){
    state.progress={id:'routilet',active:true,cancellable:true,total:5,added:2,stage:'下载与安装'};
    return new Promise((resolve,reject)=>{state.finish={resolve:()=>{state.pending=true;state.progress=null;resolve()},reject}})
   }
   if(method==='Cancel'){state.progress=null;state.finish.reject(Error('context canceled'));return true}
   if(method==='Clean')return {removed:1,protected:2,failed:['Observer: permission denied']};
   if(method==='RollbackTo'||method==='Uninstall'||method==='Ready'||method==='Close'||method==='OpenData'||method==='Shortcut')return;
   throw Error('Unexpected RPC '+method);
  };
 },{installed,available,releaseStatus,selfUpdate,launchFails,holdCheck});
 await page.goto('http://127.0.0.1:'+server.address().port);
 await page.getByRole('button',{name:'Companion 开发中'}).waitFor();
 t.after(()=>assert.deepEqual(errors,[]));
 return page;
}
test('unavailable app is disabled; update check error does not block offline opening',async t=>{
 const p=await setup(t,{releaseStatus:'unavailable'});
 const unavailable=p.getByRole('button',{name:'Observer 暂未发布'});await unavailable.waitFor();assert.equal(await unavailable.isDisabled(),true);
 assert.equal(await p.getByRole('button',{name:'Companion 开发中'}).isDisabled(),true);
 await p.evaluate(()=>{fixture.installed='0.2.0';fixture.releaseStatus='error'});
 await p.getByRole('button',{name:'检查更新',exact:true}).click();
 const open=p.getByRole('button',{name:'Observer 打开'});await open.waitFor();assert.equal(await open.isEnabled(),true);
 await p.getByRole('button',{name:'部分更新检查失败'}).click();
 assert.equal(await p.getByRole('dialog',{name:'更新检查未完成'}).isVisible(),true);
 await p.getByRole('button',{name:'关闭',exact:true}).click();await open.click();
 await p.waitForFunction(()=>fixture.calls.some(c=>c[0]==='Close'));
});
test('install is direct; launch failure reconciles actual installation',async t=>{
 const p=await setup(t,{launchFails:true});
 await p.getByRole('button',{name:'Observer 安装'}).click();
 await p.getByRole('dialog',{name:'已安装 Observer，但打开失败'}).waitFor();
 assert.equal(await p.locator('button[aria-label="Observer 打开"]').isVisible(),true);
 assert.deepEqual(await p.evaluate(()=>fixture.calls.find(c=>c[0]==='InstallVersion')),['InstallVersion','observer','0.2.0']);
 await p.getByRole('button',{name:'关闭',exact:true}).click();
 assert.equal(await p.getByRole('button',{name:'Observer 打开'}).isEnabled(),true);
});
test('menu keyboard, rollback confirmation, settings and stale channel check',async t=>{
 const p=await setup(t,{installed:'0.2.0',available:'0.2.1',holdCheck:true,releaseStatus:'error'});
 const more=p.getByRole('button',{name:'管理 Observer'});
 await more.click();await p.getByRole('menuitem',{name:'创建桌面快捷方式'}).waitFor();
 await p.keyboard.press('ArrowDown');await p.keyboard.press('ArrowDown');await p.keyboard.press('Enter');
 await p.getByRole('dialog',{name:'回退 Observer？'}).waitFor();
 assert.match(await p.locator('#message-text').textContent(),/0\.1\.9/);
 await p.getByRole('button',{name:'取消',exact:true}).click();
 await p.getByRole('button',{name:'设置',exact:true}).click();
 await p.locator('#release-channel').selectOption('test');
 await p.waitForFunction(()=>fixture.channel==='test'&&!document.querySelector('#release-channel').disabled);
 assert.equal(await p.getByRole('dialog',{name:'设置'}).isVisible(),true);
 await p.evaluate(()=>fixture.held());
 await p.getByRole('button',{name:'清理不再使用的应用版本'}).click();
 await p.waitForFunction(()=>document.querySelector('#channel-note').textContent.includes('部分文件未清理'));
 await p.getByRole('button',{name:'关闭设置'}).click();
 await p.getByRole('button',{name:'更新',exact:true}).waitFor();
 assert.equal(await p.getByRole('button',{name:'更新',exact:true}).isVisible(),true);
});
test('self update uses common progress and cancel; pending survives refresh',async t=>{
 const p=await setup(t,{selfUpdate:true});
 await p.getByRole('button',{name:'更新至 v0.1.3'}).click();
 const stop=p.getByRole('button',{name:'取消',exact:true});
 await p.waitForFunction(()=>!document.querySelector('#task-cancel').disabled);
 assert.equal(await p.locator('#task-bar').getAttribute('value'),'2');
 await stop.click();await p.waitForFunction(()=>document.querySelector('#notice').textContent.includes('已取消'));
 await p.getByRole('button',{name:'更新至 v0.1.3'}).click();
 await p.waitForFunction(()=>!document.querySelector('#task-cancel').disabled);
 await p.evaluate(()=>fixture.finish.resolve());
 await p.getByRole('button',{name:'退出并应用更新'}).waitFor();
 assert.equal(await p.getByRole('button',{name:'退出并应用更新'}).isEnabled(),true);
});
