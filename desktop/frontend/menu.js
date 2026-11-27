import {element,$} from './dialogs.js';
const menu=$('#app-menu');
let trigger=null,items=[];
export function closeMenu(restore=false) {
 if(menu.hidden)return;
 menu.hidden=true;menu.replaceChildren();trigger?.setAttribute('aria-expanded','false');
 if(restore&&trigger?.isConnected&&!trigger.disabled)trigger.focus();trigger=null;items=[];
}
export function openMenu(button,actions) {
 if(trigger===button&&!menu.hidden){closeMenu(true);return}
 closeMenu();trigger=button;button.setAttribute('aria-expanded','true');
 for(const a of actions){
  if(a.separator){const line=element('div','menu-separator');line.setAttribute('role','separator');menu.append(line);continue}
  const b=element('button','menu-item'+(a.danger?' destructive':''),a.label);b.type='button';b.setAttribute('role','menuitem');
  b.addEventListener('click',()=>{closeMenu(true);void a.run()});menu.append(b);items.push(b);
 }
 menu.hidden=false;const rect=button.getBoundingClientRect(),width=menu.offsetWidth,height=menu.offsetHeight;
 const below=innerHeight-rect.bottom-6;const top=below>=height?rect.bottom+4:Math.max(6,rect.top-height-4);
 menu.style.left=Math.max(6,Math.min(rect.right-width,innerWidth-width-6))+'px';
 menu.style.top=top+'px';items[0]?.focus();
}
menu.addEventListener('keydown',e=>{
 const i=items.indexOf(document.activeElement);
 if(e.key==='Escape'){e.preventDefault();closeMenu(true)}
 else if(e.key==='ArrowDown'||e.key==='ArrowUp'||e.key==='Home'||e.key==='End'){
  e.preventDefault();const n=e.key==='Home'?0:e.key==='End'?items.length-1:(i+(e.key==='ArrowDown'?1:-1)+items.length)%items.length;items[n]?.focus();
 }else if(e.key==='Tab')closeMenu(true);
});
document.addEventListener('pointerdown',e=>{if(!menu.hidden&&!menu.contains(e.target)&&!trigger?.contains(e.target))closeMenu()});
window.addEventListener('resize',()=>closeMenu());
document.addEventListener('scroll',()=>closeMenu(),true);
