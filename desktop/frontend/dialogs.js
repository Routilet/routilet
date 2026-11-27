export const $=s=>document.querySelector(s);
export function element(tag,cls='',text=''){const node=document.createElement(tag);if(cls)node.className=cls;if(text)node.textContent=text;return node}
const modal=$('#message'),title=$('#message-title'),body=$('#message-text'),detail=$('#error-detail'),raw=$('#error-raw'),cancel=$('#cancel'),confirm=$('#confirm');
let resolveDialog=null,returnFocus=null;
function finish(value){if(!resolveDialog)return;const done=resolveDialog;resolveDialog=null;if(modal.open)modal.close();done(value);if(returnFocus?.isConnected&&!returnFocus.disabled)returnFocus.focus();returnFocus=null}
cancel.addEventListener('click',()=>finish(false));
confirm.addEventListener('click',()=>finish(true));
modal.addEventListener('cancel',e=>{e.preventDefault();finish(false)});
export function ask({heading,text,action='关闭',confirming=false,danger=false,technical=''}) {
 if(resolveDialog)return Promise.resolve(false);
 returnFocus=document.activeElement;
 title.textContent=heading;body.textContent=text;detail.hidden=!technical;detail.open=false;raw.textContent=technical;
 cancel.hidden=!confirming;confirm.textContent=action;confirm.classList.toggle('danger',danger);modal.showModal();
 (confirming?cancel:confirm).focus();
 return new Promise(resolve=>{resolveDialog=resolve});
}
