import {Call} from '/wails/runtime.js';
const call=(method,...args)=>Call.ByName('main.Desktop.'+method,...args);
const section=document.querySelector('#apps');
const status=document.querySelector('#status');
const refresh=document.querySelector('#refresh');
const dialog=document.querySelector('#message');
const confirm=document.querySelector('#confirm');
const cancel=document.querySelector('#cancel');
const icons={companion:'<svg viewBox="0 0 24 24" fill="none"><path d="m8 6-6 6 6 6m8-12 6 6-6 6m-2-15-4 18" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>',observer:'<svg viewBox="0 0 24 24" fill="none"><path d="M2 12h5l3-8 4 16 3-8h5" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>'};
let apps=[],busy=false,checking=false,action=null,task=null,progress=null,generation=0;
const data=document.querySelector("#data");
function element(tag,cls,txt){const e=document.createElement(tag);if(cls)e.className=cls;if(txt)e.textContent=txt;return e}
function show(title,text,fn=null){action=fn;document.querySelector('#message-title').textContent=title;document.querySelector('#message-text').textContent=text;cancel.hidden=!fn;confirm.textContent=fn?'继续':'确定';dialog.showModal()}
confirm.onclick=()=>{const fn=action;action=null;dialog.close();if(fn)void fn()};
cancel.onclick=()=>{action=null;dialog.close()};
function render(){
 section.replaceChildren();
 for(const app of apps){
  const row=element('div','app');const running=task?.id===app.id;const button=element('button','launch');button.disabled=busy;if(running)row.classList.add('running');
  const tile=element('span','tile');tile.innerHTML=icons[app.id]||'';button.append(tile);
  const words=element('span','words');words.append(element('span','name',app.name));
  if(running){
   const p=progress;words.append(element('span','desc',p?.stage||'准备安装…'));
   const bar=element('progress','progress');bar.setAttribute('aria-label','安装进度');if(p?.total>0){bar.max=p.total;bar.value=Math.min(p.added,p.total)}words.append(bar);
   if(p?.total>0)words.append(element('span','detail',Math.min(p.added,p.total)+' / '+p.total+' 个包'));
   else if(p?.resolved>0)words.append(element('span','detail','已解析 '+p.resolved+' · 下载 '+p.downloaded+' · 复用 '+p.reused));
  }else words.append(element('span','desc',app.installed?app.description:app.note||app.description));
  button.append(words);if(!running)button.append(element('span','indicator',app.installed?'›':app.id==='companion'?'':'↓'));
  button.onclick=()=>{
   if(app.installed){void perform(app,'launch');return}
   if(app.id==='companion'){show(app.name,'新版 Companion 正在开发，尚无可安装的原生版本。');return}
   show('安装 '+app.name,'将从 npm 官方仓库安装可用的原生预览版，安装完成后打开。',()=>perform(app,'install'));
  };
  row.append(button);
  if(running){const stop=element("button","task-cancel","取消");stop.disabled=!progress?.cancellable;stop.onclick=async()=>{try{if(await call("Cancel")){progress={...progress,stage:"正在取消",cancellable:false,cancelled:true};render()}}catch(e){status.textContent=String(e)}};row.append(stop)}
  else if(app.update){const update=element('button','update','更新');update.disabled=busy;update.onclick=()=>show('更新 '+app.name,app.installed+' → '+app.available+'\n更新不会覆盖旧安装或修改课程数据。',()=>perform(app,'update'));row.append(update)}
  section.append(row);
 }
 refresh.disabled=busy||checking;data.disabled=busy;
}
async function check(){
 if(busy||checking)return;
 checking=true;const stamp=generation;render();
 status.textContent='检查更新…';
 try{const result=await call('Check');if(!busy&&stamp===generation){apps=result;status.textContent='';}}
 catch(e){if(!busy&&stamp===generation)status.textContent='无法检查更新，已安装应用仍可打开';}
 finally{checking=false;render()}
}
async function perform(app,mode){
 if(busy)return;busy=true;generation++;let timer=null;let polling=false;
 if(mode!=='launch'){task=app;progress=null;timer=setInterval(async()=>{if(polling||!task)return;polling=true;try{const p=await call('Progress');if(task&&p.active){progress=p;render()}}catch{}finally{polling=false}},250)}
 render();
 try{
  if(mode!=='launch'){status.textContent=mode==='update'?'正在更新 '+app.name+'…':'正在安装 '+app.name+'…';await call('Install',app.id);task=null;progress=null;render()}
  if(mode!=='update'){status.textContent='正在打开 '+app.name+'…';await call('Launch',app.id);await call('Close');return}
  apps=await call('List');status.textContent=app.name+' 已更新';
 }catch(e){if(progress?.cancelled||String(e).includes('context canceled'))status.textContent='已取消，原有版本不受影响';else{status.textContent='操作未完成';show('操作失败',String(e))}}
 finally{if(timer!==null)clearInterval(timer);task=null;progress=null;busy=false;render()}
}
refresh.onclick=check;
data.onclick=async()=>{try{await call('OpenData')}catch(e){show('无法打开目录',String(e))}};
try{apps=await call('List');render();status.textContent='';void check()}catch(e){status.textContent=String(e)}
