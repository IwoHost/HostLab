"use strict";
// ┌─ audio ─────────────────────────────────────────────────────

let SFX=false,_ac=null;
function getAC(){if(!_ac)_ac=new(window.AudioContext||window.webkitAudioContext)();return _ac;}
function tone(freq,dur,type,vol){
  try{
    const ctx=getAC();
    if(ctx.state==="suspended")ctx.resume();
    const o=ctx.createOscillator(),g=ctx.createGain();
    o.connect(g);g.connect(ctx.destination);
    o.type=type||"sine";o.frequency.value=freq;
    g.gain.setValueAtTime(vol||0.07,ctx.currentTime);
    g.gain.exponentialRampToValueAtTime(0.001,ctx.currentTime+dur);
    o.start();o.stop(ctx.currentTime+dur);
  }catch(e){}
}
function sfxDispatch(){if(!SFX)return;tone(440,.07,"square",.04);}
function sfxReturn(){if(!SFX)return;tone(330,.12,"sine",.06);setTimeout(()=>tone(500,.1,"sine",.05),140);}
function sfxBad(){if(!SFX)return;tone(180,.35,"sawtooth",.05);}
function sfxAchieve(){if(!SFX)return;[440,554,659].forEach((f,i)=>setTimeout(()=>tone(f,.18,"sine",.07),i*90));}
function sfxEvent(){if(!SFX)return;tone(200,.25,"triangle",.04);}
function sfxUpgrade(){if(!SFX)return;tone(330,.1,"sine",.05);setTimeout(()=>tone(440,.12,"sine",.05),110);}


// ┌─ game state ────────────────────────────────────────────────
let G=null,OP_SEQ=1,feedFilter="",notifEnabled=false;

const FRAG_KEY="parsec_frag_v1";
let FRAG={depth:0,totalRuns:0,totalLost:0};
function loadFrag(){try{const r=localStorage.getItem(FRAG_KEY);if(r)FRAG=Object.assign(FRAG,JSON.parse(r));}catch(e){}}
function saveFrag(){try{localStorage.setItem(FRAG_KEY,JSON.stringify(FRAG));}catch(e){}}

function newGame(){
  OP_SEQ=1;
  G={
    res:{aw:120,credits:50,salvage:16,data:1},
    cap:{aw:280,credits:260,salvage:160,data:100},
    ops:[],active:[],tech:{},fac:{cap:0,barracks:0,train:0},
    maxOps:4,feed:[],lastTick:Date.now(),
    nextEvent:Date.now()+rand(30,55)*1000,expSeq:1,clearance:0,
    dryUntil:null,gameOver:false,runs:0,lost:0,
    achievements:{},autoRuns:0,wanderersFound:0,
    totalGained:{aw:0,credits:0,salvage:0,data:0},
    startTime:Date.now(),playTime:0,
    endCleared:false,rfylCleared:false,
    memorial:[],tutorialStep:0,conditionCounts:{quiet:0,clear:0,active:0,hot:0},
    contracts:[],cleanStreak:0,intel:[],discoveredLore:[],
    researchTarget:null,researchProgress:0,
    anomalies:[],anomStudied:0,
    hubCleared:false,parkingCleared:false,
  };
  G.ops=[mkOp(),mkOp()];
  return G;
}

function mkOp(){
  let grit=2+(G?G.fac.train:0);
  let trait=null;
  if(Math.random()<0.45){trait=pick(Object.keys(TRAITS));if(trait==="vet")grit+=2;}
  return{id:OP_SEQ++,name:pick(FIRST)+" "+pick(LAST),grit,xp:0,cond:100,status:"idle",recover:0,trait,runs:0,spec:null,stress:0};
}
function opById(id){return G.ops.find(o=>o.id===id);}
function rand(a,b){return Math.floor(Math.random()*(b-a+1))+a;}
function pick(a){return a[Math.floor(Math.random()*a.length)];}
function clamp(v,a,b){return Math.max(a,Math.min(b,v));}
function fmt(n){n=Math.floor(n);if(n>=1e6)return(n/1e6).toFixed(2)+"M";if(n>=1e4)return(n/1e3).toFixed(1)+"k";return""+n;}
function has(t){return !!G.tech[t];}
function modalOpen(id){const el=document.getElementById(id);return el?el.classList.contains("show"):false;}

function log(msg,cls){
  const t=new Date();
  const stamp=String(t.getHours()).padStart(2,"0")+":"+String(t.getMinutes()).padStart(2,"0")+":"+String(t.getSeconds()).padStart(2,"0");
  G.feed.unshift({stamp,msg,cls:cls||"",fresh:true});
  if(G.feed.length>180)G.feed.pop();
  renderFeed();
}
let toastTimer=null;
function toast(m,cls){
  const el=document.getElementById("toast");
  el.textContent=m;el.className="toast show"+(cls?" "+cls:"");
  clearTimeout(toastTimer);toastTimer=setTimeout(()=>el.classList.remove("show"),2000);
}

function awPerSec(){return 0;}
function awDrainPerSec(){return G.ops.length*0.03;}
function creditsPerSec(){return has("contain")?0.7:0;}
function supplyPerOp(L){return Math.max(2,Math.round(L.time/30)+1);}
function supplyCost(L,n){return Math.max(1,Math.ceil(n*supplyPerOp(L)*(has("logistics")?0.6:1)));}
function speedMult(){return has("drones")?0.75:1;}
function speedMultTeam(ops){let m=speedMult();if(ops.some(o=>o.trait==="swift"))m*=0.85;if(ops.some(o=>o.spec==="scout"))m*=0.8;return m;}
function severityMult(){let m=1;if(has("gear1"))m*=0.7;if(has("gear2"))m*=0.7;return m;}
function recoverMult(){return has("med")?0.5:1;}
function lootMultBase(){return has("gear1")?1.12:1;}
function deepBonus(L){return(L.id>=5&&has("deep"))?1.25:1;}
function levelUnlocked(L){
  if(G.clearance<L.unlock)return false;
  if((L.id===3||L.id===4||L.id===9||L.id===10||L.id===11||L.id===12||L.id===16||L.id===17||L.id===18)&&!has("cart1"))return false;
  if((L.id===5||L.id===6||L.id===8||L.id===13||L.id===14||L.id===15||L.id===19||L.id===20)&&!has("cart2"))return false;
  return true;
}
function highestTier(){let m=0;for(const L of LEVELS)if(levelUnlocked(L))m=Math.max(m,L.id);return m;}

