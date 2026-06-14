"use strict";
// ┌─ canvas field animation ────────────────────────────────────
/* ---- field canvas animation ---- */
let _fieldAF=null,
    _catX=30,_catDir=1,_catBlink=0,_flickT=0,
    _noclipStart=0,
    _panelBgOff=[0,0],_panelWalkF=[0,0],
    _panelFlickS=[[1,0.95,1,0.9,1,0.88,1,0.92],[1,0.95,1,0.9,1,0.88,1,0.92]];

// Per-slot personality: [swingMult, yOffset, lookBackPeriod(s,0=never), bobAmp, lag(s)]
const _OPP=[
  {sw:1.0,yo:0, lb:0, bob:1.0,lag:0  }, // leader
  {sw:1.3,yo:-1,lb:0, bob:1.4,lag:0.4}, // bouncy
  {sw:0.7,yo:0, lb:14,bob:0.8,lag:0.8}, // looks back periodically
  {sw:0.9,yo:1, lb:0, bob:0.6,lag:1.5}, // tired, drags feet
  {sw:1.1,yo:0, lb:8, bob:1.1,lag:0.2}, // alert, glances back often
  {sw:1.4,yo:-2,lb:0, bob:1.5,lag:0  }, // excitable, pushes to front
];

function drawOpSocial(ctx,fi,nTotal,W,H,walkF,t,standing,drawFn){
  const p=_OPP[fi%_OPP.length];
  // Spacing: fit all ops with min 34px gap, max 52px
  const spacing=Math.min(52,Math.max(34,(W-70)/Math.max(nTotal-1,1)));
  const baseX=22+fi*spacing;
  // Lag offset: lagging ops are slightly behind (lower x)
  const lagX=p.lag*20;
  const x=baseX-lagX;
  // Subtle y bob (bouncy types bob more)
  const yBob=standing?0:Math.sin(walkF*0.22*p.bob+fi)*p.yo;
  const y=H-10+yBob;
  // Look-back: if lb period set, face backwards for 1s every lb seconds
  let dir=1;
  if(!standing&&p.lb>0){
    const phase=(t%p.lb)/p.lb;
    if(phase>0.88)dir=-1; // turn around for ~12% of the period
  }
  // Stumble: slot 3 (tired) stumbles briefly every ~18s
  const stumble=(fi===3)&&(Math.floor(t/18)%2===0)&&(t%18>16.5);
  const adjWalkF=stumble?walkF*0.1:walkF;
  // Shoulder-bump: slots 1+2 are close, slot 1 occasionally nudges slot 2
  const nudge=(fi===1)&&(Math.sin(t*0.9)>0.97)?3:0;
  drawFn(ctx,x+nudge,y,adjWalkF+fi*28,dir,standing||stumble);
}

function _getLevelScene(lid){
  if(lid===9)return'pool';
  if(lid===2||lid===10)return'dark';
  if(lid===4)return'cave';
  if(lid===6)return'end';
  if(lid===8)return'rfyl';
  if(lid===11)return'electric';
  if(lid===5)return'hotel';
  if(lid===3)return'office';
  if(lid===1||lid===12)return'warehouse';
  if(lid===13||lid===15||lid===17||lid===19)return'dark';
  if(lid===14||lid===16)return'hotel';
  if(lid===18||lid===20)return'pool';
  return'lobby';
}

function rrOp(ctx,x,y,w,h,r){
  ctx.beginPath();
  ctx.moveTo(x+r,y);ctx.lineTo(x+w-r,y);ctx.quadraticCurveTo(x+w,y,x+w,y+r);
  ctx.lineTo(x+w,y+h-r);ctx.quadraticCurveTo(x+w,y+h,x+w-r,y+h);
  ctx.lineTo(x+r,y+h);ctx.quadraticCurveTo(x,y+h,x,y+h-r);
  ctx.lineTo(x,y+r);ctx.quadraticCurveTo(x,y,x+r,y);
  ctx.closePath();
}

function startFieldAnim(){
  const wrap=document.getElementById("fieldCanvasWrap");if(!wrap)return;
  if(_fieldAF)cancelAnimationFrame(_fieldAF);
  let last=0;
  function frame(ts){
    const dt=Math.min((ts-last)/1000,0.1);last=ts;
    const exps=(typeof G!=="undefined"?G.active:[]).slice(0,2);
    const panels=Math.max(exps.length,1);
    // Sync canvas count
    while(wrap.children.length<panels){
      const c=document.createElement("canvas");
      c.width=420;c.height=110;
      c.style.cssText="width:100%;height:110px;display:block;image-rendering:crisp-edges";
      if(wrap.children.length>0)c.style.borderTop="1px solid var(--line)";
      wrap.appendChild(c);
    }
    while(wrap.children.length>panels)wrap.removeChild(wrap.lastChild);
    // Update flicker (shared)
    _flickT+=dt;
    if(_flickT>0.07+Math.random()*0.13){
      _flickT=0;
      for(let s=0;s<2;s++)for(let i=0;i<8;i++){
        if(Math.random()<0.12)_panelFlickS[s][i]=0.2+Math.random()*0.45;
        else _panelFlickS[s][i]=Math.min(1,_panelFlickS[s][i]+0.38);
      }
    }
    // Draw each canvas
    for(let pi=0;pi<panels;pi++){
      const c=wrap.children[pi];
      if(!c)continue;
      const ctx=c.getContext("2d");
      const exp=exps[pi]||null;
      const isEventOpen=typeof modalOpen==="function"&&modalOpen("eventBg");
      const isNoclipOpen=typeof modalOpen==="function"&&modalOpen("noclipBg");
      const isPaused=isEventOpen||isNoclipOpen;
      if(!exp){
        _catBlink+=dt;_catX+=_catDir*16*dt;
        if(_catX>c.width-24)_catDir=-1;if(_catX<24)_catDir=1;
        drawBase(ctx,c.width,c.height);
        drawCat(ctx,_catX,c.height-10,_catBlink,_catDir);
      } else {
        if(!isPaused){_panelBgOff[pi]+=42*dt;if(_panelBgOff[pi]>90000)_panelBgOff[pi]-=90000;}
        if(!isPaused)_panelWalkF[pi]+=dt*62;
        drawPanel(ctx,pi,c.width,c.height,exp,isEventOpen,isNoclipOpen);
      }
    }
    _fieldAF=requestAnimationFrame(frame);
  }
  _fieldAF=requestAnimationFrame(frame);
}

