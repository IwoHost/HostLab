"use strict";
// ┌─ dispatch picker ───────────────────────────────────────────
function renderPicker(){
  const wrap=document.getElementById("dispatchPicker");wrap.innerHTML="";
  const idle=G.ops.filter(o=>o.status==="idle");
  if(!idle.length){wrap.innerHTML="<div class='muted tiny'>No operatives available.</div>";return;}
  for(const o of G.ops){
    const usable=o.status==="idle";
    const slot=document.createElement("div");
    slot.className="slot"+(DS.sel.includes(o.id)?" sel":"")+(usable?"":" disabled");
    slot.innerHTML="<b>"+o.name+"</b><span class='muted'> G"+o.grit+(o.trait?" · "+TRAITS[o.trait].name:"")+"</span>";
    if(usable)slot.onclick=()=>{const i=DS.sel.indexOf(o.id);if(i>=0)DS.sel.splice(i,1);else DS.sel.push(o.id);renderPicker();updateDispatchCost();};
    wrap.appendChild(slot);
  }
}
// ┌─ DOM rendering ─────────────────────────────────────────────
function renderResbar(){
  const defs=[
    {k:"aw",  lab:"Almond Water",icon:"🥛",rate:-awDrainPerSec()},
    {k:"credits",lab:"Credits", icon:"¤",  rate:creditsPerSec()},
    {k:"salvage",lab:"Salvage", icon:"⚙",  rate:0},
    {k:"data",   lab:"Data",    icon:"▦",  rate:0},
  ];
  const bar=document.getElementById("resbar");
  let els=bar.querySelectorAll(".res");
  if(els.length!==defs.length){bar.innerHTML="";els=[];}
  defs.forEach((d,i)=>{
    const pct=G.cap[d.k]>0?Math.round(G.res[d.k]/G.cap[d.k]*100):0;
    let el=els[i];
    if(!el){el=document.createElement("div");el.className="res";bar.appendChild(el);}
    el.style.setProperty("--fill",pct+"%");
    el.className="res"+(pct<=10?" r-warn":pct>=95?" r-full":"");
    let rt="<div class='rate'>&nbsp;</div>";
    if(Math.abs(d.rate)>0.001){rt="<div class='rate' style='color:"+(d.rate>=0?"var(--green)":"var(--red)")+"'>"+(d.rate>=0?"+":"")+d.rate.toFixed(2)+"/s</div>";}
    el.innerHTML="<div class='label'>"+d.icon+" "+d.lab+"</div><div class='val'>"+fmt(G.res[d.k])+" <span class='muted tiny'>/ "+fmt(G.cap[d.k])+"</span></div>"+rt;
  });
  document.getElementById("clearance").textContent="CLEARANCE "+G.clearance+" · TIER "+highestTier();
}
function renderWarn(){
  const el=document.getElementById("warnbar");
  if(G.dryUntil){const left=Math.max(0,Math.ceil((G.dryUntil-Date.now())/1000));el.textContent="⚠ ALMOND WATER EMPTY — "+left+"s left. Extract a team with water or harvest some, fast.";el.classList.add("show");}
  else el.classList.remove("show");
}
let _rosterSig="";
function rosterSig(){return G.ops.map(o=>o.id+o.status+o.grit+o.xp+Math.round(o.cond)+Math.round(o.stress||0)+(o.trait||"")+(o.spec||"")).join("|")+G.ops.length+G.maxOps;}
function renderRoster(){
  _rosterSig=rosterSig();
  const box=document.getElementById("roster");box.innerHTML="";
  document.getElementById("rosterCnt").textContent=G.ops.length+"/"+G.maxOps;
  for(const o of G.ops){
    const div=document.createElement("div");div.className="op "+o.status;
    const secs=o.status==="injured"?Math.max(0,Math.ceil((o.recover-Date.now())/1000)):0;
    let sub=o.status==="deployed"?"In the field":(o.status==="injured"?"Recovering · "+secs+"s":"On site · ready");
    const barCls=o.cond<30?"low":o.cond<60?"mid":"";
    div.dataset.opid=o.id;
    div.innerHTML=
      "<div class='avatar'>"+(o.status==="injured"?"✚":o.name.replace(/^\(found\) /,"")[0].toUpperCase())+"</div>"+
      "<div class='meta'><div class='nm'>"+o.name+(o.trait?"<span class='tt'>"+TRAITS[o.trait].name+"</span>":"")+(o.spec?"<span class='tt' style='color:"+SPECS[o.spec].color+"'>"+SPECS[o.spec].icon+" "+SPECS[o.spec].name+"</span>":"")+"</div>"+
      "<div class='sub' data-sub='"+o.id+"'>"+sub+"</div>"+
      "<div class='bar "+barCls+"'><i style='width:"+o.cond+"%'></i></div>"+
      ((o.stress||0)>5?"<div class='bar' style='margin-top:2px'><i style='width:"+(o.stress||0)+"%;background:"+((o.stress||0)>75?"var(--red)":"var(--amber-dim)")+"'></i></div>":"")+"</div>"+
      "<div class='stat'>G "+o.grit+"<br><span class='muted' style='font-size:9.5px'>xp "+o.xp+"/"+(o.grit*12)+"</span></div>";
    div.onclick=()=>openOpDetail(o.id);
    box.appendChild(div);
  }
  const c=recruitCost();const btn=document.getElementById("hireBtn");
  if(G.ops.length>=G.maxOps){btn.textContent="At capacity ("+G.maxOps+")";btn.disabled=true;}
  else{btn.disabled=false;btn.innerHTML="Recruit operative <span class='muted tiny'>"+c.credits+"¤ "+c.aw+"🥛</span>";}
}
function tickRoster(){
  const sig=rosterSig();
  if(sig!==_rosterSig){renderRoster();return;}
  for(const o of G.ops){
    if(o.status==="injured"){
      const el=document.querySelector("[data-sub='"+o.id+"']");
      if(el)el.textContent="Recovering · "+Math.max(0,Math.ceil((o.recover-Date.now())/1000))+"s";
    }
  }
}
function renderFacility(){
  const box=document.getElementById("facility");box.innerHTML="";
  for(const f of FACILITY){
    const c=facCost(f);const aff=canAfford(c);
    const cs=Object.keys(c).map(k=>{const icon={credits:"¤",salvage:"⚙",data:"▦",aw:"🥛"}[k];return "<span class='"+(G.res[k]>=c[k]?"":"no")+"'>"+icon+" "+c[k]+"</span>";}).join("");
    const div=document.createElement("div");div.className="tech";
    div.innerHTML="<div class='tname'>"+f.name+" <span class='tag facility'>Lv "+G.fac[f.id]+"</span></div><div class='tdesc'>"+f.desc+"</div><div class='cost'>"+cs+"</div>";
    const b=document.createElement("button");b.className="btn small full";b.textContent="Upgrade";b.disabled=!aff;b.onclick=()=>buyFac(f);div.appendChild(b);
    box.appendChild(div);
  }
}
function renderActive(){
  const box=document.getElementById("active");
  document.getElementById("activeCnt").textContent=G.active.length?"("+G.active.length+")":"";
  if(!G.active.length){box.innerHTML="<div class='muted tiny'>No teams in the field.</div>";return;}
  box.innerHTML="";
  for(const e of G.active){
    const L=levelById(e.level);const left=Math.max(0,(e.end-Date.now())/1000);const pct=clamp(100*(1-left/e.dur),0,100);
    const names=e.ops.map(id=>{const o=opById(id);return o?o.name:"—";}).join(", ");
    const mins=Math.floor(left/60);const secs=Math.ceil(left%60);
    const eta=mins>0?mins+"m "+String(secs).padStart(2,"0")+"s":Math.ceil(left)+"s";
    const cond=getCondition(e.condition||"clear");
    const condLabel="<span class='cond' style='color:"+cond.color+";border-color:"+cond.color+"'>"+cond.label+"</span>";
    const div=document.createElement("div");
    div.className="exped"+(e.waiting?" waiting":"");
    div.innerHTML=
      "<div class='top'><span class='e-lvl'>"+levelTag(L)+" · "+L.name+condLabel+(e.waiting?" <span style='color:var(--red)'>[recovering]</span>":(e.auto?" <span class='muted'>[auto]</span>":""))+"</span>"+
      "<span class='e-eta'><span class='blink'>●</span> "+eta+"</span></div>"+
      "<div class='pbar'><i style='width:"+pct+"%'></i></div>"+
      "<div class='team'>"+names+"</div>"+
      "<div class='bottom'><span></span><button class='recall-btn' data-eid='"+e.id+"'>[ recall ]</button></div>";
    box.appendChild(div);
  }
  box.querySelectorAll(".recall-btn").forEach(btn=>{
    btn.onclick=()=>recallExpedition(parseInt(btn.dataset.eid));
  });
}
let openLevel=-1;
function renderLevels(){
  const box=document.getElementById("levels");box.innerHTML="";
  const ordered=[...LEVELS].sort((a,b)=>a.unlock-b.unlock||a.id-b.id);
  for(const L of ordered){
    const unlocked=levelUnlocked(L);
    const div=document.createElement("div");div.className="level"+(unlocked?"":" locked")+(openLevel===L.id?" open":"");
    const dTxt=L.danger<0.06?"minimal":L.danger<0.16?"low":L.danger<0.32?"moderate":L.danger<0.5?"high":"extreme";
    const dCls=L.danger<0.1?"lowd":L.danger<0.32?"modd":"";
    const head=document.createElement("div");head.className="lhead";
    head.innerHTML="<span class='num'>"+levelTag(L).replace("LEVEL ","L").replace("LVL ","L:")+"</span>"+
      "<span class='lname'>"+L.name+"</span>"+
      "<span class='danger "+dCls+"'>"+(unlocked?dTxt:"locked")+"</span>";
    head.onclick=()=>{openLevel=(openLevel===L.id?-1:L.id);renderLevels();};
    div.appendChild(head);
    const body=document.createElement("div");body.className="body";
    let lt=[];for(const k in L.loot){const icon={aw:"🥛",credits:"¤",salvage:"⚙",data:"▦"}[k];lt.push(icon+" "+L.loot[k][0]+"–"+L.loot[k][1]);}
    const dur=Math.round(L.time*speedMult());
    if(!unlocked){
      let req="";
      if(G.clearance<L.unlock){
        const pct=Math.min(100,Math.round(G.clearance/L.unlock*100));
        const filled=Math.round(pct/100*12);
        const barText="["+("█".repeat(filled)+"░".repeat(12-filled))+"] "+G.clearance+"/"+L.unlock+" clearance";
        req="<div class='tiny' style='color:var(--red);margin-bottom:4px'>Clearance required: "+L.unlock+"</div><div class='clr-prog'><i style='width:"+pct+"%'></i></div><div class='tiny muted' style='margin-bottom:8px;letter-spacing:.04em'>"+barText+"</div>";
      }else if((L.id===5||L.id===6||L.id===8||L.id===13||L.id===14||L.id===15||L.id===19||L.id===20)&&!has("cart2")){
        req="<div class='tiny' style='color:var(--red);margin-bottom:8px'>Needs Cartography II.</div>";
      }else if((L.id===3||L.id===4||L.id===9||L.id===10||L.id===11||L.id===12||L.id===16||L.id===17||L.id===18)&&!has("cart1")){
        req="<div class='tiny' style='color:var(--red);margin-bottom:8px'>Needs Cartography I.</div>";
      }
      body.innerHTML=
        "<div class='flav'>"+L.flav+"</div>"+
        "<div class='lvl-stats'><div>Duration <span>~"+dur+"s</span></div><div>Danger <span style='color:var(--red)'>"+Math.round(L.danger*100)+"%</span></div></div>"+
        "<div class='loot-line'>Yield: <b>"+lt.join("  ")+"</b></div>"+
        req;
    }else{
      const idleCount=G.ops.filter(o=>o.status==="idle").length;
      const qCost=idleCount>0?supplyCost(L,idleCount):0;
      body.innerHTML=
        "<div class='flav'>"+L.flav+"</div>"+
        "<div class='lvl-stats'><div>Duration <span>~"+dur+"s</span></div><div>Danger <span style='color:var(--red)'>"+Math.round(L.danger*100)+"%</span></div></div>"+
        "<div class='loot-line'>Yield: <b>"+lt.join("  ")+"</b></div>";
      const b=document.createElement("button");b.className="btn primary full";b.textContent="Dispatch team →";
      b.onclick=e=>{e.stopPropagation();openDispatch(L.id);};body.appendChild(b);
      if(idleCount>0){
        const qb=document.createElement("button");
        qb.className="btn full";
        qb.style.marginTop="5px";
        qb.textContent="Quick send — all "+idleCount+" idle →  ("+qCost+"🥛)";
        qb.onclick=e=>{e.stopPropagation();quickDispatch(L.id);};
        body.appendChild(qb);
      }
    }
    div.appendChild(body);box.appendChild(div);
  }
}
function tickResearch(){
  if(!G.researchTarget)return;
  const pct=Math.min(100,Math.round(G.researchProgress));
  const rate=researchRate();
  const eta=rate>0?Math.ceil((100-G.researchProgress)/rate)+"s":"⏸ idle op needed";
  const lbl=pct+"% — "+eta+" remaining · "+rate.toFixed(1)+" %/s";
  const f1=document.getElementById("rpBarTopFill");
  const l1=document.getElementById("rpLblTop");
  if(f1)f1.style.width=pct+"%";
  if(l1){const btn=l1.querySelector("button");l1.textContent=lbl;if(btn)l1.appendChild(btn);}
  const f2=document.getElementById("rpBarCardFill");
  const l2=document.getElementById("rpLblCard");
  if(f2)f2.style.width=pct+"%";
  if(l2)l2.textContent="In progress · "+pct+"%";
}
function renderResearch(){
  const box=document.getElementById("research");box.innerHTML="";
  // Research rate info bar
  if(G.researchTarget){
    const rate=researchRate();
    const pct=Math.min(100,Math.round(G.researchProgress));
    const t=TECH.find(x=>x.id===G.researchTarget);
    const eta=rate>0?Math.ceil((100-G.researchProgress)/rate)+"s":"⏸ idle op needed";
    const bar=document.createElement("div");bar.className="tech in-progress";
    bar.style.cssText="border-color:var(--blue-dim);margin-bottom:12px;padding:8px 11px";
    bar.innerHTML="<div class='tname' style='font-size:11px;color:var(--blue)'>▶ "+t.name+"</div>"+
      "<div class='res-prog' id='rpBarTop'><i id='rpBarTopFill' style='width:"+pct+"%'></i></div>"+
      "<div class='res-prog-lbl' id='rpLblTop'>"+pct+"% — "+eta+" remaining · "+rate.toFixed(1)+" %/s"+
      " <button class='btn small' style='float:right;font-size:9px;padding:1px 6px;margin-top:-2px' id='cancelResBtn'>cancel</button></div>";
    box.appendChild(bar);
    document.getElementById("cancelResBtn").onclick=()=>{
      const t2=TECH.find(x=>x.id===G.researchTarget);
      if(t2)for(const k in t2.cost)G.res[k]=(G.res[k]||0)+t2.cost[k];
      G.researchTarget=null;G.researchProgress=0;
      log("Research cancelled. Resources refunded.","report");renderResearch();
    };
  }
  for(const t of TECH){
    const bought=!!G.tech[t.id];
    const inProgress=G.researchTarget===t.id;
    const reqMet=t.req.every(r=>G.tech[r]);
    const aff=canAfford(t.cost);
    let cls="tech";
    if(bought)cls+=" bought";
    else if(inProgress)cls+=" in-progress";
    else if(!reqMet)cls+=" locked-dep";
    else if(aff)cls+=" can-buy";
    const div=document.createElement("div");div.className=cls;
    div.innerHTML="<div class='tname'>"+t.name+" <span class='tag "+t.tag+"'>"+t.tag+"</span></div><div class='tdesc'>"+t.desc+"</div>";
    if(bought){
      div.innerHTML+="<div class='tiny' style='color:var(--green)'>✓ Operational</div>";
    } else if(inProgress){
      const pct=Math.min(100,Math.round(G.researchProgress));
      div.innerHTML+="<div class='res-prog' id='rpBarCard'><i id='rpBarCardFill' style='width:"+pct+"%'></i></div>"+
        "<div class='res-prog-lbl' id='rpLblCard'>In progress · "+pct+"%</div>";
    } else if(!reqMet){
      const reqHtml=t.req.map(r=>{
        const done=!!G.tech[r];const rname=TECH.find(x=>x.id===r).name;
        return "<span style='color:"+(done?"var(--green)":"var(--red)")+"'>"+(done?"✓ ":"✗ ")+rname+"</span>";
      }).join("<span style='color:var(--ink-dim)'>&nbsp;·&nbsp;</span>");
      div.innerHTML+="<div style='font-size:10px;margin-top:4px;letter-spacing:.03em'>Unlock first: "+reqHtml+"</div>";
    } else {
      const cs=Object.keys(t.cost).map(k=>{
        const icon={credits:"¤",salvage:"⚙",data:"▦"}[k];
        const have=G.res[k],need=t.cost[k],ok=have>=need;
        return "<span class='"+(ok?"ok":"no")+"'>"+icon+" "+need+(ok?"":"<span style='font-size:9px;opacity:.7'>/"+fmt(have)+"</span>")+"</span>";
      }).join("");
      div.innerHTML+="<div class='cost'>"+cs+"</div>";
      if(!aff){
        const short=Object.keys(t.cost).filter(k=>G.res[k]<t.cost[k]).map(k=>{
          const icon={credits:"¤",salvage:"⚙",data:"▦"}[k];
          return icon+" "+(t.cost[k]-Math.floor(G.res[k]))+" more";
        }).join("  ·  ");
        div.innerHTML+="<div style='font-size:9.5px;color:var(--red);margin-top:3px'>need: "+short+"</div>";
      }
      const isBlocked=!!G.researchTarget;
      const b=document.createElement("button");b.className="btn small full";
      b.textContent=aff?(isBlocked?"Queue (busy)":"Research ↗"):"Research";
      b.disabled=!aff||isBlocked;b.onclick=()=>buyTech(t);
      div.appendChild(b);
    }
    box.appendChild(div);
  }
}
function renderFeed(){
  const box=document.getElementById("feed");
  let tutorialEntry=null;
  if(G.tutorialStep<3){
    let msg="";
    if(G.tutorialStep===0&&G.runs===0)msg="Step 1 — Expand LEVEL 0 in the Levels column and click Dispatch to send your first team.";
    else if(G.tutorialStep===1&&G.runs===0&&G.active.length>0)msg="Step 2 — Watch the expedition complete. Your team will haul back almond water and other resources.";
    else if(G.tutorialStep===2&&G.active.length===0&&G.runs>0)msg="Step 3 — Use credits and salvage in Research to build Hydroponics Bay for a +25% AW expedition yield boost.";
    if(msg)tutorialEntry={stamp:"--:--:--",msg,cls:"guide",fresh:false};
  }
  let entries=feedFilter?G.feed.filter(e=>e.cls===feedFilter):G.feed;
  let h="";
  if(tutorialEntry&&!feedFilter){
    h+="<div class='e guide'><span class='t'>"+tutorialEntry.stamp+"</span> <span class='msg'>"+tutorialEntry.msg+"</span></div>";
  }
  for(let i=0;i<entries.length;i++){
    const e=entries[i];
    h+="<div class='e "+e.cls+(i===0&&e.fresh?" new-entry":"")+"'><span class='t'>"+e.stamp+"</span> <span class='msg'>"+e.msg+"</span></div>";
  }
  box.innerHTML=h;
  if(G.feed.length>0)G.feed[0].fresh=false;
}
function renderStats(){
  const ag=G.totalGained||{};const pt=Math.round((G.playTime||0)/60);
  document.getElementById("statsGrid").innerHTML=`
    <div class="stat-card"><div class="slabel">Expeditions run</div><div class="sval">${G.runs}</div></div>
    <div class="stat-card"><div class="slabel">Operatives lost</div><div class="sval">${G.lost}</div></div>
    <div class="stat-card"><div class="slabel">Clearance score</div><div class="sval">${G.clearance}</div></div>
    <div class="stat-card"><div class="slabel">Play time</div><div class="sval">${pt}m</div></div>
    <div class="stat-card"><div class="slabel">AW hauled total</div><div class="sval">${fmt(ag.aw||0)}</div></div>
    <div class="stat-card"><div class="slabel">Credits earned</div><div class="sval">${fmt(ag.credits||0)}</div></div>
    <div class="stat-card"><div class="slabel">Salvage recovered</div><div class="sval">${fmt(ag.salvage||0)}</div></div>
    <div class="stat-card"><div class="slabel">Data recovered</div><div class="sval">${fmt(ag.data||0)}</div></div>`;
  const achUnlocked=ACHIEVEMENTS.filter(a=>G.achievements[a.id]).length;
  document.getElementById("achCnt").textContent=achUnlocked+"/"+ACHIEVEMENTS.length;
  const alist=document.getElementById("achieveList");alist.innerHTML="";
  for(const a of ACHIEVEMENTS){
    const done=!!G.achievements[a.id];
    const d=document.createElement("div");d.className="ach"+(done?" unlocked":"");
    d.innerHTML="<div class='aname'>"+(done?"❆ ":"")+a.name+"</div><div class='adesc'>"+a.desc+"</div>";
    alist.appendChild(d);
  }
  // Memorial section
  const memorial=G.memorial||[];
  const mlist=document.getElementById("memorialList");
  if(!memorial.length){mlist.innerHTML="<div class='tiny muted'>No operatives lost.</div>";return;}
  let mh="<div class='memorial-grid'>";
  for(const m of memorial){
    const traitStr=m.trait?(", "+TRAITS[m.trait].name):"";
    mh+="<div class='memorial-card'><div class='mname'>"+m.name+"</div><div class='mdetail'>Lost in "+m.level+"</div><div class='mdetail'>Grit "+m.grit+traitStr+" · "+m.runs+" run"+(m.runs!==1?"s":"")+"</div><div class='mdetail' style='color:var(--ink-dim);font-size:9.5px'>"+m.when+"</div></div>";
  }
  mh+="</div>";
  mlist.innerHTML=mh;
}
function openOpDetail(id){
  const o=opById(id);if(!o)return;
  const st=o.status==="idle"?"On site · ready":(o.status==="deployed"?"In the field":("Recovering · "+Math.max(0,Math.ceil((o.recover-Date.now())/1000))+"s"));
  document.getElementById("opDetailTitle").innerHTML=o.name+
    " <button class='btn small' id='opDetailClose' style='font-size:10px;padding:2px 8px'>CLOSE</button>";
  document.getElementById("opDetailBody").innerHTML=
    "<div class='mflav'>"+(o.trait?TRAITS[o.trait].name+" — "+TRAITS[o.trait].desc:"No special trait.")+"</div>"+
    "<div class='op-dstat'>"+
      "<div class='ds'>Status <b>"+st+"</b></div>"+
      "<div class='ds'>Grit <b>"+o.grit+"</b></div>"+
      "<div class='ds'>XP <b>"+o.xp+"/"+(o.grit*12)+"</b></div>"+
      "<div class='ds'>Condition <b>"+o.cond+"%</b></div>"+
      "<div class='ds'>Runs <b>"+(o.runs||0)+"</b></div>"+
      "<div class='ds'>Spec <b>"+(o.spec?SPECS[o.spec].icon+" "+SPECS[o.spec].name:"none")+"</b></div>"+
    "</div>"+
    "<div class='bar "+(o.cond<30?"low":o.cond<60?"mid":"")+"' style='height:6px;margin:8px 0 16px'><i style='width:"+o.cond+"%'></i></div>"+
    "<div class='tiny muted'>Click outside to close.</div>";
  document.getElementById("opDetailClose").onclick=()=>document.getElementById("opDetailBg").classList.remove("show");
  document.getElementById("opDetailBg").classList.add("show");
}
/* ---- contracts ---- */
function genContract(){
  const tier=Math.max(0,Math.floor((G.runs||0)/8));
  const templates=[
    {title:"Field Exercise",  type:"runs",    make:()=>{const g=rand(2,3+Math.floor(tier/2));return{goal:g,desc:"Complete "+g+" expedition"+(g>1?"s":"")+"."};},              reward:()=>({credits:22+tier*10,salvage:4+tier*3})},
    {title:"Water Detail",   type:"aw",      make:()=>{const g=rand(18,35+tier*14);return{goal:g,desc:"Haul "+g+" almond water from the field."};},                        reward:()=>({credits:18+tier*8,data:2+tier*2})},
    {title:"Credit Run",     type:"credits", make:()=>{const g=rand(28,50+tier*18);return{goal:g,desc:"Earn "+g+" credits from expeditions."};},                           reward:()=>({salvage:10+tier*6,data:2+tier*2})},
    {title:"Clean Operation",type:"clean",   make:()=>{const g=rand(1,1+Math.floor(tier/3));return{goal:g,desc:"Complete "+g+" expedition"+(g>1?"s":"")+" without injuries."};},reward:()=>({aw:18+tier*8,credits:22+tier*10})},
    {title:"Data Harvest",   type:"data",    make:()=>{const g=rand(4,9+tier*3);return{goal:g,desc:"Recover "+g+" data from the field."};},                                reward:()=>({credits:28+tier*12,salvage:8+tier*5})},
    {title:"Salvage Drive",  type:"salvage", make:()=>{const g=rand(10,22+tier*10);return{goal:g,desc:"Recover "+g+" salvage."};},                                         reward:()=>({aw:12+tier*7,data:2+tier*3})},
  ];
  const t=pick(templates);const m=t.make();
  return{id:(G.expSeq||0)+Math.floor(Math.random()*9999),title:t.title,type:t.type,desc:m.desc,goal:m.goal,progress:0,reward:t.reward(),expires:Date.now()+200000};
}
function renderContracts(){
  if(!G.contracts)G.contracts=[];
  while(G.contracts.length<2)G.contracts.push(genContract());
  const now=Date.now();
  for(let i=0;i<G.contracts.length;i++){
    const c=G.contracts[i];
    if(c.progress>=c.goal){
      for(const k in c.reward)gain(k,c.reward[k]);
      const rStr=Object.entries(c.reward).map(([k,v])=>"+"+v+({credits:"¤",salvage:"⚙",data:"▦",aw:"🥛"}[k]||k)).join(" ");
      log('Contract complete: "'+c.title+'". Reward: '+rStr+'.','good');
      sfxUpgrade();
      G.contracts[i]=genContract();
    }else if(now>c.expires){
      log('Contract expired: "'+c.title+'".','report');
      G.contracts[i]=genContract();
    }
  }
  const box=document.getElementById("contracts");if(!box)return;
  const active=G.contracts.filter(c=>c.progress<c.goal).length;
  document.getElementById("contractCnt").textContent="("+active+")";
  box.innerHTML="";
  for(const c of G.contracts){
    const done=c.progress>=c.goal;
    const pct=Math.min(100,c.goal>0?Math.round(c.progress/c.goal*100):0);
    const left=Math.max(0,Math.ceil((c.expires-now)/1000));
    const mins=Math.floor(left/60);const secs=left%60;
    const eta=mins>0?mins+"m "+String(secs).padStart(2,"0")+"s":left+"s";
    const rStr=Object.entries(c.reward).map(([k,v])=>"+"+v+({credits:"¤",salvage:"⚙",data:"▦",aw:"🥛"}[k]||k)).join(" ");
    const div=document.createElement("div");div.className="contract"+(done?" done":"");
    div.innerHTML=`<div class="ct-top"><span class="ct-title">${c.title}</span><span class="ct-eta">${done?"✓ done":eta}</span></div>`+
      `<div class="ct-desc">${c.desc}</div>`+
      `<div class="ct-row"><div class="ct-prog"><i style="width:${pct}%"></i></div><span class="ct-pct">${c.progress}/${c.goal}</span></div>`+
      `<div class="ct-reward">${rStr}</div>`;
    box.appendChild(div);
  }
}
function trackContracts(gained,hadInjury){
  if(!G.contracts)return;
  if(hadInjury)G.cleanStreak=0;else G.cleanStreak=(G.cleanStreak||0)+1;
  for(const c of G.contracts){
    if(c.progress>=c.goal)continue;
    if(c.type==="runs")c.progress=Math.min(c.goal,c.progress+1);
    if(c.type==="aw"&&gained.aw)c.progress=Math.min(c.goal,c.progress+(gained.aw||0));
    if(c.type==="credits"&&gained.credits)c.progress=Math.min(c.goal,c.progress+(gained.credits||0));
    if(c.type==="salvage"&&gained.salvage)c.progress=Math.min(c.goal,c.progress+(gained.salvage||0));
    if(c.type==="data"&&gained.data)c.progress=Math.min(c.goal,c.progress+(gained.data||0));
    if(c.type==="clean"){if(hadInjury)c.progress=0;else c.progress=Math.min(c.goal,c.progress+1);}
  }
}