function gain(type,amt){
  if(amt>0){G.res[type]=clamp(G.res[type]+amt,0,G.cap[type]);G.totalGained[type]=(G.totalGained[type]||0)+Math.floor(amt);}
}
function canAfford(c){for(const k in c)if(G.res[k]<c[k])return false;return true;}
function spend(c){if(!canAfford(c))return false;for(const k in c)G.res[k]-=c[k];return true;}

/* ---- suggestion bar ---- */
function updateSuggestBar(){
  const bar=document.getElementById("suggestBar");
  if(!bar)return;
  const idleOps=G.ops.filter(o=>o.status==="idle");
  // Tutorial steps override suggestion bar for early game guidance
  if(G.tutorialStep===0&&G.runs===0&&idleOps.length>0){
    bar.style.display="block";
    bar.textContent="▶ New? Expand a level below, then click Dispatch to send your team.";
    return;
  }
  if(G.tutorialStep===1&&G.runs===0&&G.active.length>0){
    bar.style.display="block";
    bar.textContent="Step 2 — Watch the expedition complete. Your team will haul back almond water and other resources.";
    return;
  }
  if(G.tutorialStep===2&&G.active.length===0&&G.runs>0){
    bar.style.display="block";
    bar.textContent="Step 3 — Use credits and salvage in Research to build Hydroponics Bay for a +25% AW expedition yield boost.";
    return;
  }
  // Contextual hints
  const awPct=G.cap.aw>0?G.res.aw/G.cap.aw:1;
  const hasActiveLobby=G.active.some(e=>e.level===0);
  if(awPct<0.15&&!hasActiveLobby){
    bar.style.display="block";
    bar.textContent="🥛 Almond water critical — dispatch to The Lobby to replenish.";
    return;
  }
  if(idleOps.length>=2&&G.active.length===0&&G.runs>0){
    bar.style.display="block";
    bar.textContent="▶ Operatives idle — expand a level and dispatch.";
    return;
  }
  const affordable=TECH.find(t=>{
    if(G.tech[t.id])return false;
    if(!t.req.every(r=>G.tech[r]))return false;
    return canAfford(t.cost);
  });
  if(affordable){
    bar.style.display="block";
    bar.textContent="⚙ Research available: "+affordable.name+" — check Research panel.";
    return;
  }
  bar.style.display="none";
}

/* ---- tutorial ---- */
function advanceTutorial(step){
  if(G.tutorialStep===step){G.tutorialStep=step+1;updateSuggestBar();}
}

/* ---- dispatch ---- */
let DS={level:null,sel:[],auto:false};
function openDispatch(id){
  const L=levelById(id);
  if(!levelUnlocked(L)){toast("Route not cleared yet.");return;}
  DS={level:L,sel:[],auto:false};
  document.getElementById("dispatchTitle").textContent="Dispatch · "+levelTag(L)+" — "+L.name;
  document.getElementById("dispatchFlav").textContent=L.flav;
  const dur=Math.round(L.time*speedMult());
  document.getElementById("dispatchInfo").innerHTML=
    "<b>Duration</b> ~"+dur+"s · <b>Danger</b> <span style='color:var(--red)'>"+Math.round(L.danger*100)+"%</span>"
    +" · <span class='muted'>Supply cost updates as you pick operatives.</span>";
  const activeIntel=G.intel.find(x=>x.levelId===id&&x.expires>Date.now());
  if(activeIntel){
    const mins=Math.ceil((activeIntel.expires-Date.now())/60000);
    document.getElementById("dispatchInfo").innerHTML+=" <span style='color:var(--green);font-size:10.5px'>● Intel active — +25% haul ("+mins+"m left)</span>";
  }
  document.getElementById("autoToggle").classList.remove("on");
  document.getElementById("autoBox").textContent="";
  renderPicker();updateDispatchCost();
  document.getElementById("dispatchBg").classList.add("show");
}
function levelTag(L){return L.code?("LVL "+L.code):("LEVEL "+L.id);}
// ┌─ dispatch logic ────────────────────────────────────────────
function updateDispatchCost(){
  const cost=supplyCost(DS.level,DS.sel.length);
  const ok=G.res.aw>=cost&&DS.sel.length>0;
  document.getElementById("dispatchCost").innerHTML=
    "<span class='"+(G.res.aw>=cost?"":"no")+"'>🥛 "+cost+" almond water</span>"
    +"<span class='muted'>· "+DS.sel.length+" assigned</span>";
  document.getElementById("dispatchGo").disabled=!ok;
}
function confirmDispatch(){
  if(G.active.length>=2){toast("Max 2 simultaneous expeditions.");return;}
  const cost=supplyCost(DS.level,DS.sel.length);
  if(!DS.sel.length||G.res.aw<cost)return;
  G.res.aw-=cost;
  const L=DS.level;
  const selOps=DS.sel.map(opById).filter(Boolean);
  const dur=Math.round(L.time*speedMultTeam(selOps));
  const condId=rollCondition();
  if(!G.conditionCounts)G.conditionCounts={quiet:0,clear:0,active:0,hot:0};
  G.conditionCounts[condId]=(G.conditionCounts[condId]||0)+1;
  for(const id of DS.sel){const o=opById(id);if(o)o.status="deployed";}
  G.active.push({id:G.expSeq++,level:L.id,ops:[...DS.sel],end:Date.now()+dur*1000,dur,auto:DS.auto,noclipResolved:false,waiting:false,condition:condId,midEventFired:false});
  log("Team deployed to "+levelTag(L)+" — "+L.name+". "+DS.sel.length+" in the field"+(DS.auto?" (auto-repeat).":".")+" ["+condId.toUpperCase()+"]","report");
  sfxDispatch();
  document.getElementById("dispatchBg").classList.remove("show");
  advanceTutorial(0);
  renderAll();
}