function drawEntityFigure(ctx,x,y,t){
  ctx.save();
  const flicker=0.55+Math.sin(t*9.3)*0.35;
  ctx.globalAlpha=flicker;
  ctx.shadowBlur=14;ctx.shadowColor="#ff1100";
  ctx.strokeStyle="#ff2200";ctx.lineWidth=1.6;ctx.fillStyle="#ff1100";
  // Head
  ctx.beginPath();ctx.arc(x,y-22,5.5+Math.sin(t*7)*0.4,0,Math.PI*2);ctx.fill();
  // Body
  ctx.beginPath();ctx.moveTo(x,y-16);ctx.lineTo(x+Math.sin(t*3)*1.5,y-3);ctx.stroke();
  // Arms — reaching
  ctx.beginPath();
  ctx.moveTo(x,y-14);ctx.lineTo(x-15+Math.sin(t*4)*2,y-8+Math.cos(t*3.5)*2);
  ctx.moveTo(x,y-14);ctx.lineTo(x+15+Math.sin(t*4+1)*2,y-8+Math.cos(t*3.5+1)*2);
  ctx.stroke();
  // Legs
  ctx.beginPath();
  ctx.moveTo(x,y-3);ctx.lineTo(x-7+Math.sin(t*5)*1,y+9);
  ctx.moveTo(x,y-3);ctx.lineTo(x+7+Math.sin(t*5+2)*1,y+9);
  ctx.stroke();
  ctx.restore();
}
function drawPanel(ctx,pi,W,H,exp,isEventOpen,isNoclipOpen){
  const bgOff=_panelBgOff[pi],walkF=_panelWalkF[pi],flickS=_panelFlickS[pi];
  const scene=_getLevelScene(exp.level);
  const isDark=scene==='dark'||scene==='cave'||scene==='end';
  const isHot=(exp.condition||"clear")==="hot";
  const nOps=(exp.ops||[]).length; // no cap — show them all
  const drawFn=isDark?drawOpDark:drawOp;
  const t=Date.now()*0.001;

  switch(scene){
    case'lobby':   drawBgLobby(ctx,W,H,bgOff,flickS);break;
    case'pool':    drawBgPool(ctx,W,H,bgOff);break;
    case'dark':    drawBgDark(ctx,W,H,bgOff,false);break;
    case'cave':    drawBgDark(ctx,W,H,bgOff,true);break;
    case'end':     drawBgEnd(ctx,W,H);break;
    case'rfyl':    drawBgRFYL(ctx,W,H,bgOff,walkF);break;
    case'electric':drawBgElectric(ctx,W,H,bgOff,walkF);break;
    case'hotel':   drawBgHotel(ctx,W,H,bgOff,flickS);break;
    case'office':  drawBgOffice(ctx,W,H,bgOff,flickS);break;
    case'warehouse':drawBgWarehouse(ctx,W,H,bgOff);break;
    default:       drawBgLobby(ctx,W,H,bgOff,flickS);
  }

  if(isNoclipOpen){
    if(!_noclipStart)_noclipStart=Date.now();
    const sink=Math.min(1,(Date.now()-_noclipStart)/3500);
    for(let fi=0;fi<nOps-1;fi++)drawOpSocial(ctx,fi,nOps,W,H,walkF,t,true,drawFn);
    ctx.save();ctx.globalAlpha=Math.max(0.1,0.95-sink*0.9);
    drawFn(ctx,W*0.55,H-10+sink*38,0,1,true);ctx.restore();
    if(sink<0.85){
      ctx.save();ctx.globalAlpha=0.5*(1-sink);ctx.strokeStyle="#ff5500";ctx.lineWidth=1.5;
      ctx.beginPath();ctx.ellipse(W*0.55,H-8,sink*32+4,(sink*32+4)*0.28,0,0,Math.PI*2);ctx.stroke();ctx.restore();
    }
    ctx.save();ctx.globalAlpha=0.8;ctx.fillStyle="#ff6600";
    ctx.font="bold 8px monospace";ctx.textAlign="center";
    ctx.fillText("⚠ NOCLIP",W*0.55,H-54);ctx.restore();
  } else if(exp._encounterActive){
    if(_noclipStart)_noclipStart=0;
    for(let fi=0;fi<nOps;fi++)drawOpSocial(ctx,fi,nOps,W,H,walkF,t,true,drawFn);
    const entX=W-26+Math.sin(t*1.8)*2;
    drawEntityFigure(ctx,entX,H-12,t);
    ctx.save();ctx.fillStyle="#ff2200";ctx.font="bold 8px monospace";ctx.textAlign="center";
    ctx.globalAlpha=0.7+Math.sin(t*6)*0.3;
    ctx.fillText("⚠ CONTACT",W/2,H-72);ctx.restore();
  } else if(isEventOpen){
    if(_noclipStart)_noclipStart=0;
    for(let fi=0;fi<nOps;fi++)drawOpSocial(ctx,fi,nOps,W,H,walkF,t,true,drawFn);
    const bounce=Math.sin(Date.now()*0.0042)*3;
    ctx.save();ctx.globalAlpha=0.85+Math.sin(Date.now()*0.0065)*0.12;
    ctx.fillStyle="#d4a830";ctx.font="bold 13px monospace";ctx.textAlign="center";
    ctx.fillText("!",W/2,H-54+bounce);ctx.restore();
    const evEl=document.getElementById("eventTitle");
    if(evEl&&evEl.textContent.toLowerCase().includes("wanderer"))
      drawWanderer(ctx,W-50,H-10,Date.now()*0.001);
  } else {
    if(_noclipStart)_noclipStart=0;
    for(let fi=0;fi<nOps;fi++)drawOpSocial(ctx,fi,nOps,W,H,walkF,t,false,drawFn);
    if(isHot){
      const sx=W-38+Math.sin(walkF*0.011)*8;
      ctx.save();ctx.globalAlpha=0.28+Math.sin(walkF*0.013)*0.07;
      ctx.fillStyle="#8b0000";ctx.beginPath();ctx.ellipse(sx,H-16,16,44,0,0,Math.PI*2);ctx.fill();
      ctx.globalAlpha=0.8;ctx.fillStyle="#ff2200";
      ctx.beginPath();ctx.ellipse(sx-5,H-52,2.5,3.5,0.1,0,Math.PI*2);ctx.fill();
      ctx.beginPath();ctx.ellipse(sx+5,H-52,2.5,3.5,-0.1,0,Math.PI*2);ctx.fill();
      ctx.restore();
    }
  }
}