/* ---- specializations ---- */
function openSpecModal(opId){
  const o=opById(opId);if(!o||o.spec)return;
  document.getElementById("specTitle").textContent="Field Promotion — "+o.name;
  document.getElementById("specFlav").textContent="After "+(o.runs||0)+" runs at Grit "+o.grit+", "+o.name+" has earned a specialization.";
  const box=document.getElementById("specChoices");box.innerHTML="";
  for(const[key,sp]of Object.entries(SPECS)){
    const b=document.createElement("button");b.className="choice";
    b.innerHTML=`<span style="color:${sp.color}">${sp.icon} ${sp.name}</span><span class="rk">${sp.desc}</span>`;
    b.onclick=()=>{o.spec=key;log(o.name+" specialised as "+sp.name+".","good");document.getElementById("specBg").classList.remove("show");sfxAchieve();renderAll();};
    box.appendChild(b);
  }
  document.getElementById("specBg").classList.add("show");
}

/* ---- lore system ---- */
function openLore(id,isNew){
  const doc=LORE.find(l=>l.id===id);if(!doc)return;
  document.getElementById("lorePopupTitle").textContent="RECOVERED: "+doc.title;
  document.getElementById("lorePopupBody").textContent=doc.body;
  if(isNew){log("Field team found a document: \""+doc.title+"\". Check the archive.","report");sfxAchieve();}
  document.getElementById("loreBg").classList.add("show");
}
function closeLorePopup(){document.getElementById("loreBg").classList.remove("show");}
function openLoreArchive(){
  closeLorePopup();
  renderLoreArchive();
  document.getElementById("loreArchiveBg").classList.add("show");
}
function renderLoreArchive(){
  const list=document.getElementById("loreList");
  const reader=document.getElementById("loreReader");
  const discovered=G.discoveredLore||[];
  document.getElementById("loreCntBadge").textContent="("+discovered.length+"/"+LORE.length+" recovered)";
  list.innerHTML="";
  if(!discovered.length){
    list.innerHTML="<div class='tiny muted' style='padding:4px'>No documents recovered yet. Complete expeditions to find them.</div>";
    reader.textContent="";return;
  }
  let first=true;
  for(const id of discovered){
    const doc=LORE.find(l=>l.id===id);if(!doc)continue;
    const btn=document.createElement("button");
    btn.className="btn small full";
    btn.style.cssText="text-align:left;font-size:10px;margin-bottom:4px;padding:5px 8px;white-space:normal;line-height:1.3";
    btn.textContent=doc.title;
    btn.onclick=()=>{reader.textContent=doc.body;list.querySelectorAll(".btn").forEach(b=>b.classList.remove("active"));btn.classList.add("active");};
    list.appendChild(btn);
    if(first){reader.textContent=doc.body;btn.classList.add("active");first=false;}
  }
}