/* ---- quick dispatch ---- */
function quickDispatch(levelId){
  if(G.active.length>=2){toast("Max 2 simultaneous expeditions.");return;}
  const L=levelById(levelId);
  if(!levelUnlocked(L)){toast("Route not cleared yet.");return;}
  const idle=G.ops.filter(o=>o.status==="idle");
  if(!idle.length){toast("No idle operatives.");return;}
  const cost=supplyCost(L,idle.length);
  if(G.res.aw<cost){toast("Insufficient almond water (need "+cost+").");return;}
  G.res.aw-=cost;
  const ids=idle.map(o=>o.id);
  const dur=Math.round(L.time*speedMultTeam(idle));
  const condId=rollCondition();
  if(!G.conditionCounts)G.conditionCounts={quiet:0,clear:0,active:0,hot:0};
  G.conditionCounts[condId]=(G.conditionCounts[condId]||0)+1;
  for(const o of idle)o.status="deployed";
  G.active.push({id:G.expSeq++,level:L.id,ops:ids,end:Date.now()+dur*1000,dur,auto:false,noclipResolved:false,waiting:false,condition:condId,midEventFired:false});
  toast("Sent "+idle.length+" to "+L.name+" ["+condId.toUpperCase()+"].");
  log("Quick dispatch: "+idle.length+" operatives sent to "+levelTag(L)+" — "+L.name+". ["+condId.toUpperCase()+"]","report");
  sfxDispatch();
  advanceTutorial(0);
  renderAll();
}

/* ---- recall ---- */
function recallExpedition(expId){
  const idx=G.active.findIndex(e=>e.id===expId);
  if(idx<0)return;
  const exp=G.active.splice(idx,1)[0];
  const L=levelById(exp.level);
  for(const id of exp.ops){const o=opById(id);if(o&&o.status==="deployed")o.status="idle";}
  log("team recalled early from "+levelTag(L)+" — "+L.name+".","report");
  renderAll();
}

/* ---- noclip ---- */
function beginResolve(exp){
  const team=exp.ops.map(opById).filter(Boolean);
  if(!exp.noclipResolved&&team.length>=2){
    const pathers=team.filter(o=>o.trait==="path").length;
    const scouts=team.filter(o=>o.spec==="scout").length;
    const chance=0.16*(pathers?0.4:1)*(has("psi")?0.4:1)*(scouts?0.6:1);
    const cand=team.filter(o=>o.trait!=="path");
    if(cand.length&&Math.random()<chance){openNoclip(exp,pick(cand).id);return true;}
  }
  finishResolve(exp);return false;
}
function openNoclip(exp,opId){
  const L=levelById(exp.level),o=opById(opId);
  const wait=rand(30,55);
  document.getElementById("noclipFlav").textContent=o.name+" stepped through a wall that wasn't load-bearing and dropped into a space between rooms. Comms are faint but alive.";
  document.getElementById("noclipQ").textContent="The rest of the team in "+levelTag(L)+" can hold position to pull them back, or cut losses and extract now.";
  const box=document.getElementById("noclipChoices");box.innerHTML="";
  const b1=document.createElement("button");b1.className="choice";
  b1.innerHTML="Wait for "+o.name+"<span class='rk'>Expedition extends ~"+wait+"s · everyone comes home</span>";
  b1.onclick=()=>{
    exp.noclipResolved=true;exp.waiting=true;exp.end=Date.now()+wait*1000;exp.dur=wait;
    G.active.push(exp);
    log(levelTag(L)+": team holding to recover "+o.name+". +"+wait+"s.","report");
    closeNoclip();
  };
  const b2=document.createElement("button");b2.className="choice";
  b2.innerHTML="Leave "+o.name+"<span class='rk'>"+(has("beacon")?"Beacon traces them — returned badly injured":"They are lost for good")+" · rest extract now</span>";
  b2.onclick=()=>{
    if(has("beacon")){
      o.cond=clamp(o.cond-rand(40,70),0,100);o.status="injured";o.recover=Date.now()+rand(80,150)*1000*recoverMult();
      log(levelTag(L)+": "+o.name+" left behind, but beacon network traced them. Badly injured.","bad");
    }else{
      if(!G.memorial)G.memorial=[];
      G.memorial.push({name:o.name,grit:o.grit,runs:o.runs||0,trait:o.trait,spec:o.spec,level:L.name,cause:"noclipped — space between the walls",when:new Date().toLocaleTimeString()});
      G.ops=G.ops.filter(x=>x.id!==opId);G.lost++;
      log(levelTag(L)+": "+o.name+" was left behind. The space between rooms kept them.","bad");
    }
    exp.ops=exp.ops.filter(id=>id!==opId);finishResolve(exp);closeNoclip();
  };
  box.appendChild(b1);box.appendChild(b2);
  document.getElementById("noclipBg").classList.add("show");
}
function closeNoclip(){document.getElementById("noclipBg").classList.remove("show");renderAll();}