// ---- scene backgrounds ----
function drawBgLobby(ctx,W,H,bgOff,flickS){
  const bg=ctx.createLinearGradient(0,0,0,H);
  bg.addColorStop(0,"#1e1b00");bg.addColorStop(0.18,"#161500");bg.addColorStop(1,"#1a1800");
  ctx.fillStyle=bg;ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.06;ctx.strokeStyle="#d4b800";ctx.lineWidth=0.5;
  const ps=36,ox=-(bgOff%ps);
  for(let x=ox-ps;x<W+ps;x+=ps)for(let y=0;y<H+ps;y+=ps){
    ctx.beginPath();ctx.moveTo(x,y+ps/2);ctx.lineTo(x+ps/2,y);ctx.lineTo(x+ps,y+ps/2);ctx.lineTo(x+ps/2,y+ps);ctx.closePath();ctx.stroke();
  }
  ctx.restore();
  ctx.save();ctx.globalAlpha=0.1;ctx.strokeStyle="#c8a800";ctx.lineWidth=0.7;
  ctx.setLineDash([14,20]);ctx.lineDashOffset=-(bgOff*0.62)%34;
  ctx.beginPath();ctx.moveTo(0,H*0.22);ctx.lineTo(W,H*0.22);ctx.stroke();
  ctx.setLineDash([]);ctx.restore();
  ctx.fillStyle="#0d0c00";ctx.fillRect(0,0,W,10);
  ctx.fillStyle="#2a2500";ctx.fillRect(0,H-10,W,10);
  ctx.fillStyle="#100f00";ctx.fillRect(0,H-11,W,1);
  const lampSp=85,lampOff=-(bgOff%lampSp);let li=0;
  for(let lp=lampOff-lampSp;lp<W+lampSp;lp+=lampSp,li++){
    const fi=((Math.floor(bgOff/lampSp)+li)%8+8)%8;
    const fl=flickS[fi]||1;
    const cone=ctx.createRadialGradient(lp,10,1,lp,10,55);
    cone.addColorStop(0,"rgba(255,238,150,"+(0.17*fl)+")");cone.addColorStop(1,"rgba(255,238,100,0)");
    ctx.fillStyle=cone;ctx.beginPath();ctx.moveTo(lp-38,H);ctx.lineTo(lp+38,H);ctx.lineTo(lp+7,10);ctx.lineTo(lp-7,10);ctx.closePath();ctx.fill();
    ctx.fillStyle="rgba(255,240,200,"+(0.65*fl)+")";ctx.fillRect(lp-9,7,18,4);
    if(fl<0.55){ctx.fillStyle="rgba(255,180,0,0.85)";ctx.beginPath();ctx.arc(lp,10,1.5,0,Math.PI*2);ctx.fill();}
  }
}
function drawBgDark(ctx,W,H,bgOff,isCave){
  ctx.fillStyle=isCave?"#040402":"#050504";ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.035;ctx.strokeStyle=isCave?"#302010":"#403828";ctx.lineWidth=0.5;
  const ps=38,ox=-(bgOff%ps);
  for(let x=ox-ps;x<W+ps;x+=ps){ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,H);ctx.stroke();}
  ctx.restore();
  ctx.fillStyle=isCave?"#090806":"#0a0908";ctx.fillRect(0,0,W,8);
  ctx.fillStyle=isCave?"#070604":"#080706";ctx.fillRect(0,H-8,W,8);
  const t=Date.now()*0.001;
  ctx.save();ctx.globalAlpha=0.025*Math.abs(Math.sin(t*0.6));
  const g=ctx.createRadialGradient(W/2,0,0,W/2,0,H);
  g.addColorStop(0,"rgba(180,150,80,1)");g.addColorStop(1,"rgba(180,150,80,0)");
  ctx.fillStyle=g;ctx.fillRect(0,0,W,H);ctx.restore();
}
function drawBgPool(ctx,W,H,bgOff){
  const bg=ctx.createLinearGradient(0,0,0,H);
  bg.addColorStop(0,"#040a0e");bg.addColorStop(0.7,"#061218");bg.addColorStop(1,"#081a24");
  ctx.fillStyle=bg;ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.05;ctx.strokeStyle="#1a4a5a";ctx.lineWidth=0.5;
  const ts=22,ox=-(bgOff*0.5%ts);
  for(let x=ox-ts;x<W+ts;x+=ts){ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,H*0.7);ctx.stroke();}
  for(let y=0;y<H*0.7;y+=ts){ctx.beginPath();ctx.moveTo(0,y);ctx.lineTo(W,y);ctx.stroke();}
  ctx.restore();
  const wy=H*0.76;
  const wg=ctx.createLinearGradient(0,wy,0,H);
  wg.addColorStop(0,"rgba(8,36,62,0.9)");wg.addColorStop(1,"rgba(4,18,36,1)");
  ctx.fillStyle=wg;ctx.fillRect(0,wy,W,H-wy);
  ctx.save();ctx.globalAlpha=0.2;ctx.strokeStyle="#1e5070";ctx.lineWidth=0.7;
  const t=Date.now()*0.001;
  for(let i=0;i<5;i++){
    const rx=(bgOff*0.38+i*55)%W;
    ctx.beginPath();ctx.ellipse(rx,wy+4+i*3,18+Math.sin(t*1.1+i)*7,1.5,0,0,Math.PI*2);ctx.stroke();
  }
  ctx.restore();
  ctx.fillStyle="#020508";ctx.fillRect(0,0,W,8);
  ctx.save();ctx.globalAlpha=0.07;
  const cg=ctx.createLinearGradient(0,0,0,H*0.35);
  cg.addColorStop(0,"rgba(20,80,120,1)");cg.addColorStop(1,"rgba(20,80,120,0)");
  ctx.fillStyle=cg;ctx.fillRect(0,0,W,H*0.35);ctx.restore();
}
function drawBgEnd(ctx,W,H){
  ctx.fillStyle="#030302";ctx.fillRect(0,0,W,H);
  ctx.fillStyle="#050504";ctx.fillRect(0,0,W,8);
  ctx.fillStyle="#040403";ctx.fillRect(0,H-8,W,8);
  const dx=W*0.64,dy=H*0.62;
  ctx.fillStyle="#0c0c0a";ctx.fillRect(dx-24,dy,48,10);
  ctx.fillStyle="#0a0a08";ctx.fillRect(dx-3,dy+10,6,H-dy-10);
  const lx=dx+12,ly=dy-5;
  const lg=ctx.createRadialGradient(lx,ly,1,lx,ly,44);
  lg.addColorStop(0,"rgba(255,220,120,0.2)");lg.addColorStop(1,"rgba(255,220,120,0)");
  ctx.fillStyle=lg;ctx.fillRect(lx-44,ly-44,88,88);
  ctx.fillStyle="rgba(255,220,120,0.8)";ctx.beginPath();ctx.arc(lx,ly,2.2,0,Math.PI*2);ctx.fill();
  ctx.save();ctx.fillStyle="rgba(0,190,80,0.75)";ctx.font="bold 7px monospace";ctx.textAlign="right";
  ctx.fillText("EXIT →",W-6,H-11);ctx.restore();
}
function drawBgRFYL(ctx,W,H,bgOff,walkF){
  const bg=ctx.createLinearGradient(0,0,0,H);
  bg.addColorStop(0,"#0a0000");bg.addColorStop(1,"#110300");
  ctx.fillStyle=bg;ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.04;ctx.strokeStyle="#800000";ctx.lineWidth=0.5;
  const ps=30,ox=-(bgOff*2.2%ps);
  for(let x=ox-ps;x<W+ps;x+=ps){ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,H);ctx.stroke();}
  ctx.restore();
  for(let i=0;i<3;i++){
    const ex=(bgOff*3.8+i*120)%(W+80)-20;
    ctx.save();ctx.globalAlpha=0.15+i*0.05;
    ctx.fillStyle="#5a0000";ctx.beginPath();ctx.ellipse(ex,H-18,11,36,0,0,Math.PI*2);ctx.fill();
    ctx.restore();
  }
  ctx.fillStyle="#060002";ctx.fillRect(0,0,W,8);
  ctx.fillStyle="#0a0003";ctx.fillRect(0,H-8,W,8);
}
function drawBgElectric(ctx,W,H,bgOff,walkF){
  ctx.fillStyle="#040508";ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.07;ctx.fillStyle="#181828";
  const gx=((W*0.68+bgOff*0.08)%(W+40))-10;
  ctx.fillRect(gx,H*0.18,16,H*0.56);ctx.fillRect(gx+20,H*0.3,10,H*0.44);ctx.restore();
  const t=Date.now()*0.001;
  const arcAmt=Math.pow(Math.max(0,Math.sin(t*4.6)),5);
  if(arcAmt>0.25){
    const ax=W*0.74,ay=H*0.28;
    ctx.save();ctx.globalAlpha=arcAmt*0.85;ctx.strokeStyle="rgba(110,150,255,0.9)";ctx.lineWidth=0.8;
    ctx.beginPath();ctx.moveTo(ax,ay);
    for(let i=0;i<5;i++)ctx.lineTo(ax+rand(-7,7),ay+(i+1)*5);
    ctx.stroke();ctx.restore();
    ctx.save();ctx.globalAlpha=arcAmt*0.04;ctx.fillStyle="rgba(110,150,255,1)";ctx.fillRect(0,0,W,H);ctx.restore();
  }
  ctx.fillStyle="#020304";ctx.fillRect(0,0,W,8);
}
function drawBgHotel(ctx,W,H,bgOff,flickS){
  const bg=ctx.createLinearGradient(0,0,0,H);
  bg.addColorStop(0,"#130904");bg.addColorStop(1,"#1a0c06");
  ctx.fillStyle=bg;ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.05;ctx.strokeStyle="#a86040";ctx.lineWidth=0.5;
  const ps=24,ox=-(bgOff*0.4%ps);
  for(let x=ox-ps;x<W+ps;x+=ps)for(let y=0;y<H+ps;y+=ps){
    ctx.beginPath();ctx.arc(x+ps/2,y+ps/2,ps*0.34,0,Math.PI*2);ctx.stroke();
  }
  ctx.restore();
  const lampSp=68,lampOff=-(bgOff*0.5%lampSp);let li=0;
  for(let lp=lampOff-lampSp;lp<W+lampSp;lp+=lampSp,li++){
    const fl=(flickS[li%8]||1)*0.7+0.3;
    const cone=ctx.createRadialGradient(lp,10,1,lp,10,48);
    cone.addColorStop(0,"rgba(255,160,60,"+(0.14*fl)+")");cone.addColorStop(1,"rgba(255,100,20,0)");
    ctx.fillStyle=cone;ctx.beginPath();ctx.moveTo(lp-30,H);ctx.lineTo(lp+30,H);ctx.lineTo(lp+6,10);ctx.lineTo(lp-6,10);ctx.closePath();ctx.fill();
    ctx.fillStyle="rgba(255,170,70,"+(0.48*fl)+")";ctx.fillRect(lp-6,7,12,3);
  }
  ctx.fillStyle="#0a0403";ctx.fillRect(0,0,W,8);
  ctx.fillStyle="#150b05";ctx.fillRect(0,H-8,W,8);
}
function drawBgOffice(ctx,W,H,bgOff,flickS){
  const bg=ctx.createLinearGradient(0,0,0,H);
  bg.addColorStop(0,"#14120b");bg.addColorStop(1,"#1a1710");
  ctx.fillStyle=bg;ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.07;ctx.strokeStyle="#c8b888";ctx.lineWidth=0.5;
  const ts=20,ox=-(bgOff%ts);
  for(let x=ox-ts;x<W+ts;x+=ts){ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,14);ctx.stroke();}
  ctx.beginPath();ctx.moveTo(0,14);ctx.lineTo(W,14);ctx.stroke();ctx.restore();
  const lampSp=72,lampOff=-(bgOff%lampSp);let li=0;
  for(let lp=lampOff-lampSp;lp<W+lampSp;lp+=lampSp,li++){
    const fl=flickS[li%8]||1;
    ctx.fillStyle="rgba(225,215,175,"+(0.48*fl)+")";ctx.fillRect(lp-14,8,28,4);
    ctx.save();ctx.globalAlpha=0.07*fl;
    const g=ctx.createRadialGradient(lp,10,1,lp,10,H*0.7);
    g.addColorStop(0,"rgba(225,215,175,1)");g.addColorStop(1,"rgba(225,215,175,0)");
    ctx.fillStyle=g;ctx.fillRect(0,0,W,H);ctx.restore();
  }
  ctx.fillStyle="#0e0d08";ctx.fillRect(0,0,W,8);
  ctx.fillStyle="#201e15";ctx.fillRect(0,H-8,W,8);
}
function drawBgWarehouse(ctx,W,H,bgOff){
  const bg=ctx.createLinearGradient(0,0,0,H);
  bg.addColorStop(0,"#0e0f0f");bg.addColorStop(1,"#111313");
  ctx.fillStyle=bg;ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.04;ctx.strokeStyle="#505560";ctx.lineWidth=0.5;
  const bh=16,bw=32,ox=-(bgOff*0.4%bw);
  for(let y=0;y<H;y+=bh){
    const ro=(Math.floor(y/bh)%2)*bw*0.5;
    for(let x=ox-bw+ro;x<W+bw;x+=bw)ctx.strokeRect(x,y,bw,bh);
  }
  ctx.restore();
  const px=((W*0.64+bgOff*0.07)%(W+50))-10;
  ctx.save();ctx.globalAlpha=0.09;ctx.fillStyle="#2a2820";
  ctx.fillRect(px,H-30,30,15);ctx.fillRect(px+4,H-38,22,8);ctx.restore();
  ctx.fillStyle="#080808";ctx.fillRect(0,0,W,8);ctx.fillStyle="#181816";ctx.fillRect(0,H-8,W,8);
  const lx=W/2;
  ctx.save();ctx.globalAlpha=0.055;
  const lg=ctx.createRadialGradient(lx,8,1,lx,8,H*0.6);
  lg.addColorStop(0,"rgba(195,195,175,1)");lg.addColorStop(1,"rgba(195,195,175,0)");
  ctx.fillStyle=lg;ctx.fillRect(0,0,W,H);ctx.restore();
  ctx.fillStyle="rgba(195,195,175,0.45)";ctx.fillRect(lx-12,6,24,4);
}

