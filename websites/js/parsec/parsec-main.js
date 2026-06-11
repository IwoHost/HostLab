"use strict";
// ┌─ keyboard + notifications + boot ──────────────────────────
document.addEventListener("keydown",e=>{
  if(e.target.tagName==="INPUT"||e.target.tagName==="TEXTAREA"||e.target.isContentEditable)return;
  if(e.key==="Escape"){clearEventTimers();clearEncounterTimers();["dispatchBg","eventBg","statsBg","opDetailBg","specBg","loreBg","loreArchiveBg","fragBg","encBg"].forEach(id=>{const el=document.getElementById(id);if(el)el.classList.remove("show");});return;}
  if(e.key==="h"||e.key==="H"){document.getElementById("hireBtn").click();return;}
  if((e.key==="r"||e.key==="R")&&!G.gameOver){openDispatch(0);return;}
  if(e.key==="s"||e.key==="S"){save(false);return;}
});

/* ---- notifications ---- */
async function toggleNotifs(){
  if(!("Notification" in window)){toast("Notifications not supported in this browser.");return;}
  if(!notifEnabled){
    if(Notification.permission==="default")await Notification.requestPermission();
    notifEnabled=Notification.permission==="granted";
  }else notifEnabled=false;
  const btn=document.getElementById("notifBtn");
  btn.textContent=notifEnabled?"ALERTS ON":"ALERTS OFF";
  btn.classList.toggle("active",notifEnabled);
  toast(notifEnabled?"Alerts enabled.":"Alerts disabled.");
}

/* ---- boot ---- */
function boot(){
  loadFrag();
  if(!load())startFresh();
  else if(G.gameOver){localStorage.removeItem(SAVE_KEY);startFresh();}
  else{offlineCatchup();log("Welcome back. Dispatch console restored.","report");}
  renderAll();
  setInterval(tick,250);
  setInterval(()=>{if(!G.gameOver){save(true);document.getElementById("saveStatus").textContent="Autosaved · "+new Date().toLocaleTimeString();}},8000);
  window.addEventListener("beforeunload",()=>{if(!G.gameOver)save(true);});

  document.getElementById("sfxBtn").onclick=()=>{
    SFX=!SFX;
    const btn=document.getElementById("sfxBtn");
    btn.textContent=SFX?"SFX ON":"SFX OFF";
    btn.classList.toggle("active",SFX);
    if(SFX)sfxAchieve();
  };
  document.getElementById("hireBtn").onclick=recruit;
  document.getElementById("dispatchGo").onclick=confirmDispatch;
  document.getElementById("dispatchCancel").onclick=()=>document.getElementById("dispatchBg").classList.remove("show");
  document.getElementById("autoToggle").onclick=()=>{DS.auto=!DS.auto;document.getElementById("autoToggle").classList.toggle("on",DS.auto);document.getElementById("autoBox").textContent=DS.auto?"✓":"";};
  document.getElementById("selectAllBtn").onclick=()=>{
    const idle=G.ops.filter(o=>o.status==="idle");
    DS.sel=idle.map(o=>o.id);
    renderPicker();updateDispatchCost();
  };
  document.getElementById("selectHalfBtn").onclick=()=>{
    const idle=G.ops.filter(o=>o.status==="idle");
    const half=Math.ceil(idle.length/2);
    DS.sel=idle.slice(0,half).map(o=>o.id);
    renderPicker();updateDispatchCost();
  };
  document.getElementById("saveBtn").onclick=()=>save(false);
  document.getElementById("resetBtn").onclick=wipe;
  document.getElementById("goRestart").onclick=()=>{document.getElementById("gameoverBg").classList.remove("show");startFresh();};
  document.getElementById("statsBtn").onclick=()=>{renderStats();document.getElementById("statsBg").classList.add("show");};
  document.getElementById("statsClose").onclick=()=>document.getElementById("statsBg").classList.remove("show");
  document.getElementById("notifBtn").onclick=toggleNotifs;
  document.getElementById("dispatchBg").addEventListener("click",e=>{if(e.target.id==="dispatchBg")e.currentTarget.classList.remove("show");});
  document.getElementById("statsBg").addEventListener("click",e=>{if(e.target.id==="statsBg")e.currentTarget.classList.remove("show");});
  document.getElementById("opDetailBg").addEventListener("click",e=>{if(e.target.id==="opDetailBg")e.currentTarget.classList.remove("show");});
  document.getElementById("specBg").addEventListener("click",e=>{if(e.target.id==="specBg")e.currentTarget.classList.remove("show");});
  document.getElementById("loreBtn").onclick=()=>{openLoreArchive();};
  document.getElementById("loreAckBtn").onclick=closeLorePopup;
  document.getElementById("loreToArchiveBtn").onclick=openLoreArchive;
  document.getElementById("loreArchiveClose").onclick=()=>document.getElementById("loreArchiveBg").classList.remove("show");
  document.getElementById("loreBg").addEventListener("click",e=>{if(e.target.id==="loreBg")e.currentTarget.classList.remove("show");});
  document.getElementById("loreArchiveBg").addEventListener("click",e=>{if(e.target.id==="loreArchiveBg")e.currentTarget.classList.remove("show");});
  document.getElementById("newFragBtn").onclick=openFragModal;
  document.getElementById("fragConfirm").onclick=doFragment;
  document.getElementById("fragCancel").onclick=()=>document.getElementById("fragBg").classList.remove("show");
  document.getElementById("fragBg").addEventListener("click",e=>{if(e.target.id==="fragBg")e.currentTarget.classList.remove("show");});
  startFieldAnim();
  document.querySelectorAll(".feed-tab").forEach(tab=>{
    tab.onclick=()=>{feedFilter=tab.dataset.f;document.querySelectorAll(".feed-tab").forEach(t=>t.classList.toggle("active",t===tab));renderFeed();};
  });
}
boot();