function finishResolve(exp){
  const L=levelById(exp.level);
  const cond=getCondition(exp.condition||"clear");
  const team=exp.ops.map(opById).filter(Boolean);
  const teamGrit=team.reduce((s,o)=>s+o.grit,0);
  const sev=L.danger*severityMult()*cond.dangerMult;
  const teamFactor=clamp(1-(teamGrit-team.length)*0.06-(team.length-1)*0.05,0.22,1);

  let hurt=[],lost_ops=[];
  if(Math.random()<sev*1.3){
    const ent=pick(ENTITY_NAMES);
    for(const o of team){
      let risk=sev*teamFactor*(1-(o.grit+(o.spec==="enforcer"?2:0))*0.04);
      if(o.trait==="tough")risk*=0.5;
      if(o.spec==="enforcer")risk*=0.5;
      risk*=(1+(o.stress||0)*0.004);
      const roll=Math.random();
      if(roll<risk*0.22&&L.danger>0.35&&o.trait!=="path")lost_ops.push(o);
      else if(roll<risk)hurt.push(o);
    }
    for(const o of lost_ops){
      if(!G.memorial)G.memorial=[];
      G.memorial.push({name:o.name,grit:o.grit,runs:o.runs||0,trait:o.trait,spec:o.spec,level:L.name,cause:"lost on expedition",when:new Date().toLocaleTimeString()});
      G.ops=G.ops.filter(x=>x.id!==o.id);G.lost++;
    }
    for(const o of hurt){
      o.cond=clamp(o.cond-rand(25,55),0,100);o.status="injured";
      o.recover=Date.now()+rand(40,95)*1000*recoverMult()*(team.some(t=>t.spec==="medic")?0.4:1);
      o.stress=clamp((o.stress||0)+rand(10,22),0,100);
    }
    if(lost_ops.length){log(levelTag(L)+": contact with "+ent+". Lost "+lost_ops.map(o=>o.name).join(", ")+".","bad");sfxBad();}
    if(hurt.length){log(levelTag(L)+": contact with "+ent+". Injured "+hurt.map(o=>o.name).join(", ")+".","bad");sfxBad();}
    if(!lost_ops.length&&!hurt.length)log(levelTag(L)+": brief contact with "+ent+", broke line of sight.","report");
  }

  // Stress from lost ops and HOT condition
  const survivors=team.filter(o=>opById(o.id));
  if(lost_ops.length){for(const o of survivors)o.stress=clamp((o.stress||0)+20,0,100);}
  if(cond.id==="hot"){for(const o of survivors)o.stress=clamp((o.stress||0)+rand(5,12),0,100);}
  // Log psychological strain
  for(const o of survivors){if((o.stress||0)>75)log(o.name+" is showing signs of psychological strain.","report");}

  let lm=lootMultBase()*deepBonus(L)*(1+teamGrit*0.05)*cond.lootMult;
  lm+=survivors.filter(o=>o.trait==="lucky").length*0.15;
  lm*=(0.5+0.5*(survivors.length/Math.max(1,team.length)));
  // Stress loot penalty
  lm*=(1-(G.ops.filter(o=>o.status!=="deployed").reduce((s,o)=>s+(o.stress||0),0)/Math.max(1,G.ops.length))*0.002);
  // Intel boost
  const intelIdx=(G.intel||[]).findIndex(x=>x.levelId===L.id&&x.expires>Date.now());
  if(intelIdx>=0){lm*=G.intel[intelIdx].boost;G.intel.splice(intelIdx,1);log("Intel acted on — "+L.name+" haul boosted.","good");}
  lm*=(exp._lootMult||1);
  const mules=survivors.filter(o=>o.trait==="mule").length;
  const gained={};
  for(const k in L.loot){
    let[a,b]=L.loot[k];
    let v=Math.round(rand(a,b)*lm);
    if(k==="aw"&&mules)v=Math.round(v*(1+mules*0.2));
    if(k==="aw"&&has("hydro"))v=Math.round(v*1.25);
    if((k==="credits"||k==="salvage")&&survivors.some(o=>o.spec==="salvager"))v=Math.round(v*1.3);
    if(v>0){gain(k,v);gained[k]=v;}
  }
  if(survivors.length&&Math.random()<0.07){
    const bonus=rand(20,60)+L.id*10;gain("credits",bonus);gained.credits=(gained.credits||0)+bonus;
    log(levelTag(L)+": NOTABLE FIND — intact anomaly secured. +"+bonus+" ¤.","good");
  }
  if(survivors.length&&Math.random()<0.12){
    const w=rand(4,9);gain("aw",w);gained.aw=(gained.aw||0)+w;
    log(levelTag(L)+": working vending machine. +"+w+" bottles.","good");
  }

  G.clearance+=1+Math.floor(L.id/2);G.runs++;
  if(L.id===6)G.endCleared=true;
  if(L.id===8)G.rfylCleared=true;
  if(L.id===13)G.hubCleared=true;
  if(L.id===12)G.parkingCleared=true;
  if(L.id===16)G.voidLoungeCleared=true;
  if(L.id===18)G.greenhouseCleared=true;
  if(L.id===19)G.cataCleared=true;

  for(const o of survivors){
    if(o.status==="injured")continue;
    o.status="idle";o.xp+=4+L.id*2;o.runs=(o.runs||0)+1;
    if(o.xp>=o.grit*12){o.xp-=o.grit*12;o.grit++;log(o.name+" promoted to Grit "+o.grit+".","good");if(o.grit===4&&!o.spec)setTimeout(()=>openSpecModal(o.id),400);}
  }

  const back=survivors.filter(o=>o.status==="idle");
  if(back.length){log(levelTag(L)+" — "+L.name+": extracted. Back: "+back.map(o=>o.name).join(", ")+".","report");sfxReturn();}
  else log(levelTag(L)+" — "+L.name+": expedition closed.","report");
  if(gained.aw)log("Hauled back +"+gained.aw+" 🥛 almond water.","good");
  const other=[];
  if(gained.credits)other.push(gained.credits+" ¤");
  if(gained.salvage)other.push(gained.salvage+" ⚙");
  if(gained.data)other.push(gained.data+" ▦");
  if(other.length)log("Also recovered: "+other.join(", ")+".","good");
  if(!gained.aw&&!other.length)log("The level gave nothing this time.","report");
  const r=Math.random();
  if(r<0.32)log(pick(DISCOVERIES),"report");
  else if(r<0.55)log(pick(DREAD),"dread");

  // Intel generation
  if(survivors.length&&Math.random()<0.18){
    const candidates=LEVELS.filter(l=>l.id!==L.id&&levelUnlocked(l));
    if(candidates.length){
      const target=pick(candidates);
      const flavours=["running unusually quiet","showing dense salvage readings","reporting active entity movement","holding steady — good window to move"];
      const fl=pick(flavours);
      G.intel=(G.intel||[]).filter(x=>x.levelId!==target.id);
      G.intel.push({levelId:target.id,boost:1.25,expires:Date.now()+180000});
      log("Field intel: "+target.name+" is "+fl+". +25% haul if you move in the next 3 minutes.","report");
    }
  }

  // Lore discovery
  if(!G.discoveredLore)G.discoveredLore=[];
  const undiscoveredLore=LORE.filter(l=>!G.discoveredLore.includes(l.id));
  if(undiscoveredLore.length&&Math.random()<0.08){
    const loreDoc=pick(undiscoveredLore);
    G.discoveredLore.push(loreDoc.id);
    setTimeout(()=>openLore(loreDoc.id,true),900);
  }

  // Anomaly generation
  const anomCap=has("anomreg")?5:3;
  if(survivors.length&&Math.random()<0.22&&G.anomalies.length<anomCap){
    const anom={
      id:Date.now(),
      name:pick(ANOMALY_NAMES),
      levelId:L.id,
      progress:0,
      reward:{data:rand(8,20)+L.id,credits:rand(25,60)+L.id*4},
      risk:0.08+L.danger*0.12,
    };
    G.anomalies.push(anom);
    log("Team returned with an anomalous object: \""+anom.name+"\". Containment initiated.","dread");
  }

  if(exp.auto){
    const reSel=survivors.filter(o=>o.status==="idle").map(o=>o.id);
    const cost=supplyCost(L,reSel.length);
    if(reSel.length&&G.res.aw>=cost){
      G.res.aw-=cost;
      for(const id of reSel){const o=opById(id);if(o)o.status="deployed";}
      const selOps=reSel.map(opById).filter(Boolean);
      const dur=Math.round(L.time*speedMultTeam(selOps));
      const condId=rollCondition();
      if(!G.conditionCounts)G.conditionCounts={quiet:0,clear:0,active:0,hot:0};
      G.conditionCounts[condId]=(G.conditionCounts[condId]||0)+1;
      G.active.push({id:G.expSeq++,level:exp.level,ops:reSel,end:Date.now()+dur*1000,dur,auto:true,noclipResolved:false,waiting:false,condition:condId,midEventFired:false});
      G.autoRuns=(G.autoRuns||0)+1;
    }else if(reSel.length)log("Auto-repeat halted: insufficient supplies.","report");
  }

  const hadInjury=hurt.length>0||lost_ops.length>0;
  trackContracts(gained,hadInjury);

  advanceTutorial(1);
  maybeNotify(L,back);
  checkAchievements();
}

