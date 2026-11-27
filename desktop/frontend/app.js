import {Call} from '/wails/runtime.js';
const call=(method,...args)=>Call.ByName('main.Desktop.'+method,...args);
const section=document.querySelector('#apps');
const status=document.querySelector('#status');
const refresh=document.querySelector('#refresh');
const dialog=document.querySelector('#message');
const confirm=document.querySelector('#confirm');
const cancel=document.querySelector('#cancel');
const icons={companion:'/icons/companion.svg',observer:'/icons/observer.svg'};
let apps=[],busy=false,checking=false,action=null,task=null,progress=null,generation=0;
const data=document.querySelector("#data");const self=document.querySelector("#self");let selfInfo=null;let channel="stable";let selfGeneration=0;
function element(tag,cls,txt){const e=document.createElement(tag);if(cls)e.className=cls;if(txt)e.textContent=txt;return e}
function show(title,text,fn=null){action=fn;document.querySelector('#message-title').textContent=title;document.querySelector('#message-text').textContent=text;cancel.hidden=!fn;confirm.textContent=fn?'继续':'确定';dialog.showModal()}
confirm.onclick=()=>{const fn=action;action=null;dialog.close();if(fn)void fn()};
cancel.onclick=()=>{action=null;dialog.close()};
function render(){
 section.replaceChildren();
 for(const app of apps){
  const row=element('div','app');const running=task?.id===app.id;const button=element('button','launch');button.disabled=busy;if(running)row.classList.add('running');
  const tile=element('span','tile');if(icons[app.id]){const img=element('img','app-icon');img.src=icons[app.id];img.alt='';tile.append(img)}button.append(tile);
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
   show('安装 '+app.name,'将从 npm 官方仓库安装当前通道中最新的兼容版本，安装完成后打开。',()=>perform(app,'install'));
  };
  row.append(button);
  if(running){const stop=element("button","task-cancel","取消");stop.disabled=!progress?.cancellable;stop.onclick=async()=>{try{if(await call("Cancel")){progress={...progress,stage:"正在取消",cancellable:false,cancelled:true};render()}}catch(e){status.textContent=String(e)}};row.append(stop)}
  else if(app.update){const update=element('button','update','更新');update.disabled=busy;update.onclick=()=>show('更新 '+app.name,app.installed+' → '+app.available+'\n更新不会覆盖旧安装或修改课程数据。',()=>perform(app,'update'));row.append(update)}
  if(app.installed&&!running){const more=element('button','icon-button','⋯');more.title='管理 '+app.name;more.disabled=busy;more.onclick=()=>manage(app);row.append(more)}
  section.append(row);
 }
 refresh.disabled=busy||checking;data.disabled=busy;self.disabled=busy;self.textContent=selfInfo?.update?"Routilet 更新":selfInfo?.pending?"重启以更新":"Routilet";
}
async function check(){
 if(busy||checking)return;
 checking=true;const stamp=generation;render();
 status.textContent='检查更新…';
 try{const result=await call('Check');if(!busy&&stamp===generation){apps=result;status.textContent='';}}
 catch(e){if(!busy&&stamp===generation)status.textContent='无法检查更新，已安装应用仍可打开';}
 finally{checking=false;render()}
}
function manage(app){
 const box=document.createElement('div');box.className='management';
 const actions=[['修复安装','repair'],...(app.rollback?[['回退版本','rollback']]:[]),['创建桌面快捷方式','shortcut'],['卸载','uninstall']];
 for(const [label,method] of actions){const b=element('button','management-button',label);b.onclick=()=>{dialog.close();box.remove();if(method==='shortcut'){void perform(app,method);return}show(label+' '+app.name,method==='uninstall'?'卸载应用，不删除课程数据；运行中的版本保留。':method==='rollback'?'切换到上一个已安装版本。':'重新安装当前版本。',()=>perform(app,method))};box.append(b)}
 show(app.name,app.installed);dialog.insertBefore(box,dialog.querySelector('.actions'));dialog.addEventListener('close',()=>box.remove(),{once:true});
}
async function perform(app,mode){
 if(busy)return;busy=true;generation++;let timer=null;let polling=false;
 if(['install','update','repair'].includes(mode)){task=app;progress=null;timer=setInterval(async()=>{if(polling||!task)return;polling=true;try{const p=await call('Progress');if(task&&p.active){progress=p;render()}}catch{}finally{polling=false}},250)}
 render();
 try{
  if(['install','update','repair'].includes(mode)){status.textContent=mode==='update'?'正在更新 '+app.name+'…':mode==='repair'?'正在修复 '+app.name+'…':'正在安装 '+app.name+'…';await call(mode==='repair'?'Repair':'Install',app.id);task=null;progress=null;render()}
  if(['rollback','uninstall','shortcut'].includes(mode))await call({rollback:'Rollback',uninstall:'Uninstall',shortcut:'Shortcut'}[mode],app.id);
  if(['launch','install'].includes(mode)){status.textContent='正在打开 '+app.name+'…';await call('Launch',app.id);await call('Close');return}
  apps=await call('List');status.textContent=app.name+({update:' 已更新',repair:' 已修复',rollback:' 已回退',uninstall:' 已卸载',shortcut:' 快捷方式已创建'}[mode]||' 操作完成');
 }catch(e){if(progress?.cancelled||String(e).includes('context canceled'))status.textContent='已取消，原有版本不受影响';else{status.textContent='操作未完成';show('操作失败',String(e))}}
 finally{if(timer!==null)clearInterval(timer);task=null;progress=null;busy=false;render()}
}
refresh.onclick=()=>{void check();void checkSelf()};
data.onclick=async()=>{try{await call('OpenData')}catch(e){show('无法打开目录',String(e))}};
async function checkSelf(){const stamp=++selfGeneration;try{const result=await call('SelfInfo');if(stamp===selfGeneration){selfInfo=result;render()}}catch{}}
self.onclick=async()=>{
 if(busy||dialog.open)return;
 try{channel=await call('Channel')}catch(e){show('无法读取发行通道',String(e));return}
 if(dialog.open||busy)return;
 const box=document.createElement('div');box.className='management';
 const label=element('label','desc','发行通道');label.htmlFor='release-channel';
 const select=element('select','channel-select');select.id='release-channel';
 for(const [id,name] of [['stable','正式通道'],['test','测试通道']]){const o=element('option','',name);o.value=id;select.append(o)}select.value=channel;
 const explanation=element('span','detail','测试通道允许更新的测试版；正式版较新时仍使用正式版。切换通道不会自动降级已安装版本。');
 select.onchange=async()=>{
  const requested=select.value;select.disabled=true;
  try{
   await call('SetChannel',requested);channel=requested;generation++;selfGeneration++;selfInfo=null;
   apps=await call('List');render();dialog.close();status.textContent=requested==='test'?'已切换测试通道':'已切换正式通道';
   while(checking)await new Promise(r=>setTimeout(r,50));
   void check();void checkSelf();
  }catch(e){select.value=channel;explanation.textContent=String(e)}
  finally{select.disabled=false}
 };
 box.append(label,select,explanation);
 const button=(name,fn)=>{const b=element('button','management-button',name);b.onclick=()=>{dialog.close();box.remove();void fn()};box.append(b)};
 if(selfInfo?.update)button('更新 Routilet 至 '+selfInfo.available,()=>show('更新 Routilet',selfInfo.version+' → '+selfInfo.available+'，完成后重新打开。',updateSelf));
 else if(selfInfo?.pending)button('关闭并准备重启',()=>call('Close'));
 button('清理不再使用的应用版本',async()=>{try{await call('Clean');status.textContent='清理完成，保留当前、上一版及运行中的版本'}catch(e){show('清理失败',String(e))}});
 show('Routilet '+(selfInfo?.version||''),selfInfo?.note||'管理与发行通道');
 dialog.insertBefore(box,dialog.querySelector('.actions'));dialog.addEventListener('close',()=>box.remove(),{once:true});
};
async function updateSelf(){
 if(busy)return;busy=true;generation++;render();status.textContent='正在更新 Routilet…';
 const timer=setInterval(async()=>{try{const p=await call('Progress');if(p.active)status.textContent=p.stage+(p.total>0?' '+p.added+'/'+p.total+' 个包':'')}catch{}},300);
 try{await call('UpdateSelf');status.textContent='新版已准备就绪，重新打开以更新';await checkSelf()}
 catch(e){status.textContent='更新未完成';show('更新失败',String(e))}
 finally{clearInterval(timer);busy=false;render()}
}
try{apps=await call('List');render();await call('Ready');status.textContent='';void check();void checkSelf()}catch(e){status.textContent=String(e)}