// ┌─ anomaly / memorial / renderAll ────────────────────────────
function renderAnomalies(){
  let box=document.getElementById("anomalies");if(!box)return;
  box.innerHTML="";
  const cnt=document.getElementById("anomalyCnt");
  if(cnt)cnt.textContent=(G.anomalies&&G.anomalies.length)?"("+G.anomalies.length+")":"";
  if(!G.anomalies||!G.anomalies.length){box.innerHTML="<div class='muted tiny'>No anomalies in containment.</div>";return;}
  const idleCount=G.ops.filter(o=>o.status==="idle").length;
  const rate=idleCount>0?idleCount*0.4:0;
  for(const a of G.anomalies){
    const pct=Math.min(100,Math.round(a.progress));
    const eta=rate>0?Math.ceil((100-a.progress)/rate)+"s":"⏸";
    const rStr=Object.entries(a.reward).map(([k,v])=>"+"+v+({credits:"¤",data:"▦"}[k]||k)).join(" ");
    const div=document.createElement("div");
    div.className="tech";div.style.cssText="border-color:var(--amber-dim);margin-bottom:8px";
    div.innerHTML=`<div class='tname' style='font-size:11.5px'>${a.name} <span class='tag unlock' style='font-size:8px'>ANOMALY</span></div>`+
      `<div class='tdesc' style='font-size:10.5px'>From: ${levelById(a.levelId).name} · Risk: ${Math.round(a.risk*100)}%</div>`+
      `<div class='res-prog'><i style='width:${pct}%;background:var(--amber-dim)'></i></div>`+
      `<div class='res-prog-lbl'>${pct}% — ${eta} · reward: ${rStr}</div>`;
    box.appendChild(div);
  }
}