function maybeNotify(L,back){
  if(!notifEnabled||!("Notification" in window)||Notification.permission!=="granted")return;
  try{new Notification("PARSEC",{body:levelTag(L)+" — "+L.name+": "+back.length+" returned.",silent:true});}catch(e){}
}

/* ---- recruit ---- */
function recruitCost(){const n=G.ops.length;return{credits:22+n*16,aw:8+n*3};}
function recruit(){
  if(G.ops.length>=G.maxOps){toast("Site at capacity. Build a Barracks.");return;}
  const c=recruitCost();if(!spend(c)){toast("Not enough resources.");return;}
  const o=mkOp();
  const input=prompt("Name this operative (blank = random):");
  if(input&&input.trim())o.name=input.trim();
  G.ops.push(o);
  log("Recruited "+o.name+" (Grit "+o.grit+(o.trait?", "+TRAITS[o.trait].name:"")+"). Welcome to PARSEC.","good");
  checkAchievements();renderAll();
}

/* ---- research / facility ---- */
function researchRate(){
  const idle=G.ops.filter(o=>o.status==="idle").length;
  if(!idle)return 0;
  return 0.5+Math.min(idle,4)*0.5; // 1.0/s @1 idle → 2.5/s @4+ idle
}
function buyTech(t){
  if(G.tech[t.id])return;
  for(const r of t.req){if(!G.tech[r]){toast("Requires "+TECH.find(x=>x.id===r).name+".");return;}}
  if(G.researchTarget){
    const cur=TECH.find(x=>x.id===G.researchTarget);
    toast("Already researching "+(cur?cur.name:"…")+". Cancel or wait.");return;
  }
  if(!spend(t.cost)){toast("Insufficient resources.");return;}
  G.researchTarget=t.id;G.researchProgress=0;
  const rate=researchRate();
  const eta=rate>0?Math.ceil(100/rate)+"s":"no idle operatives — research paused";
  log("Research queued: "+t.name+". ETA ~"+eta+".","report");
  sfxUpgrade();renderResearch();
}
function facCost(f){const l=G.fac[f.id];const c={};for(const k in f.base)c[k]=Math.round(f.base[k]*Math.pow(f.mult,l));return c;}
function buyFac(f){
  const c=facCost(f);if(!spend(c)){toast("Insufficient resources.");return;}
  G.fac[f.id]++;
  if(f.id==="cap"){G.cap.aw=Math.round(G.cap.aw*1.5);G.cap.credits=Math.round(G.cap.credits*1.5);G.cap.salvage=Math.round(G.cap.salvage*1.5);G.cap.data=Math.round(G.cap.data*1.5);}
  if(f.id==="barracks")G.maxOps++;
  log(f.name+" upgraded to level "+G.fac[f.id]+".","good");sfxUpgrade();renderAll();
}

/* ---- events ---- */
function hurtRandom(reason){
  const idle=G.ops.filter(o=>o.status!=="injured");
  if(!idle.length)return;
  const o=pick(idle);o.cond=clamp(o.cond-rand(25,50),0,100);o.status="injured";o.recover=Date.now()+rand(40,90)*1000*recoverMult();
}
// ┌─ encounter & event handlers ────────────────────────────────
function fireLevelEvent(ev,exp){
  document.getElementById("eventTitle").textContent="Field event · "+ev.title;
  document.getElementById("eventFlav").textContent=ev.flav;
  document.getElementById("eventQ").textContent=ev.q||"";
  const box=document.getElementById("eventChoices");box.innerHTML="";
  const btns=[];
  for(const c of ev.choices){
    const b=document.createElement("button");b.className="choice";
    b.innerHTML=c.label+"<span class='rk'>"+c.rk+"</span>";
    b.onclick=()=>{clearEventTimers();c.run(exp);document.getElementById("eventBg").classList.remove("show");checkAchievements();renderAll();};
    box.appendChild(b);btns.push(b);
  }
  clearEventTimers();
  let remaining=10;
  const cd=document.getElementById("eventCountdown");
  if(cd)cd.textContent="Auto: "+remaining+"s";
  _evCdTimer=setInterval(()=>{remaining--;if(cd)cd.textContent=remaining>0?"Auto: "+remaining+"s":"";if(remaining<=0)clearInterval(_evCdTimer);},1000);
  _evAutoTimer=setTimeout(()=>{clearEventTimers();if(btns.length)btns[btns.length-1].click();},10000);
  document.getElementById("eventBg").classList.add("show");
  sfxEvent();
}