function drawBase(ctx,W,H){
  // Static HQ scene (lobby style, no scroll)
  const bg=ctx.createLinearGradient(0,0,0,H);
  bg.addColorStop(0,"#1e1b00");bg.addColorStop(0.18,"#161500");bg.addColorStop(1,"#1a1800");
  ctx.fillStyle=bg;ctx.fillRect(0,0,W,H);
  ctx.save();ctx.globalAlpha=0.04;ctx.strokeStyle="#d4b800";ctx.lineWidth=0.5;
  const ps=36;
  for(let x=-ps;x<W+ps;x+=ps)for(let y=0;y<H+ps;y+=ps){
    ctx.beginPath();ctx.moveTo(x,y+ps/2);ctx.lineTo(x+ps/2,y);ctx.lineTo(x+ps,y+ps/2);ctx.lineTo(x+ps/2,y+ps);ctx.closePath();ctx.stroke();
  }
  ctx.restore();
  ctx.fillStyle="#0d0c00";ctx.fillRect(0,0,W,10);
  ctx.fillStyle="#2a2500";ctx.fillRect(0,H-10,W,10);
  const lx=W/2;
  const cone=ctx.createRadialGradient(lx,10,1,lx,10,55);
  cone.addColorStop(0,"rgba(255,238,150,0.12)");cone.addColorStop(1,"rgba(255,238,100,0)");
  ctx.fillStyle=cone;ctx.beginPath();ctx.moveTo(lx-38,H);ctx.lineTo(lx+38,H);ctx.lineTo(lx+7,10);ctx.lineTo(lx-7,10);ctx.closePath();ctx.fill();
  ctx.fillStyle="rgba(255,240,200,0.5)";ctx.fillRect(lx-9,7,18,4);
  // Idle ops standing at base — all of them, with personalities
  const idle=typeof G!=="undefined"?G.ops.filter(o=>o.status==="idle"):[];
  const n=idle.length;
  if(n>0){
    const spacing=Math.min(58,Math.max(32,(W-50)/Math.max(n-1,1)));
    const t=Date.now()*0.001;
    for(let i=0;i<n;i++){
      const p=_OPP[i%_OPP.length];
      const x=26+i*spacing;
      // At base: ops have slight fidget — weight shift, look around
      const fidget=Math.sin(t*0.6+i*2.1)*p.yo;
      // Some ops face each other at base (social pairs)
      const dir=(i%2===1&&i<n-1)?-1:1; // alternating for adjacent pairs
      drawOp(ctx,x,H-10+fidget,i*18,dir,true);
    }
  }
  // Research label
  if(typeof G!=="undefined"&&G.researchTarget){
    const t=TECH.find(x=>x.id===G.researchTarget);
    const pct=Math.round(G.researchProgress||0);
    ctx.save();ctx.globalAlpha=0.7;ctx.fillStyle="#4e7a9c";
    ctx.font="6.5px monospace";ctx.textAlign="center";
    ctx.fillText("▶ RESEARCHING: "+((t?t.name:"")).substring(0,18)+" ["+pct+"%]",W/2,H-60);
    ctx.restore();
  } else if(typeof G!=="undefined"&&n>0){
    ctx.save();ctx.globalAlpha=0.35;ctx.fillStyle="#8f8a72";
    ctx.font="6.5px monospace";ctx.textAlign="center";
    ctx.fillText("HQ · "+n+" idle",W/2,H-60);ctx.restore();
  }
}

