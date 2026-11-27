// Pure view state. No DOM or Wails runtime dependency.
export function mergeLocal(previous, local) {
 const cached=new Map(previous.map(a=>[a.id,a]));
 return local.map(a=>{
  const old=cached.get(a.id);
  if(!old||a.disabled!==old.disabled||a.preferenceError||old.preferenceError||a.localError||a.releaseStatus==='development'||old.installed!==a.installed||old.localError) return a;
  return {...a,available:old.available,update:old.update,releaseStatus:old.releaseStatus,note:old.note};
 });
}
export function mergeSelf(previous, local) {
 if(!previous||previous.version!==local.version||previous.pendingVersion!==local.pendingVersion) return local;
 return {...local,available:previous.available,update:previous.update,releaseStatus:previous.releaseStatus,note:previous.note};
}
export function appView(app) {
 const installed=!!app.installed,ready=app.releaseStatus==='available'&&!!app.available;
 const base=app.description+(installed?' · v'+app.installed:'');
 if(app.preferenceError)return {description:base,action:'设置异常',enabled:false,update:false,manage:true};
 if(app.disabled)return {description:base,action:'已禁用',enabled:false,update:false,manage:true};
 if(installed) return {description:base,action:app.component?'已安装':'打开',enabled:!app.component,update:ready&&app.update,manage:true};
 if(app.releaseStatus==='development')return {description:base,action:'开发中',enabled:false,update:false,manage:true};
 if(ready)return {description:base+(app.localError?' · 安装记录异常':' · v'+app.available),action:app.localError?'重新安装':'安装',enabled:true,update:false,manage:true};
 return {description:base+(app.localError?' · 安装记录异常':''),action:({unknown:'待检查',unavailable:'暂未发布',error:'检查失败'})[app.releaseStatus]||'待检查',enabled:false,update:false,manage:true};
}
export function selfView(info) {
 if(info?.update&&info.available&&info.releaseStatus==='available')return {action:'update',label:'更新至 v'+info.available};
 if(info?.pending)return {action:'exit',label:'退出并应用更新',title:'退出后重新打开以使用 v'+info.pendingVersion};
 return {action:'',label:''};
}
export function checkSummary(apps,self,checking=false) {
 if(checking)return '检查更新中…';
 if(apps.some(a=>!a.disabled&&(a.releaseStatus==='error'||a.preferenceError))||self?.releaseStatus==='error')return '部分更新检查失败';
 if(apps.some(a=>!a.disabled&&!a.preferenceError&&a.releaseStatus==='unknown')||!self||self.releaseStatus==='unknown')return '尚未检查更新';
 if(self.releaseStatus==='unavailable')return '自身暂无兼容版本';
 if(apps.some(a=>!a.disabled&&a.update)||self.update)return '有可用更新';
 if(self.pending)return '新版已准备就绪';
 return '检查完成';
}
export function progressView(task,p){
 if(!task)return null;
 if(!task.installing)return {stage:task.label,indeterminate:true,cancellable:false,detail:''};
 const valid=p?.active&&p.id===task.id;
 const stage=task.cancelAccepted?'正在取消…':valid?p.stage||task.label:task.label;
 const total=valid&&p.total>0?p.total:0;
 const added=total?Math.min(Math.max(p.added||0,0),total):0;
 return {stage,indeterminate:!total,total,added,cancellable:valid&&p.cancellable&&!task.cancelRequested&&!task.cancelAccepted,
 detail:total?'已添加 '+added+' / '+total+' 个包':valid&&p.resolved>0?'已解析 '+p.resolved+' · 下载 '+p.downloaded+' · 复用 '+p.reused:''};
}
export function errorText(e) {
 const raw=typeof e==='string'?e:e?.message||String(e);
 if(/可用版本已变化|回退目标已变化/.test(raw))return {summary:'版本已变化，请重新检查后操作。',raw};
 if(/另一 Routilet 实例/.test(raw))return {summary:'另一个 Routilet 实例正在操作，请稍后重试。',raw};
 if(/context deadline exceeded|ETIMEDOUT|ENOTFOUND|ECONNRESET|网络|软件源 HTTP|dial tcp/i.test(raw))return {summary:'无法完成网络请求，请检查连接后重试。',raw};
 if(/随附 pnpm/.test(raw))return {summary:'找不到或无法访问随附运行组件，请完整解压发行包。',raw};
 if(/校验应用|解析软件包目录|解析原生入口|读取应用清单/.test(raw))return {summary:'安装文件校验失败。请展开技术详情查看具体阶段与路径。',raw};
 if(/pnpm.*失败|pnpm:|ERR_PNPM/i.test(raw))return {summary:'软件包下载或安装失败。可展开技术详情查看原因。',raw};
 if(/校验|清单|协议/.test(raw))return {summary:'软件包未通过兼容性或完整性检查。',raw};
 return {summary:raw.split('\n')[0].slice(0,160)||'操作未完成，请重试。',raw};
}
export class Session {
 constructor(){this.apps=[];this.self=null;this.generation=0;this.checking=false;this.task=null;this.setting=false;}
 get busy(){return !!this.task||this.setting}
 invalidate(){this.generation++;this.checking=false}
 startCheck(){if(this.busy||this.checking)return null;this.checking=true;return ++this.generation}
 current(stamp){return stamp===this.generation}
 finishCheck(stamp){if(!this.current(stamp))return false;this.checking=false;return true}
 setLocal(apps,self){this.apps=mergeLocal(this.apps,apps);this.self=mergeSelf(this.self,self)}
 resetRemote(){this.apps=this.apps.map(a=>a.disabled||a.preferenceError||a.releaseStatus==='development'?a:{...a,releaseStatus:'unknown',available:'',update:false,note:''});if(this.self)this.self={...this.self,releaseStatus:'unknown',available:'',update:false,note:''}}
 startTask(id,label,installing=false){if(this.busy)return null;this.invalidate();return this.task={id,label,installing,cancelRequested:false,cancelAccepted:false,progress:null}}
 finishTask(task){if(this.task!==task)return false;this.task=null;return true}
 beginSetting(){if(this.busy)return false;this.invalidate();this.setting=true;return true}
 endSetting(){this.setting=false}
}
