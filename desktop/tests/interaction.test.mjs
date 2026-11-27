import test from 'node:test';
import assert from 'node:assert/strict';
import {Session,appView,mergeLocal,selfView,checkSummary,progressView,errorText} from '../frontend/state.js';
const observer={id:'observer',name:'Observer',description:'观察',installed:'',available:'',releaseStatus:'unknown',update:false};
test('only a compatible discovered release is installable',()=>{
 for(const releaseStatus of ['unknown','error','unavailable','development'])assert.equal(appView({...observer,releaseStatus}).enabled,false);
 assert.equal(appView({...observer,releaseStatus:'available',available:'0.2.0'}).action,'安装');
 assert.equal(appView({...observer,releaseStatus:'error',installed:'0.2.0'}).action,'打开');
 assert.equal(appView({...observer,releaseStatus:'error',installed:'0.2.0'}).enabled,true);
});
test('local reconciliation after install or launch failure reflects installed version; preserves other update notices',()=>{
 const a={...observer,available:'0.2.1',releaseStatus:'available'};
 const b={id:'other',installed:'1.0.0',available:'1.1.0',releaseStatus:'available',update:true};
 const result=mergeLocal([a,b],[{...observer,installed:'0.2.1'},{id:'other',installed:'1.0.0',releaseStatus:'unknown'}]);
 assert.equal(appView(result[0]).action,'打开');assert.equal(result[1].update,true);
});
test('channel change and operations invalidate stale network results',()=>{
 const s=new Session();let stamp=s.startCheck();assert.ok(stamp);
 assert.ok(s.beginSetting());assert.equal(s.current(stamp),false);assert.equal(s.startTask('observer','安装',true),null);
 s.apps=[{...observer,available:'0.2.0',releaseStatus:'available'}];s.resetRemote();assert.equal(appView(s.apps[0]).enabled,false);
 s.endSetting();stamp=s.startCheck();const t=s.startTask('observer','安装',true);assert.ok(t);assert.equal(s.finishCheck(stamp),false);
 assert.equal(s.startTask('observer','更新',true),null);assert.equal(s.finishTask(t),true);assert.ok(s.startCheck());
});
test('self update and pending state are independent of online availability',()=>{
 assert.equal(selfView({pending:true,pendingVersion:'0.1.2',releaseStatus:'error'}).action,'exit');
 assert.equal(selfView({pending:true,update:true,available:'0.1.3',releaseStatus:'available'}).action,'update');
 assert.equal(checkSummary([{...observer,installed:'0.2.0',releaseStatus:'error'}],{releaseStatus:'available'}),'部分更新检查失败');
});
test('progress is scoped to the current task; cancellation is only available before activation',()=>{
 const t={id:'routilet',label:'更新',installing:true};
 assert.equal(progressView(t,{id:'observer',active:true,total:5,added:4}).indeterminate,true);
 const p={id:'routilet',active:true,total:5,added:3,cancellable:true,stage:'下载'};
 assert.equal(progressView(t,p).added,3);assert.equal(progressView(t,p).cancellable,true);
 assert.equal(progressView({...t,cancelAccepted:true},p).cancellable,false);
 assert.equal(progressView(t,{...p,total:0,cancellable:false,stage:'激活'}).indeterminate,true);
});
test('errors have user-readable summary and retain technical detail',()=>{
 const p=errorText('pnpm: exit status 1\nERR_PNPM_FETCH');assert.match(p.summary,/软件包/);assert.match(p.raw,/ERR_PNPM/);
});

test('disabled apps remain manageable without install, launch or update',()=>{
 const a={id:'observer',description:'观察',installed:'0.2.0',available:'0.2.1',update:true,releaseStatus:'available',disabled:true};
 assert.equal(appView(a).action,'已禁用');assert.equal(appView(a).manage,true);assert.equal(appView(a).enabled,false);assert.equal(appView(a).update,false);
 assert.equal(appView({...a,installed:''}).manage,true);
 assert.equal(appView({...a,disabled:false}).action,'打开');
 assert.equal(mergeLocal([a],[{...a,disabled:false,releaseStatus:'unknown',update:false}])[0].releaseStatus,'unknown');
 assert.equal(checkSummary([{...a,releaseStatus:'unknown'}],{releaseStatus:'available'}),'检查完成');
});

test('optional adapter installs without becoming a launchable GUI',()=>{
 const a={id:'accoding-adapter',component:true,description:'在线组件',installed:'',available:'0.1.0-beta.1',releaseStatus:'available'};
 assert.equal(appView(a).action,'安装');assert.equal(appView(a).enabled,true);
 const installed={...a,installed:a.available};assert.equal(appView(installed).action,'已安装');assert.equal(appView(installed).enabled,false);assert.equal(appView(installed).manage,true);
 assert.equal(appView({...installed,available:'0.1.0-beta.2',update:true}).update,true);
 assert.equal(appView({...installed,disabled:true,update:true}).update,false);
});