let _evAutoTimer=null,_evCdTimer=null;
function clearEventTimers(){
  if(_evAutoTimer)clearTimeout(_evAutoTimer);
  if(_evCdTimer)clearInterval(_evCdTimer);
  _evAutoTimer=null;_evCdTimer=null;
  const cd=document.getElementById("eventCountdown");if(cd)cd.textContent="";
}

let _encAutoTimer=null,_encCdTimer=null;
function clearEncounterTimers(){
  if(_encAutoTimer)clearTimeout(_encAutoTimer);
  if(_encCdTimer)clearInterval(_encCdTimer);
  _encAutoTimer=null;_encCdTimer=null;
}
function fireEncounter(exp,L,ent){
  document.getElementById("encName").textContent=ent.icon+" "+ent.name;
  document.getElementById("encFlav").textContent=ent.flav;
  const team=exp.ops.map(id=>opById(id)).filter(Boolean);
  const grit=team.reduce((a,o)=>a+o.grit,0);
  const fightChance=Math.min(0.85,Math.max(0.12,grit*9/(ent.threat*100+grit*9)));
  document.getElementById("encOdds").textContent="Fight odds: "+Math.round(fightChance*100)+"%  ·  Grit "+grit;
  document.getElementById("fightBtn").onclick=()=>{
    clearEncounterTimers();
    exp._encounterActive=false;
    document.getElementById("encBg").classList.remove("show");
    if(Math.random()<fightChance){
      const sal=rand(10,24);const cr=rand(15,40+Math.round(ent.threat*60));
      gain("salvage",sal);gain("credits",cr);
      log(ent.name+" driven off. Spoils found in the chaos: +"+sal+" salvage, +"+cr+" ¤.","good");sfxAchieve();
    }else{
      hurtRandom("fighting "+ent.name);
      if(ent.threat>0.5&&Math.random()<0.4)hurtRandom(ent.name+" second strike");
      log("Team engaged "+ent.name+". Not all came through cleanly.","bad");sfxBad();
    }
    renderAll();
  };
  document.getElementById("escapeBtn").onclick=()=>resolveEscape(exp,ent);
  clearEncounterTimers();
  let rem=10;
  const cd=document.getElementById("encCd");if(cd)cd.textContent="Auto-evade: "+rem+"s";
  _encCdTimer=setInterval(()=>{rem--;if(cd)cd.textContent=rem>0?"Auto-evade: "+rem+"s":"";if(rem<=0)clearInterval(_encCdTimer);},1000);
  _encAutoTimer=setTimeout(()=>{if(document.getElementById("encBg").classList.contains("show"))resolveEscape(exp,ent);},10000);
  document.getElementById("encBg").classList.add("show");
  sfxBad();
}
function resolveEscape(exp,ent){
  clearEncounterTimers();
  exp._encounterActive=false;
  exp._lootMult=(exp._lootMult||1)*0.8;
  document.getElementById("encBg").classList.remove("show");
  log("Team evaded the "+ent.name+". Some haul abandoned in the retreat.","report");
  renderAll();
}
function fireEvent(){
  const ev=pick(EVENTS);
  document.getElementById("eventTitle").textContent="Field event · "+ev.title;
  document.getElementById("eventFlav").textContent=ev.flav;
  document.getElementById("eventQ").textContent=ev.q;
  const box=document.getElementById("eventChoices");box.innerHTML="";
  const btns=[];
  for(const c of ev.choices){
    const b=document.createElement("button");b.className="choice";
    b.innerHTML=c.label+"<span class='rk'>"+c.rk+"</span>";
    b.onclick=()=>{clearEventTimers();c.run();document.getElementById("eventBg").classList.remove("show");checkAchievements();renderAll();};
    box.appendChild(b);btns.push(b);
  }
  // Auto-choose last (safer) option after 10 seconds
  clearEventTimers();
  let remaining=10;
  const cd=document.getElementById("eventCountdown");
  if(cd)cd.textContent="Auto: "+remaining+"s";
  _evCdTimer=setInterval(()=>{
    remaining--;
    if(cd)cd.textContent=remaining>0?"Auto: "+remaining+"s":"";
    if(remaining<=0)clearInterval(_evCdTimer);
  },1000);
  _evAutoTimer=setTimeout(()=>{
    clearEventTimers();
    if(btns.length)btns[btns.length-1].click();
  },10000);
  document.getElementById("eventBg").classList.add("show");
  sfxEvent();
}

/* ---- achievements ---- */
function checkAchievements(){
  for(const a of ACHIEVEMENTS){
    if(!G.achievements[a.id]&&a.check(G)){
      G.achievements[a.id]=Date.now();
      log("ACHIEVEMENT UNLOCKED: "+a.name+" — "+a.desc,"achieve");
      toast("❆ "+a.name,"good-t");
      sfxAchieve();
    }
  }
  checkFragmentUnlock();
}

/* ---- gameover ---- */
function doGameOver(reason){
  G.gameOver=true;
  document.getElementById("goReason").textContent=reason;
  document.getElementById("goStats").textContent="Expeditions: "+G.runs+" · Lost: "+G.lost+" · Clearance: "+G.clearance+" · Time: "+Math.round((G.playTime||0)/60)+"m";
  document.getElementById("gameoverBg").classList.add("show");
  log("CONTAINMENT LOST. "+reason,"bad");
  sfxBad();
}

