/* Shared offline controls. Native selects remain the model and emit normal change events. */
(()=>{
 if(window.RoutiletUI)return;
 let popup=null,seq=0;
 const views=new Map();
 function icon(name){const svg=document.createElementNS("http://www.w3.org/2000/svg","svg");svg.setAttribute("class","ui-icon");svg.setAttribute("aria-hidden","true");const path=document.createElementNS(svg.namespaceURI,"path");path.setAttribute("d",name==="check"?"m5 12 4 4L19 6":"m6 9 6 6 6-6");svg.append(path);return svg}
 function position(){
  if(!popup)return;const {anchor,panel}=popup;
  if(!anchor.isConnected||!anchor.getClientRects().length||anchor.disabled){close(false);return}
  const r=anchor.getBoundingClientRect(),margin=8,below=innerHeight-r.bottom-margin,above=r.top-margin;
  panel.style.minWidth=Math.min(r.width,innerWidth-2*margin)+"px";
  panel.style.maxWidth=Math.max(0,innerWidth-2*margin)+"px";
  panel.style.maxHeight=Math.max(32,Math.min(320,Math.max(below,above)-4))+"px";
  const w=panel.offsetWidth,h=panel.offsetHeight;
  panel.style.left=Math.max(margin,Math.min(r.left,innerWidth-w-margin))+"px";
  panel.style.top=Math.max(margin,Math.min(below>=h+4||below>=above?r.bottom+4:r.top-h-4,innerHeight-h-margin))+"px";
 }
 function close(restore=false){
  if(!popup)return;const p=popup;popup=null;p.anchor.setAttribute("aria-expanded","false");p.panel.hidden=true;p.onClose?.();if(restore&&p.anchor.isConnected&&!p.anchor.disabled)p.anchor.focus({preventScroll:true});
 }
 function open(anchor,panel,onClose){
  if(popup?.anchor===anchor){close(true);return false}close(false);
  popup={anchor,panel,onClose};panel.hidden=false;anchor.setAttribute("aria-expanded","true");position();return true;
 }
 document.addEventListener("pointerdown",e=>{if(popup&&!popup.anchor.contains(e.target)&&!popup.panel.contains(e.target))close(false)},true);
 document.addEventListener("focusin",e=>{if(popup&&!popup.anchor.contains(e.target)&&!popup.panel.contains(e.target))close(false)},true);
 document.addEventListener("keydown",e=>{if(!popup)return;if(e.key==="Escape"){e.preventDefault();e.stopImmediatePropagation();close(true)}},true);
 window.addEventListener("resize",position);document.addEventListener("scroll",e=>{if(popup&&!popup.panel.contains(e.target))position()},true);
 function enhance(select){
  if(views.has(select)||select.multiple||select.size>1)return;
  const trigger=document.createElement("button"),label=document.createElement("span"),panel=document.createElement("div");
  trigger.type="button";trigger.className="ui-select ui-select-trigger";trigger.setAttribute("role","combobox");trigger.setAttribute("aria-haspopup","listbox");trigger.setAttribute("aria-expanded","false");
  trigger.dataset.selectFor=select.id||"";label.className="ui-select-value";trigger.append(label,icon("chevron"));
  panel.id="ui-options-"+(++seq);panel.className="ui-popover ui-options";panel.hidden=true;panel.setAttribute("role","listbox");trigger.setAttribute("aria-controls",panel.id);
  select.after(trigger);select.classList.add("ui-select-model");select.tabIndex=-1;select.setAttribute("aria-hidden","true");(select.closest("dialog")||document.body).append(panel);
  let active=-1,signature="",choices=[],type="",lastType=0;
  const options=()=>Array.from(select.options);
  function sync(){
   trigger.hidden=select.hidden;trigger.disabled=select.disabled;const o=select.selectedOptions[0];const text=o?.label||"请选择";
   if(label.textContent!==text)label.textContent=text;
   const names=select.getAttribute("aria-label")||Array.from(select.labels||[]).map(x=>x.textContent.trim()).join(" ")||"选择";
   trigger.setAttribute("aria-label",names);panel.setAttribute("aria-label",names);
   if(popup?.anchor===trigger){if(trigger.hidden||trigger.disabled||!select.getClientRects().length){close(false);return}build(false)}
  }
  function activate(i){active=i;for(let n=0;n<choices.length;n++)choices[n].classList.toggle("ui-option-active",n===i);if(i>=0){trigger.setAttribute("aria-activedescendant",choices[i].id);choices[i].scrollIntoView({block:"nearest"})}else trigger.removeAttribute("aria-activedescendant")}
  function choose(i){const o=options()[i];if(!o||o.disabled||o.hidden||o.parentElement?.disabled)return;select.selectedIndex=i;select.dispatchEvent(new Event("input",{bubbles:true}));select.dispatchEvent(new Event("change",{bubbles:true}));sync();close(true)}
  function usable(o){return !o.disabled&&!o.hidden&&!o.parentElement?.disabled}
  function build(force){
   const os=options(),key=JSON.stringify(os.map(o=>[o.value,o.label,o.selected,o.disabled,o.hidden,!!o.parentElement?.disabled]));
   if(!force&&key===signature)return;signature=key;choices=[];panel.replaceChildren();
   os.forEach((o,i)=>{const b=document.createElement("div");b.id=panel.id+"-"+i;b.className="ui-option";b.setAttribute("role","option");b.setAttribute("aria-selected",String(o.selected));b.setAttribute("aria-disabled",String(!usable(o)));b.hidden=o.hidden||(!o.value&&o.disabled);const t=document.createElement("span");t.textContent=o.label;b.append(t);if(o.selected)b.append(icon("check"));b.addEventListener("pointerdown",e=>e.preventDefault());b.onclick=()=>choose(i);choices.push(b);panel.append(b)});
   const i=usable(os[active]||{disabled:true})?active:os.findIndex(o=>o.selected&&usable(o));activate(i>=0?i:os.findIndex(usable));
  }
  function show(){sync();if(trigger.disabled)return;active=select.selectedIndex;build(true);if(open(trigger,panel,()=>trigger.removeAttribute("aria-activedescendant")))activate(active)}
  function step(dir){const os=options();let n=active;for(let i=0;i<os.length;i++){n=(n+dir+os.length)%os.length;if(usable(os[n])){activate(n);return}}}
  trigger.onclick=show;
  trigger.onkeydown=e=>{
   const opened=popup?.anchor===trigger;
   if(e.key==="Tab"){if(opened)close(false);return}
   if(["ArrowDown","ArrowUp","Home","End","Enter"," "].includes(e.key)){e.preventDefault();e.stopPropagation();if(!opened){show();return}if(e.key==="Enter"||e.key===" "){choose(active);return}if(e.key==="Home"||e.key==="End"){const os=options();activate(e.key==="Home"?os.findIndex(usable):os.map(usable).lastIndexOf(true));return}step(e.key==="ArrowDown"?1:-1);return}
   if(e.key.length===1&&!e.ctrlKey&&!e.metaKey&&!e.altKey){e.preventDefault();if(!opened)show();const now=Date.now();type=(now-lastType>700?"":type)+e.key.toLocaleLowerCase();lastType=now;const i=options().findIndex(o=>usable(o)&&o.label.toLocaleLowerCase().startsWith(type));if(i>=0)activate(i)}
  };
  select.addEventListener("change",sync);select.addEventListener("input",sync);select.addEventListener("focus",()=>trigger.focus());views.set(select,{trigger,panel,sync});sync();
 }
 function scan(){
  document.querySelectorAll("select.ui-select:not(.ui-select-model)").forEach(enhance);
  for(const [select,v] of views){if(!select.isConnected){if(popup?.anchor===v.trigger)close(false);v.trigger.remove();v.panel.remove();views.delete(select)}else v.sync()}
 }
 function start(){scan();let queued=false;new MutationObserver(records=>{if(!records.some(r=>r.target instanceof HTMLSelectElement||r.target instanceof HTMLOptionElement||[...r.addedNodes,...r.removedNodes].some(n=>n instanceof Element&&(n.matches("select,option")||n.querySelector("select")))))return;if(!queued){queued=true;queueMicrotask(()=>{queued=false;scan()})}}).observe(document.body,{childList:true,subtree:true,attributes:true,attributeFilter:["disabled","hidden","selected","label","value"]});setInterval(scan,120)}
 window.RoutiletUI={open,close,position};
 if(document.readyState==="loading")document.addEventListener("DOMContentLoaded",start,{once:true});else start();
})();