function renderMemorial(){
  const box=document.getElementById("memorial");if(!box)return;
  const cnt=document.getElementById("memCnt");
  const mem=G.memorial||[];
  if(cnt)cnt.textContent=mem.length?"("+mem.length+")":"";
  if(!mem.length){box.innerHTML="<div class='muted tiny'>No one has been lost.</div>";return;}
  box.innerHTML="<div class='memorial-grid'></div>";
  const grid=box.querySelector(".memorial-grid");
  for(const m of [...mem].reverse()){
    const trait=m.trait?TRAITS[m.trait]?.name:"—";
    const spec=m.spec?SPECS[m.spec]?.name:"—";
    const div=document.createElement("div");div.className="memorial-card";
    div.innerHTML=
      "<div class='mname'>"+m.name+"</div>"+
      "<div class='mdetail'>Grit "+m.grit+" · "+m.runs+" mission"+(m.runs!==1?"s":"")+"</div>"+
      "<div class='mdetail'>Lost in: "+m.level+"</div>"+
      (m.cause?"<div class='mdetail' style='color:#6e3020'>"+m.cause+"</div>":"")+
      (m.trait?"<div class='mdetail'>"+trait+"</div>":"")+
      "<div class='mdetail' style='opacity:.5;font-size:9.5px'>"+m.when+"</div>";
    grid.appendChild(div);
  }
}
function checkFragmentUnlock(){
  const eligible=G.endCleared&&TECH.every(t=>G.tech[t.id]);
  const btn=document.getElementById("newFragBtn");
  if(btn)btn.style.display=eligible?"inline-block":"none";
}

function fragmentRoman(n){const v=[10,9,5,4,1],s=["X","IX","V","IV","I"];let r="";for(let i=0;i<v.length;i++)while(n>=v[i]){r+=s[i];n-=v[i];}return r;}

function renderAll(){renderResbar();renderWarn();renderRoster();renderFacility();renderActive();renderContracts();renderLevels();renderResearch();renderAnomalies();renderMemorial();renderFeed();updateSuggestBar();checkFragmentUnlock();
  const fd=document.getElementById("fragDepth");
  if(fd)fd.textContent=FRAG.depth>0?"Fragment "+fragmentRoman(FRAG.depth):"";
}

/* ---- save / load ---- */