/* ---- tick ---- */
function tick(){
  if(G.gameOver)return;
  const now=Date.now();
  let dt=(now-G.lastTick)/1000;if(dt<0)dt=0;if(dt>2)dt=2;
  G.playTime=(G.playTime||0)+dt;
  if(dt>0){gain("aw",awPerSec()*dt);gain("credits",creditsPerSec()*dt);G.res.aw=Math.max(0,G.res.aw-awDrainPerSec()*dt);G.lastTick=now;}
  G.intel=(G.intel||[]).filter(x=>x.expires>now);

  if(G.res.aw<=0.01){
    if(!G.dryUntil){G.dryUntil=now+35000;log("⚠ Almond water empty. Site can hold ~35s. Get water fast.","bad");}
    else if(now>=G.dryUntil){doGameOver("The site ran dry. Without almond water the operatives could not be sustained.");return;}
  }else if(G.dryUntil){G.dryUntil=null;log("Almond water restored. Rationing lifted.","good");}

  if(!modalOpen("noclipBg")){
    for(let i=G.active.length-1;i>=0;i--){
      if(now>=G.active[i].end){const exp=G.active.splice(i,1)[0];const opened=beginResolve(exp);if(opened)break;}
    }
  }
  let changed=false;
  for(const o of G.ops){
    if(o.status==="injured"&&now>=o.recover){o.status="idle";o.cond=clamp(o.cond+40,0,100);log(o.name+" cleared by medical and back on the roster.","good");changed=true;}
  }
  // Stress decrease for idle ops
  for(const o of G.ops){
    if(o.status==="idle"&&(o.stress||0)>0){
      o.stress=Math.max(0,(o.stress||0)-1.5*dt);
    }
  }
  // Stress breakdown
  const burnout=G.ops.filter(o=>o.status==="idle"&&(o.stress||0)>=90);
  if(burnout.length&&Math.random()<0.001){
    const o=pick(burnout);
    o.status="injured";o.recover=Date.now()+90000;o.stress=clamp((o.stress||0)-30,0,100);
    log(o.name+" broke down from accumulated stress. Recovering for 90s.","bad");sfxBad();
    changed=true;
  }
  if(now>=G.nextEvent){G.nextEvent=now+rand(50,100)*1000;if(!modalOpen("eventBg")&&!modalOpen("dispatchBg")&&!modalOpen("noclipBg"))fireEvent();}

  // Mid-expedition level events
  if(!modalOpen("eventBg")&&!modalOpen("noclipBg")&&!modalOpen("dispatchBg")&&!modalOpen("encBg")){
    for(const exp of G.active){
      if(!exp.midEventFired){
        const elapsed=(exp.dur*1000-(exp.end-Date.now()));
        const pct=elapsed/(exp.dur*1000);
        if(pct>0.45&&pct<0.75&&Math.random()<0.0008){
          const levEvs=LEVEL_EVENTS[exp.level];
          if(levEvs&&levEvs.length){
            exp.midEventFired=true;
            fireLevelEvent(pick(levEvs),exp);
          }
        }
      }
    }
  }

  // Entity encounters
  if(!modalOpen("encBg")&&!modalOpen("eventBg")&&!modalOpen("noclipBg")&&!modalOpen("dispatchBg")){
    for(const exp of G.active){
      if(!exp.encounterFired){
        const elapsed=(exp.dur*1000-(exp.end-Date.now()));
        const pct=elapsed/(exp.dur*1000);
        const L=levelById(exp.level);
        if(L&&pct>0.2&&pct<0.7&&Math.random()<L.danger*0.0018){
          exp.encounterFired=true;
          exp._encounterActive=true;
          const ent=ENTITIES[L.id]||{name:"Unknown Form",icon:"◈",threat:0.3,flav:"Something unidentified in the facility. The team can feel it before they can see it."};
          exp._encEntity=ent;
          fireEncounter(exp,L,ent);
        }
      }
    }
  }

  // Stress decay for idle operatives
  for(const o of G.ops){
    if(o.status==="idle"&&(o.stress||0)>0){
      o.stress=Math.max(0,(o.stress||0)-(has("psych")?2:1)*dt);
    }
  }

  // Anomaly study
  if(G.anomalies&&G.anomalies.length){
    const idleCount=G.ops.filter(o=>o.status==="idle").length;
    const anomRate=idleCount>0?idleCount*0.4*(has("anomreg")?1.8:1):0;
    for(let i=G.anomalies.length-1;i>=0;i--){
      G.anomalies[i].progress+=anomRate*dt;
      if(G.anomalies[i].progress>=100){
        const a=G.anomalies.splice(i,1)[0];
        for(const k in a.reward)gain(k,a.reward[k]);
        const rStr=Object.entries(a.reward).map(([k,v])=>"+"+v+({credits:"¤",data:"▦"}[k]||k)).join(" ");
        log("Study complete: \""+a.name+"\" — "+rStr+".","good");
        sfxAchieve();
        G.anomStudied=(G.anomStudied||0)+1;
        if(Math.random()<a.risk){
          hurtRandom("containment breach");
          log("Containment breach during study. Operative injured.","bad");sfxBad();
        }
        checkAchievements();
        changed=true;
      }
    }
  }

  if(G.researchTarget){
    G.researchProgress+=researchRate()*dt;
    if(G.researchProgress>=100){
      const t=TECH.find(x=>x.id===G.researchTarget);
      G.tech[G.researchTarget]=true;
      if(G.researchTarget==="contain")G.cap.credits+=400;
      log("Research complete: "+t.name+".","good");sfxAchieve();
      G.researchTarget=null;G.researchProgress=0;
      checkAchievements();renderResearch();
    }
  }
  renderResbar();renderWarn();renderActive();renderContracts();updateSuggestBar();renderAnomalies();
  tickRoster();
  tickResearch();
  if(changed)renderMemorial();
  checkAchievements();
}

function offlineCatchup(){
  const now=Date.now();const away=(now-G.lastTick)/1000;
  if(away>20){
    gain("aw",awPerSec()*away);gain("credits",creditsPerSec()*away);
    for(let i=G.active.length-1;i>=0;i--){if(now>=G.active[i].end){const exp=G.active.splice(i,1)[0];exp.noclipResolved=true;finishResolve(exp);}}
    if(G.res.aw<1)G.res.aw=1;
    G.lastTick=now;G.dryUntil=null;
    log("Site idled "+Math.round(away/60)+" min while you were away. Yields collected, runs resolved.","report");
  }
}

