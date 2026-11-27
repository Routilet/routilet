import {Call} from '/wails/runtime.js';
const call=(method,...args)=>Call.ByName('main.Desktop.'+method,...args);
const section=document.querySelector('#apps');
const status=document.querySelector('#status');
const refresh=document.querySelector('#refresh');
const dialog=document.querySelector('#message');
const confirm=document.querySelector('#confirm');
const cancel=document.querySelector('#cancel');
const icons={companion:'<svg viewBox="0 0 24 24" fill="none"><path d="m8 6-6 6 6 6m8-12 6 6-6 6m-2-15-4 18" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>',observer:'<svg viewBox="0 0 24 24" fill="none"><path d="M2 12h5l3-8 4 16 3-8h5" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>'};
let apps=[],busy=false,checking=false,action=null;
function element(tag,cls,txt){const e=document.createElement(tag);if(cls)e.className=cls;if(txt)e.textContent=txt;return e}
function show(title,text,fn=null){action=fn;document.querySelector('#message-title').textContent=title;document.querySelector('#message-text').textContent=text;cancel.hidden=!fn;confirm.textContent=fn?'继续':'确定';dialog.showModal()}
confirm.onclick=()=>{const fn=action;action=null;dialog.close();if(fn)void fn()};
cancel.onclick=()=>{action=null;dialog.close()};
function render(){
 section.replaceChildren();
 for(const app of apps){
  const row=element('div','app');const button=element('button','launch');button.disabled=busy;
  const tile=element('span','tile');tile.innerHTML=icons[app.id]||'';button.append(tile);
  const words=element('span','words');words.append(element('span','name',app.name));
  words.append(element('span','desc',app.installed?app.description:app.note||app.description));
  button.append(words);button.append(element('span','indicator',app.installed?'›':app.id==='companion'?'':'↓'));
  button.onclick=()=>{
   if(app.installed){void perform(app,'launch');return}
   if(app.id==='companion'){show(app.name,'新版 Companion 正在开发，尚无可安装的原生版本。');return}
   show('安装 '+app.name,'将从 npm 官方仓库安装可用的原生预览版，安装完成后打开。',()=>perform(app,'install'));
  };
  row.append(button);
  if(app.update){const update=element('button','update','更新');update.disabled=busy;update.onclick=()=>show('更新 '+app.name,app.installed+' → '+app.available+'\n更新不会覆盖旧安装或修改课程数据。',()=>perform(app,'update'));row.append(update)}
  section.append(row);
 }
 refresh.disabled=busy||checking;
}
async function check(){
 if(busy||checking)return;
 checking=true;render();
 status.textContent='检查更新…';
 try{const result=await call('Check');if(!busy){apps=result;status.textContent='';}}
 catch(e){if(!busy)status.textContent='无法检查更新，已安装应用仍可打开';}
 finally{checking=false;render()}
}
async function perform(app,mode){
 if(busy)return;busy=true;render();
 try{
  if(mode!=='launch'){status.textContent=mode==='update'?'正在更新 '+app.name+'…':'正在安装 '+app.name+'…';await call('Install',app.id)}
  if(mode!=='update'){status.textContent='正在打开 '+app.name+'…';await call('Launch',app.id);await call('Close');return}
  apps=await call('List');status.textContent=app.name+' 已更新';
 }catch(e){status.textContent='操作未完成';show('操作失败',String(e))}
 finally{busy=false;render()}
}
refresh.onclick=check;
try{apps=await call('List');render();status.textContent='';void check()}catch(e){status.textContent=String(e)}
