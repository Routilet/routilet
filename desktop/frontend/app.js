import {Call} from '/wails/runtime.js';
import {Session,appView,selfView,checkSummary,progressView,errorText} from './state.js';
import {$,element,ask} from './dialogs.js';
import {openMenu,closeMenu} from './menu.js';
const call=(method,...args)=>Call.ByName('main.Desktop.'+method,...args);
const s=new Session(),rows=new Map();
const section=$('#apps'),components=$('#components'),componentList=$('#component-list'),refresh=$('#refresh'),selfAction=$('#self-action'),settings=$('#settings');
const settingsDialog=$('#settings-dialog'),channelSelect=$('#release-channel'),channelNote=$('#channel-note'),clean=$('#clean'),settingsClose=$('#settings-close');
const taskPanel=$('#task'),taskStage=$('#task-stage'),taskDetail=$('#task-detail'),taskBar=$('#task-bar'),taskCancel=$('#task-cancel');
const checkStatus=$('#check-status'),notice=$('#notice'),version=$('#version');
const downloadSelect=$('#download-source'),downloadNote=$('#download-note');
let downloadSource='auto',downloadLoaded=false;
const downloadDescription=()=>downloadSource==='auto'?'优先使用国内镜像检查更新和下载安装，镜像不可用时自动切换官方源。':'始终使用 npm 官方源下载。';
let channel='stable',channelLoaded=false,settingsOpening=false,settingsReturn=null,noticeTimer=null,settingsEpoch=0;
const settingRead=(method)=>{let timer;return Promise.race([Promise.resolve().then(()=>call(method)),new Promise((_,reject)=>{timer=setTimeout(()=>reject(Error('读取超时，请关闭设置后重试')),4000)})]).finally(()=>clearTimeout(timer))};
function feedback(text,persistent=false) {
 clearTimeout(noticeTimer);notice.textContent=text;notice.hidden=!text;
 if(text&&!persistent)noticeTimer=setTimeout(()=>{notice.hidden=true;notice.textContent=''},6500);
}
async function report(heading,e){const {summary,raw}=errorText(e);await ask({heading,text:summary,technical:raw===summary?'':raw})}
function appById(id){return s.apps.find(a=>a.id===id)}
function createRow(id) {
 const row=element('div','app'),launch=element('button','launch'),tile=element('img','tile'),words=element('span','words'),name=element('span','name'),desc=element('span','desc'),indicator=element('span','indicator'),update=element('button','update ui-button','更新'),more=element('button','icon-button more ui-button ui-icon-button');
 const icon=document.createElementNS('http://www.w3.org/2000/svg','svg');icon.setAttribute('class','ui-icon');icon.setAttribute('aria-hidden','true');const use=document.createElementNS('http://www.w3.org/2000/svg','use');use.setAttribute('href','/ui/icons/sprite.svg#ellipsis');icon.append(use);more.append(icon);
 update.dataset.kind='secondary';tile.src='/icons/'+id+'.svg';tile.alt='';words.append(name,desc);launch.append(tile,words,indicator);row.append(launch,update,more);
 launch.onclick=()=>{const a=appById(id);if(!a||s.busy||!appView(a).enabled)return;void perform(a,a.installed?'launch':'install')};
 update.onclick=()=>{const a=appById(id);if(a&&!s.busy&&appView(a).update)void perform(a,'update')};
 more.setAttribute('aria-haspopup','menu');more.setAttribute('aria-expanded','false');more.onclick=()=>{const a=appById(id);if(a&&!s.busy)manage(a,more)};
 (appById(id)?.component?componentList:section).append(row);const record={row,launch,name,desc,indicator,update,more};rows.set(id,record);return record;
}
function render(){
 components.hidden=!s.apps.some(a=>a.component);
 for(const app of s.apps){
  const r=rows.get(app.id)||createRow(app.id),v=appView(app);
  const target=app.component?componentList:section;
  if(r.row.parentElement!==target)target.append(r.row);
  r.name.textContent=app.name;r.desc.textContent=v.description;r.indicator.textContent=v.action;
  r.launch.disabled=s.busy||!v.enabled;r.launch.title=app.preferenceError||app.localError||app.note||v.action+' '+app.name;
  r.launch.setAttribute('aria-label',app.name+' '+v.action);
  r.update.hidden=!v.update;r.update.disabled=s.busy;r.update.title='更新至 v'+app.available;
  r.more.hidden=!v.manage;r.more.disabled=s.busy;r.more.title='管理 '+app.name;r.more.setAttribute('aria-label','管理 '+app.name);
  r.row.classList.toggle('unavailable',!v.enabled&&!(app.component&&app.installed&&!app.disabled&&!app.preferenceError));
  if(!v.manage&&r.more.getAttribute('aria-expanded')==='true')closeMenu();
 }
 for(const [id,r] of rows){if(!appById(id)){r.row.remove();rows.delete(id)}}
 const v=selfView(s.self);
 version.textContent=s.self?.version?'v'+s.self.version:'';
 selfAction.hidden=!v.action;selfAction.textContent=v.label;selfAction.title=v.title||v.label;selfAction.disabled=s.busy;
 selfAction.onclick=()=>{if(s.busy)return;const current=selfView(s.self);if(current.action==='update')void perform({id:'routilet',name:'Routilet',available:s.self.available},'self');else if(current.action==='exit')void exitForUpdate()};
 refresh.disabled=s.busy||s.checking;refresh.classList.toggle('checking',s.checking);refresh.setAttribute('aria-busy',String(s.checking));
 settings.disabled=s.busy||settingsOpening;
 checkStatus.textContent=checkSummary(s.apps,s.self,s.checking);
 const problems=[...s.apps.filter(a=>a.preferenceError||(!a.disabled&&a.releaseStatus==='error')).map(a=>a.name+': '+(a.preferenceError||a.note)),...(s.self?.releaseStatus==='error'?['Routilet: '+s.self.note]:[])];
 checkStatus.title=problems.join('\n')||checkStatus.textContent;
 checkStatus.classList.toggle('check-error',problems.length>0);checkStatus.disabled=!problems.length;
 checkStatus.onclick=()=>{if(problems.length&&!s.busy)void ask({heading:'更新检查未完成',text:'无法确认部分软件的最新版本。已安装的应用仍可离线打开。',technical:problems.join('\n')})};
 renderTask();
}
function renderTask(){
 const p=progressView(s.task,s.task?.progress);taskPanel.hidden=!p;if(!p)return;
 taskStage.textContent=s.task.cancelAccepted?'正在取消…':s.task.label;taskDetail.textContent=p.stage===s.task.label?'正在处理…':p.stage;taskDetail.hidden=false;
 const d=p.download;taskBar.hidden=!d;
 if(d){
  taskBar.max=d.total;taskBar.value=Math.min(d.bytes,d.total*.999);
  const mb=n=>(n/1e6).toFixed(1);
  taskDetail.textContent='正在下载文件 · '+mb(d.bytes)+' / '+mb(d.total)+' MB · '+Math.min(99,Math.floor(d.bytes/d.total*100))+'%';
 }
 
 taskCancel.hidden=!s.task.installing;taskCancel.disabled=!p.cancellable;taskCancel.textContent=s.task.cancelAccepted?'正在取消…':'取消';
}
async function local(){
 const [apps,self]=await Promise.all([call('List'),call('SelfLocal')]);s.setLocal(apps,self);render();
}
async function check(){
 const stamp=s.startCheck();if(stamp===null)return;
 render();
 const updateApps=async()=>{
  try{const apps=await call('Check');if(s.current(stamp)){s.apps=apps;render()}}
  catch(e){if(s.current(stamp)){s.apps=s.apps.map(a=>a.disabled||a.preferenceError||a.releaseStatus==='development'?a:{...a,releaseStatus:'error',available:'',update:false,note:errorText(e).raw});render()}}
 };
 const updateSelf=async()=>{
  try{const info=await call('SelfInfo');if(s.current(stamp)){s.self=info;render()}}
  catch(e){if(s.current(stamp)){s.self={...s.self,releaseStatus:'error',available:'',update:false,note:errorText(e).raw};render()}}
 };
 await Promise.all([updateApps(),updateSelf()]);
 if(s.finishCheck(stamp))render();
}
function manage(a,button){
 const actions=[];
 if(a.installed&&!a.disabled&&!a.preferenceError){
  if(!a.component)actions.push({label:'创建桌面快捷方式',run:()=>perform(a,'shortcut')},{separator:true});
  actions.push({label:'重新安装',run:()=>confirmOperation(a,'repair')});
  if(a.previous)actions.push({label:'恢复至 v'+a.previous,run:()=>confirmOperation(a,'rollback')});
  actions.push({separator:true});
 }
 actions.push({label:a.disabled||a.preferenceError?'启用':'停用',run:()=>perform(a,a.disabled||a.preferenceError?'enable':'disable')});
 if(a.installed)actions.push({separator:true},{label:'卸载',danger:true,run:()=>confirmOperation(a,'uninstall')});
 openMenu(button,actions);
}
async function confirmOperation(a,mode){
 const labels={repair:'重新安装',rollback:'恢复',uninstall:'卸载'};
 const texts={repair:'重新下载并安装 v'+a.installed+'，保留课程数据。',rollback:'从 v'+a.installed+' 切换到 v'+a.previous+'，保留课程数据。',uninstall:'移除已安装的 '+a.name+'，保留课程数据和运行中的版本。'};
 if(await ask({heading:labels[mode]+' '+a.name+'？',text:texts[mode],action:labels[mode],confirming:true,danger:mode==='uninstall'}))await perform(a,mode);
}
function poll(task){
 async function tick(){
  if(s.task!==task||!task.installing)return;
  try{const p=await call('Progress');if(s.task===task&&p.active&&p.id===task.id){task.progress=p;renderTask()}}catch{}
  if(s.task===task&&task.installing)task.timer=setTimeout(tick,250);
 }
 void tick();
}
taskCancel.onclick=async()=>{
 const task=s.task;if(!task||!progressView(task,task.progress)?.cancellable)return;
 task.cancelRequested=true;renderTask();
 try{const accepted=await call('Cancel');if(s.task===task){task.cancelAccepted=accepted;if(!accepted){task.progress={...task.progress,cancellable:false};feedback('已进入不可取消的阶段')}}}
 catch(e){if(s.task===task)feedback('取消请求失败，请稍后重试',true)}
 finally{if(s.task===task){task.cancelRequested=false;renderTask()}}
};
async function perform(a,mode){
 const installing=['install','update','repair','self'].includes(mode);
 const verbs={install:'安装',update:'更新',repair:'重新安装',self:'更新',launch:'打开',rollback:'恢复',uninstall:'卸载',shortcut:'创建快捷方式',enable:'启用',disable:'停用'};
 const task=s.startTask(a.id,'正在'+verbs[mode]+' '+a.name+'…',installing);if(!task)return;
 closeMenu();feedback('');render();let operationError=null,installed=false,closing=false;
 if(installing)poll(task);
 try{
  if(mode==='self')await call('UpdateSelfVersion',a.available);
  else if(mode==='install'||mode==='update'){await call('InstallVersion',a.id,a.available);installed=mode==='install'&&!a.component}
  else if(mode==='enable'||mode==='disable')await call('SetDisabled',a.id,mode==='disable');
  else if(mode==='repair')await call('Repair',a.id);
  else if(mode==='rollback')await call('RollbackTo',a.id,a.previous);
  else if(mode==='uninstall'||mode==='shortcut')await call(mode==='uninstall'?'Uninstall':'Shortcut',a.id);
  // Always reconcile persisted state before launching, including first install.
  task.installing=false;task.progress=null;clearTimeout(task.timer);
  if((mode==='install'||mode==='launch')&&!a.component){
   task.label='正在打开 '+a.name+'…';renderTask();
   if(mode==='install')await local();
   await call('Launch',a.id);await call('Close');closing=true;s.finishTask(task);render();
  }else{
   const done={install:'已安装',update:'已更新',repair:'已重新安装',rollback:'已恢复',uninstall:'已卸载',shortcut:'快捷方式已创建',enable:'已启用',disable:'已停用',self:'新版已准备就绪，退出后重新打开'};
   feedback(mode==='self'?done.self:a.name+' '+done[mode],mode==='self');
  }
 }catch(e){operationError=e}
 finally{
  clearTimeout(task.timer);
  if(!closing){
   task.installing=false;task.progress=null;
   try{await local()}catch(e){feedback('无法刷新本地状态，请重新打开 Routilet',true)}
   s.finishTask(task);render();
   if(operationError){
    const canceled=task.cancelAccepted||/context canceled/i.test(errorText(operationError).raw);
    if(canceled)feedback('已取消，原有版本不受影响');
    else{feedback(installed?'已安装，但未能打开':'操作未完成',true);void report(installed?'已安装 '+a.name+'，但打开失败':verbs[mode]+'失败',operationError)}
   }
   void check();
  }
 }
}
async function exitForUpdate(){
 if(s.busy)return;
 try{await call('Close')}catch(e){void report('退出失败',e)}
}
async function openData(){if(s.busy)return;try{await call('OpenData')}catch(e){if(settingsDialog.open)channelNote.textContent='无法打开数据目录：'+errorText(e).summary;else void report('无法打开数据目录',e)}}
refresh.onclick=()=>void check();$('#settings-data').onclick=()=>void openData();
function renderSettings(){
 channelSelect.value=channelLoaded?channel:'';channelSelect.disabled=s.setting||!channelLoaded;
 downloadSelect.value=downloadLoaded?downloadSource:'';downloadSelect.disabled=s.setting||!downloadLoaded;
 settingsClose.disabled=s.setting;clean.disabled=s.setting||settingsOpening;$('#settings-data').disabled=s.setting||settingsOpening;
 $('#settings-version').textContent=s.self?.version?'Routilet v'+s.self.version:'Routilet';
}
settings.onclick=async()=>{
 if(s.busy||settingsOpening||settingsDialog.open)return;settingsOpening=true;const epoch=++settingsEpoch;settingsReturn=settings;render();closeMenu();
 channelLoaded=false;downloadLoaded=false;
 channelSelect.options[0].textContent='读取中';downloadSelect.options[0].textContent='读取中';
 downloadNote.textContent='正在读取设置…';channelNote.textContent='正在读取设置…';renderSettings();settingsDialog.showModal();settingsClose.focus();
 const current=()=>epoch===settingsEpoch&&settingsDialog.open;
 async function read(method,select,note,allowed,apply,description){
  try{const value=await settingRead(method);if(!current())return;if(!allowed.includes(value))throw Error('设置值无效');apply(value);note.textContent=description()}
  catch(e){if(!current())return;select.options[0].textContent='读取失败';note.textContent=errorText(e).summary+'；关闭设置后可重试。'}
  finally{if(current())renderSettings()}
 }
 await Promise.all([
  read('Channel',channelSelect,channelNote,['stable','test'],v=>{channel=v;channelLoaded=true},()=> '测试版可提前体验新功能，但可能存在问题。切换通道不会自动降级已安装的应用。'),
  read('DownloadSource',downloadSelect,downloadNote,['auto','official'],v=>{downloadSource=v;downloadLoaded=true},downloadDescription)
 ]);
 if(current()){settingsOpening=false;renderSettings();render()}
};
function closeSettings(){if(s.setting)return;settingsEpoch++;settingsOpening=false;settingsDialog.close();render();settingsReturn?.focus()}
settingsClose.onclick=closeSettings;settingsDialog.addEventListener('cancel',e=>{e.preventDefault();closeSettings()});
channelSelect.onchange=async()=>{
 const requested=channelSelect.value;if(!channelLoaded||requested===channel||!s.beginSetting()){renderSettings();return}
 channelNote.textContent='正在保存…';renderSettings();render();
 try{
  await call('SetChannel',requested);channel=requested;s.resetRemote();
  channelNote.textContent='已保存。正在重新检查更新；已安装版本不会降级。';
 }catch(e){channelNote.textContent='保存失败：'+errorText(e).summary}
 finally{s.endSetting();renderSettings();render();void check()}
};
downloadSelect.onchange=async()=>{
 const requested=downloadSelect.value;if(!downloadLoaded||requested===downloadSource||!s.beginSetting()){renderSettings();return}
 downloadNote.textContent='正在保存…';renderSettings();render();
 try{await call('SetDownloadSource',requested);downloadSource=requested;downloadNote.textContent=downloadDescription()}
 catch(e){downloadNote.textContent='保存失败：'+errorText(e).summary}
 finally{s.endSetting();renderSettings();render();void check()}
};
clean.onclick=async()=>{
 if(settingsOpening||!s.beginSetting())return;renderSettings();render();channelNote.textContent='正在清理未使用的应用版本…';
 try{const r=await call('Clean');channelNote.textContent='已清理 '+r.removed+' 个旧版本，保留 '+r.protected+' 个当前、可恢复或受保护的版本。'+(r.failed?.length?' 部分文件未清理：'+r.failed.join('；'):'')}
 catch(e){channelNote.textContent='清理失败：'+errorText(e).summary}
 finally{s.endSetting();renderSettings();render();void check()}
};
try{await local();await call('Ready');void check()}catch(e){feedback('无法读取本地状态',true);void report('Routilet 启动失败',e)}