/* ---- render ---- */
// ┌─ fragment prestige ─────────────────────────────────────────
/* ---- fragment prestige ---- */
let _fragTransferOp=null;
function openFragModal(){
  const depth=FRAG.depth;
  document.getElementById("fragFlav").textContent=depth===0
    ?"Survey teams have found a new fragment of the facility — uncharted, fresh. Moving the operation there means starting over. But you'd take what you've learned. And one person."
    :"Fragment "+(depth+1)+". Another section. The facility goes deeper than anyone expected. Each base you've built ends the same way — a door, and a new corridor beyond it.";
  document.getElementById("fragInfo").innerHTML=
    "<b style='color:var(--green)'>Fragment depth: "+depth+"</b> · Bonus on arrival: +"+(depth*20)+"% starting AW · Transferred operative gains +"+depth+" Grit<br>"+
    "<span style='color:var(--ink-dim);font-size:10.5px'>All resources, research, and facility upgrades reset. Achievements and memorial carry over.</span>";
  const picker=document.getElementById("fragPicker");picker.innerHTML="";
  _fragTransferOp=null;
  for(const o of G.ops){
    const slot=document.createElement("div");slot.className="slot";
    slot.innerHTML="<b>"+o.name+"</b><span class='muted'> G"+o.grit+(o.spec?" · "+SPECS[o.spec].name:"")+"</span>";
    slot.onclick=()=>{
      _fragTransferOp=o.id;
      picker.querySelectorAll(".slot").forEach(s=>s.classList.remove("sel"));
      slot.classList.add("sel");
      document.getElementById("fragConfirm").disabled=false;
    };
    picker.appendChild(slot);
  }
  document.getElementById("fragConfirm").disabled=true;
  document.getElementById("fragBg").classList.add("show");
}
function doFragment(){
  const transferredOp=opById(_fragTransferOp);
  if(!transferredOp)return;
  FRAG.depth++;FRAG.totalRuns=(FRAG.totalRuns||0)+G.runs;FRAG.totalLost=(FRAG.totalLost||0)+G.lost;
  saveFrag();
  const savedMemorial=[...G.memorial];const savedAch={...G.achievements};const savedLore=[...G.discoveredLore];
  const transferName=transferredOp.name;const transferGrit=transferredOp.grit;const transferTrait=transferredOp.trait;const transferSpec=transferredOp.spec;
  newGame();
  G.memorial=savedMemorial;G.achievements=savedAch;G.discoveredLore=savedLore;
  G.res.aw=Math.round(G.res.aw*(1+FRAG.depth*0.2));
  G.res.credits=Math.round(G.res.credits*(1+FRAG.depth*0.15));
  const newOp=G.ops[0];
  newOp.name="(transfer) "+transferName;
  newOp.grit=transferGrit+FRAG.depth;
  newOp.trait=transferTrait;newOp.spec=transferSpec;
  document.getElementById("fragBg").classList.remove("show");
  log("PARSEC Fragment "+(FRAG.depth)+" established. Welcome back, "+transferName+".","good");
  log("Fragment depth: "+FRAG.depth+" · Bonus applied: +"+(FRAG.depth*20)+"% starting resources.","report");
  sfxAchieve();checkFragmentUnlock();renderAll();save(false);
}

// ┌─ save / load ───────────────────────────────────────────────
function save(silent){
  try{localStorage.setItem(SAVE_KEY,JSON.stringify({G,OP_SEQ}));if(!silent){document.getElementById("saveStatus").textContent="Saved "+new Date().toLocaleTimeString()+".";toast("Saved.");}}
  catch(e){document.getElementById("saveStatus").textContent="Save failed (storage blocked)";}
}
function load(){
  try{
    const raw=localStorage.getItem(SAVE_KEY);if(!raw)return false;
    const data=JSON.parse(raw);G=data.G;OP_SEQ=data.OP_SEQ||100;
    if(!G.feed)G.feed=[];if(!G.fac)G.fac={cap:0,barracks:0,train:0};
    if(G.dryUntil===undefined)G.dryUntil=null;if(G.gameOver===undefined)G.gameOver=false;
    if(G.runs===undefined)G.runs=0;if(G.lost===undefined)G.lost=0;
    if(!G.achievements)G.achievements={};if(!G.totalGained)G.totalGained={aw:0,credits:0,salvage:0,data:0};
    if(!G.autoRuns)G.autoRuns=0;if(!G.wanderersFound)G.wanderersFound=0;
    if(!G.playTime)G.playTime=0;if(!G.endCleared)G.endCleared=false;if(!G.rfylCleared)G.rfylCleared=false;
    // Migration for new fields
    if(!G.memorial)G.memorial=[];
    if(G.tutorialStep===undefined)G.tutorialStep=G.runs>0?3:0;
    if(!G.conditionCounts)G.conditionCounts={quiet:0,clear:0,active:0,hot:0};
    for(const o of G.ops)if(!o.runs)o.runs=0;
    if(!G.contracts)G.contracts=[];if(G.cleanStreak===undefined)G.cleanStreak=0;
    if(!G.intel)G.intel=[];
    if(!G.discoveredLore)G.discoveredLore=[];
    if(G.researchTarget===undefined)G.researchTarget=null;
    if(G.researchProgress===undefined)G.researchProgress=0;
    for(const o of G.ops)if(o.spec===undefined)o.spec=null;
    for(const o of G.ops)if(o.stress===undefined)o.stress=0;
    if(!G.anomalies)G.anomalies=[];
    if(!G.anomStudied)G.anomStudied=0;
    if(!G.hubCleared)G.hubCleared=false;
    if(!G.parkingCleared)G.parkingCleared=false;
    if(!G.voidLoungeCleared)G.voidLoungeCleared=false;
    if(!G.greenhouseCleared)G.greenhouseCleared=false;
    if(!G.cataCleared)G.cataCleared=false;
    if(!G.memorial)G.memorial=[];
    // Bug fix: ops stuck as "deployed" with no active expedition get reset to idle
    const activeOpIds=new Set((G.active||[]).flatMap(e=>e.ops||[]));
    for(const o of G.ops){
      if(o.status==="deployed"&&!activeOpIds.has(o.id))o.status="idle";
      if(o.status==="injured"&&o.recover&&o.recover<=Date.now()){o.status="idle";o.cond=Math.min(100,(o.cond||50)+20);}
    }
    return true;
  }catch(e){return false;}
}
function wipe(){if(!confirm("Wipe the site? All progress lost."))return;localStorage.removeItem(SAVE_KEY);startFresh();}
function startFresh(){
  newGame();
  log("PARSEC advance base online. Settlement authority granted by the Sync. Two operatives on site.","report");
  log("Mission: scout, clear, and prepare sectors for settler intake. The Lobby is your first site.","report");
  log("Almond water keeps the advance team alive — it will keep the settlers alive too. Reserve hits zero → site fails in 35 seconds.","report");
  renderAll();
}