function drawOp(ctx,x,baseY,walkF,dir,standing){
  ctx.save();ctx.translate(x,baseY);if(dir<0)ctx.scale(-1,1);
  const swing=standing?0:Math.sin(walkF*0.22)*6;
  const as=standing?0:Math.sin(walkF*0.22)*5;
  const skinColor='#c49060';
  const bodyColor='#3d5030';  // field jacket — dark olive
  const legColor='#1e1c14';   // cargo pants
  const shoeColor='#141210';  // boots
  const helmColor='#2a3020';  // tactical helmet
  const packColor='#241e14';  // field pack

  // Ground shadow
  ctx.fillStyle='rgba(0,0,0,0.2)';ctx.beginPath();ctx.ellipse(0,1,11,3,0,0,Math.PI*2);ctx.fill();

  // Backpack (behind body, drawn first)
  ctx.fillStyle=packColor;rrOp(ctx,-11,-25,7,12,3);ctx.fill();
  // Pack strap
  ctx.strokeStyle='rgba(60,48,28,0.9)';ctx.lineWidth=1.5;
  ctx.beginPath();ctx.moveTo(-6,-24);ctx.lineTo(-4,-16);ctx.stroke();

  // ── Legs ──
  const lLenL=9+swing*0.5,lLenR=9-swing*0.5;
  ctx.fillStyle=legColor;
  rrOp(ctx,-5,-8,5,lLenL,2);ctx.fill();
  rrOp(ctx, 1,-8,5,lLenR,2);ctx.fill();
  // Boots
  ctx.fillStyle=shoeColor;
  ctx.beginPath();ctx.ellipse(-2,-8+lLenL+1,4.5,2.5,0,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.ellipse( 4,-8+lLenR+1,4.5,2.5,0,0,Math.PI*2);ctx.fill();

  // ── Body (field jacket) ──
  ctx.fillStyle=bodyColor;rrOp(ctx,-6.5,-24,13,16,4);ctx.fill();
  // Jacket chest pocket
  ctx.fillStyle='rgba(255,255,255,0.05)';rrOp(ctx,-4,-23,8,5,3);ctx.fill();
  // PARSEC patch (tiny amber square)
  ctx.fillStyle='rgba(200,160,60,0.4)';rrOp(ctx,2,-22,4,3,1);ctx.fill();

  // ── Arms ──
  ctx.fillStyle=bodyColor;
  rrOp(ctx,-13,-21+as,7,4,2);ctx.fill();
  rrOp(ctx,  7,-21-as,7,4,2);ctx.fill();
  // Gloved hands
  ctx.fillStyle='#2a2218';
  ctx.beginPath();ctx.arc(-9.5,-19+as,2.5,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.arc(10.5,-19-as,2.5,0,Math.PI*2);ctx.fill();

  // ── Neck ──
  ctx.fillStyle=skinColor;rrOp(ctx,-2.5,-27,5,4,1);ctx.fill();

  // ── Head ──
  ctx.fillStyle=skinColor;ctx.beginPath();ctx.arc(0,-33,8,0,Math.PI*2);ctx.fill();

  // ── Helmet ──
  ctx.fillStyle=helmColor;rrOp(ctx,-8.5,-43,17,11,4);ctx.fill();
  // Helmet brim
  ctx.fillStyle='#1c1c10';rrOp(ctx,-10,-34,20,3,2);ctx.fill();
  // Headlamp — glows amber
  ctx.fillStyle='rgba(255,220,100,0.9)';ctx.beginPath();ctx.arc(6,-38,2.5,0,Math.PI*2);ctx.fill();
  // Lamp glow halo
  const glow=ctx.createRadialGradient(6,-38,1,6,-38,8);
  glow.addColorStop(0,'rgba(255,220,100,0.25)');glow.addColorStop(1,'rgba(255,220,100,0)');
  ctx.fillStyle=glow;ctx.beginPath();ctx.arc(6,-38,8,0,Math.PI*2);ctx.fill();

  // ── Face ──
  ctx.fillStyle='rgba(80,50,20,0.8)';
  ctx.beginPath();ctx.arc(-3,-33,1.5,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.arc( 3,-33,1.5,0,Math.PI*2);ctx.fill();

  ctx.restore();
}

function drawOpDark(ctx,x,baseY,walkF,dir,standing){
  // Flashlight cone (drawn before operative, behind them)
  ctx.save();ctx.translate(x,baseY);
  const as2=standing?0:Math.sin(walkF*0.22)*5;
  const coneDir=dir<0?-1:1;
  const cx2=coneDir*14,cy2=-20-as2;
  const fl2=ctx.createRadialGradient(cx2,cy2,1,cx2+coneDir*42,cy2-2,44);
  fl2.addColorStop(0,"rgba(210,190,130,0.22)");fl2.addColorStop(1,"rgba(210,190,130,0)");
  ctx.fillStyle=fl2;
  ctx.beginPath();
  ctx.moveTo(cx2,cy2);
  ctx.lineTo(cx2+coneDir*72,cy2-22);
  ctx.lineTo(cx2+coneDir*72,cy2+22);
  ctx.closePath();ctx.fill();ctx.restore();

  // Operative with very dark colours
  ctx.save();ctx.translate(x,baseY);if(dir<0)ctx.scale(-1,1);
  const swing=standing?0:Math.sin(walkF*0.22)*6;
  const as=standing?0:Math.sin(walkF*0.22)*5;
  const skinColor='#4a2c10';
  const bodyColor='#18190f';
  const legColor='#0e0d08';
  const shoeColor='#090808';
  const helmColor='#111508';
  const packColor='#0e0a06';

  ctx.fillStyle='rgba(0,0,0,0.25)';ctx.beginPath();ctx.ellipse(0,1,11,3,0,0,Math.PI*2);ctx.fill();
  ctx.fillStyle=packColor;rrOp(ctx,-11,-25,7,12,3);ctx.fill();
  ctx.strokeStyle='rgba(25,18,8,0.9)';ctx.lineWidth=1.5;
  ctx.beginPath();ctx.moveTo(-6,-24);ctx.lineTo(-4,-16);ctx.stroke();
  const lLenL=9+swing*0.5,lLenR=9-swing*0.5;
  ctx.fillStyle=legColor;
  rrOp(ctx,-5,-8,5,lLenL,2);ctx.fill();rrOp(ctx,1,-8,5,lLenR,2);ctx.fill();
  ctx.fillStyle=shoeColor;
  ctx.beginPath();ctx.ellipse(-2,-8+lLenL+1,4.5,2.5,0,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.ellipse(4,-8+lLenR+1,4.5,2.5,0,0,Math.PI*2);ctx.fill();
  ctx.fillStyle=bodyColor;rrOp(ctx,-6.5,-24,13,16,4);ctx.fill();
  ctx.fillStyle='rgba(255,255,255,0.02)';rrOp(ctx,-4,-23,8,5,3);ctx.fill();
  // Left arm
  ctx.fillStyle=bodyColor;rrOp(ctx,-13,-21+as,7,4,2);ctx.fill();
  ctx.fillStyle='#141008';ctx.beginPath();ctx.arc(-9.5,-19+as,2.5,0,Math.PI*2);ctx.fill();
  // Right arm + flashlight
  ctx.fillStyle=bodyColor;rrOp(ctx,7,-21-as,8,4,2);ctx.fill();
  ctx.fillStyle='#282820';ctx.fillRect(12,-23-as,8,3);
  ctx.fillStyle='rgba(210,190,130,0.7)';ctx.beginPath();ctx.arc(20,-21-as,1.5,0,Math.PI*2);ctx.fill();
  ctx.fillStyle=skinColor;rrOp(ctx,-2.5,-27,5,4,1);ctx.fill();
  ctx.fillStyle=skinColor;ctx.beginPath();ctx.arc(0,-33,8,0,Math.PI*2);ctx.fill();
  ctx.fillStyle=helmColor;rrOp(ctx,-8.5,-43,17,11,4);ctx.fill();
  ctx.fillStyle='#0c0c08';rrOp(ctx,-10,-34,20,3,2);ctx.fill();
  ctx.fillStyle='rgba(210,190,130,0.45)';ctx.beginPath();ctx.arc(6,-38,2,0,Math.PI*2);ctx.fill();
  ctx.fillStyle='rgba(60,38,12,0.7)';
  ctx.beginPath();ctx.arc(-3,-33,1.5,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.arc(3,-33,1.5,0,Math.PI*2);ctx.fill();
  ctx.restore();
}

function drawCat(ctx,x,baseY,ph,dir){
  ctx.save();ctx.translate(x,baseY);if(dir<0)ctx.scale(-1,1);
  ctx.strokeStyle="rgba(160,140,70,0.65)";ctx.fillStyle="rgba(40,38,0,0.88)";ctx.lineWidth=1;
  // Body
  ctx.beginPath();ctx.ellipse(0,-5,7,4,0,0,Math.PI*2);ctx.fill();ctx.stroke();
  // Head
  ctx.beginPath();ctx.arc(8,-8,4,0,Math.PI*2);ctx.fill();ctx.stroke();
  // Ears
  ctx.beginPath();ctx.moveTo(5,-11);ctx.lineTo(7,-15);ctx.lineTo(10,-11);ctx.stroke();
  ctx.beginPath();ctx.moveTo(10,-11);ctx.lineTo(12,-15);ctx.lineTo(14,-11);ctx.stroke();
  // Eyes
  const blink=Math.floor(ph*1.2)%9===0;
  if(blink){ctx.beginPath();ctx.moveTo(7,-8);ctx.lineTo(9,-8);ctx.stroke();}
  else{ctx.fillStyle="rgba(220,180,0,0.9)";ctx.beginPath();ctx.arc(8,-8,1.5,0,Math.PI*2);ctx.fill();}
  // Whiskers
  ctx.strokeStyle="rgba(200,180,80,0.4)";ctx.lineWidth=0.5;
  ctx.beginPath();ctx.moveTo(12,-8);ctx.lineTo(17,-7);ctx.stroke();
  ctx.beginPath();ctx.moveTo(12,-8);ctx.lineTo(17,-9);ctx.stroke();
  // Tail
  const tw=Math.sin(ph*1.8)*5;
  ctx.strokeStyle="rgba(160,140,70,0.65)";ctx.lineWidth=1;
  ctx.beginPath();ctx.moveTo(-7,-5);ctx.quadraticCurveTo(-14,-14+tw,-11,-20+tw);ctx.stroke();
  ctx.restore();
}

function drawWanderer(ctx,x,baseY,t){
  ctx.save();ctx.translate(x,baseY);ctx.scale(-1,1); // faces left toward the team
  const sway=Math.sin(t*0.35)*2;
  ctx.fillStyle="rgba(0,0,0,0.15)";ctx.beginPath();ctx.ellipse(0,1,10,3,0,0,Math.PI*2);ctx.fill();
  // Legs — standing
  ctx.fillStyle="#2e2820";
  rrOp(ctx,-5,-8,5,9,2);ctx.fill();rrOp(ctx,1,-8,5,9,2);ctx.fill();
  ctx.fillStyle="#1a1614";
  ctx.beginPath();ctx.ellipse(-2,2,4.5,2.5,0,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.ellipse( 4,2,4.5,2.5,0,0,Math.PI*2);ctx.fill();
  // Body — worn tan jacket (unmistakably not PARSEC kit)
  ctx.fillStyle="#7a6234";rrOp(ctx,-6.5,-24,13,16,4);ctx.fill();
  ctx.fillStyle="rgba(255,255,255,0.04)";rrOp(ctx,-4,-23,8,5,3);ctx.fill();
  // Arms — slightly raised, uncertain
  ctx.fillStyle="#7a6234";
  rrOp(ctx,-13,-21+sway,7,4,2);ctx.fill();rrOp(ctx,7,-21-sway,7,4,2);ctx.fill();
  ctx.fillStyle="#c8905a";
  ctx.beginPath();ctx.arc(-9.5,-19+sway,2.5,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.arc(10.5,-19-sway,2.5,0,Math.PI*2);ctx.fill();
  // Neck
  ctx.fillStyle="#c8905a";rrOp(ctx,-2.5,-27,5,4,1);ctx.fill();
  // Head — no helmet
  ctx.fillStyle="#c8905a";ctx.beginPath();ctx.arc(0,-33,8,0,Math.PI*2);ctx.fill();
  // Messy hair
  ctx.fillStyle="#2e1e0a";
  rrOp(ctx,-7,-42,6,6,3);ctx.fill();rrOp(ctx,2,-42,6,6,3);ctx.fill();rrOp(ctx,-2,-44,5,5,3);ctx.fill();
  // Eyes — wide, a bit lost
  ctx.fillStyle="#1e1008";
  ctx.beginPath();ctx.arc(-3.5,-33,2.2,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.arc( 3.5,-33,2.2,0,Math.PI*2);ctx.fill();
  ctx.fillStyle="rgba(255,255,255,0.55)";
  ctx.beginPath();ctx.arc(-4,-33.5,1,0,Math.PI*2);ctx.fill();
  ctx.beginPath();ctx.arc( 3,-33.5,1,0,Math.PI*2);ctx.fill();
  ctx.restore();
}
