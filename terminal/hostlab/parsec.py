#!/usr/bin/env python3
"""
PARSEC // Dispatch Terminal — Linux Terminal Edition
Full port of the PARSEC Initiative browser game. Run standalone or via: hostlab parsec
"""
import curses, json, math, random, time
from pathlib import Path

SAVE_FILE = Path.home() / ".hostlab" / "parsec.json"
FRAG_FILE = Path.home() / ".hostlab" / "parsec_frag.json"

# ═══════════════════════════════════════════════════ DATA ════════════════════

FIRST = ["Vance","Okoye","Reyes","Halloran","Sato","Mbeki","Kovac","Lindqvist",
         "Ferreira","Adeyemi","Novak","Petrov","Chen","Dubois","Aaltonen","Marek",
         "Iqbal","Sorensen","Romano","Bauer","Costa","Ng","Haddad","Ortega",
         "Yakimov","Thorne","Mwangi","Sundaram","Voss","Papadopoulos"]
LAST  = ["D.","M.","R.","K.","T.","L.","S.","V.","E.","N.",
         "J.","P.","A.","B.","O.","W.","C.","F.","H.","I."]
ENTITY_NAMES = [
    "a Hound","a Smiler","a Skin-Stealer","a Clump","a Death Moth swarm",
    "a Wretch","a Partygoer","something wearing a colleague's face",
    "a Bacterium bloom","a Faceling","a Hunting Frenzy","a Crawler",
    "a Blur","something that doesn't cast a shadow",
]
DREAD = [
    "The carpet was damp. It is always damp.",
    "Someone had written 'IT KNOWS' on the wall in almond water.",
    "The fluorescents hummed at a frequency that made two operatives weep.",
    "A door that wasn't on the map. It is on the map now.",
    "The team found their own footprints. Going the other way. Fresher than theirs.",
    "Three phones rang simultaneously. The same voice answered on all of them.",
    "The hum stopped for three seconds. Everyone remembers it differently.",
    "Operative count on exit matched count on entry. We are choosing to believe this.",
]
DISCOVERIES = [
    "Mapped a clean corridor — routing updated.",
    "Textbook extraction. Everyone out before the lights flickered.",
    "Quiet run. The hum stayed even the whole way.",
    "Someone left a cold thermos of coffee. No ill effects so far.",
    "A room of clocks, all stopped at different times.",
]
TRAITS = {
    "vet":   {"name": "Veteran",    "desc": "+2 starting Grit"},
    "lucky": {"name": "Lucky",      "desc": "+15% loot on their team"},
    "tough": {"name": "Tough",      "desc": "Half injury chance"},
    "path":  {"name": "Pathfinder", "desc": "Reduces noclip risk"},
    "swift": {"name": "Swift",      "desc": "-15% expedition duration for team"},
    "mule":  {"name": "Pack Mule",  "desc": "+20% AW yield for team"},
}
SPECS = {
    "scout":    {"name": "Scout",    "icon": "◈", "desc": "Team 20% faster, noclip -40%"},
    "enforcer": {"name": "Enforcer", "icon": "◆", "desc": "Injury risk -50%, +2 Grit vs entities"},
    "salvager": {"name": "Salvager", "icon": "⚙", "desc": "+30% credits & salvage yield"},
    "medic":    {"name": "Medic",    "icon": "✚", "desc": "Team recovery time -60%"},
}
LEVELS = [
  {"id":0, "name":"The Lobby",        "danger":.03, "time":30,  "unlock":0,
   "flav":"Endless yellow rooms, wet carpet, the hum of fluorescent lights.",
   "loot":{"aw":[14,22],"salvage":[1,4],"credits":[3,7],"data":[0,2]}},
  {"id":1, "name":"Habitable Zone",   "danger":.08, "time":55,  "unlock":0,
   "flav":"A concrete warehouse the size of grief. People have lived here.",
   "loot":{"aw":[6,12],"salvage":[5,11],"credits":[7,16],"data":[1,3]}},
  {"id":2, "name":"Pipe Dreams",      "danger":.15, "time":85,  "unlock":1,
   "flav":"Dark tunnels of groaning pipe. The temperature swings without reason.",
   "loot":{"aw":[4,9],"salvage":[9,18],"credits":[12,24],"data":[2,5]}},
  {"id":7, "name":"The Window",       "danger":.05, "time":70,  "unlock":4,  "code":"188",
   "flav":"A single endless room, soft brown light, one window onto nothing.",
   "loot":{"aw":[10,16],"salvage":[3,7],"credits":[16,30],"data":[2,4]}},
  {"id":3, "name":"The Office",       "danger":.22, "time":120, "unlock":3,
   "flav":"Cubicles to the horizon. The phones ring. Don't give your real name.",
   "loot":{"aw":[3,8],"salvage":[7,15],"credits":[22,42],"data":[3,7]}},
  {"id":9, "name":"The Pool Rooms",   "danger":.18, "time":100, "unlock":5,
   "flav":"Warm tiles, no exit on the first try. The water is the wrong temperature.",
   "loot":{"aw":[8,14],"salvage":[6,13],"credits":[18,36],"data":[2,6]}},
  {"id":4, "name":"Cave System",      "danger":.30, "time":175, "unlock":7,
   "flav":"Wet rock. Only what you carry. The dark is patient.",
   "loot":{"aw":[2,6],"salvage":[15,28],"credits":[26,50],"data":[4,9]}},
  {"id":10,"name":"Suburbs",          "danger":.26, "time":140, "unlock":8,
   "flav":"Identical streets, cold cars. The street lights follow you.",
   "loot":{"aw":[3,7],"salvage":[10,20],"credits":[30,58],"data":[3,8]}},
  {"id":12,"name":"The Parking Lot",  "danger":.21, "time":115, "unlock":6,
   "flav":"Concrete under strip lights. The exits are always behind you.",
   "loot":{"aw":[5,10],"salvage":[9,19],"credits":[18,38],"data":[2,5]}},
  {"id":16,"name":"The Void Lounge",  "danger":.25, "time":155, "unlock":8,
   "flav":"Comfortable armchairs, soft light, jazz from nowhere.",
   "loot":{"aw":[5,11],"salvage":[9,20],"credits":[26,50],"data":[3,8]}},
  {"id":18,"name":"The Greenhouse",   "danger":.18, "time":130, "unlock":6,
   "flav":"Glowing flora, bioluminescent and silent. Nothing appears hostile — yet.",
   "loot":{"aw":[10,22],"salvage":[7,15],"credits":[20,42],"data":[3,8]}},
  {"id":5, "name":"Terror Hotel",     "danger":.42, "time":240, "unlock":11,
   "flav":"A grand hotel run by things that smile too wide.",
   "loot":{"aw":[4,10],"salvage":[11,24],"credits":[48,88],"data":[6,12]}},
  {"id":11,"name":"Electrical Station","danger":.36,"time":200, "unlock":14,
   "flav":"Humming generators, catwalks over nothing. Ozone and fear.",
   "loot":{"aw":[2,5],"salvage":[18,34],"credits":[42,78],"data":[8,16]}},
  {"id":13,"name":"The Hub",          "danger":.33, "time":165, "unlock":10,
   "flav":"The convergence of many levels. Things use it as a highway.",
   "loot":{"aw":[3,7],"salvage":[13,25],"credits":[32,62],"data":[5,11]}},
  {"id":17,"name":"Sub-Basement ∞",   "danger":.37, "time":215, "unlock":12,
   "flav":"Stairs going down, always the same stairwell.",
   "loot":{"aw":[3,7],"salvage":[13,26],"credits":[36,68],"data":[5,11]}},
  {"id":6, "name":"The End",          "danger":.55, "time":330, "unlock":17,
   "flav":"A small office. A desk. A door marked EXIT.",
   "loot":{"aw":[6,14],"salvage":[20,36],"credits":[85,160],"data":[10,20]}},
  {"id":15,"name":"The Grey Rooms",   "danger":.40, "time":225, "unlock":15,
   "flav":"Featureless grey corridors, identical from every angle.",
   "loot":{"aw":[4,9],"salvage":[15,28],"credits":[44,82],"data":[6,14]}},
  {"id":14,"name":"The False Light",  "danger":.50, "time":270, "unlock":18,
   "flav":"Warm amber light from no visible source. Safety was the trap.",
   "loot":{"aw":[5,13],"salvage":[17,32],"credits":[58,108],"data":[8,16]}},
  {"id":19,"name":"Catacombs",        "danger":.44, "time":255, "unlock":16,
   "flav":"Stone corridors carved with text no one can read. Cold and patient.",
   "loot":{"aw":[2,6],"salvage":[17,34],"credits":[52,96],"data":[7,15]}},
  {"id":20,"name":"The Quiet Room",   "danger":.60, "time":370, "unlock":21,
   "flav":"One room. One desk. Someone who looks like you sits in the chair.",
   "loot":{"aw":[7,17],"salvage":[20,38],"credits":[95,175],"data":[11,23]}},
  {"id":8, "name":"Run For Your Life","danger":.72, "time":300, "unlock":24, "code":"!",
   "flav":"The moment you arrive, it is already chasing.",
   "loot":{"aw":[8,18],"salvage":[24,44],"credits":[120,220],"data":[14,28]}},
]
TECH = [
  {"id":"hydro",    "name":"Hydroponics Bay",    "tag":"facility",
   "desc":"+25% AW yield from all expeditions.",
   "cost":{"credits":35,"salvage":12,"data":3},   "req":[]},
  {"id":"gear1",    "name":"Reinforced Kit",     "tag":"safety",
   "desc":"-30% encounter severity, +12% loot.",
   "cost":{"credits":45,"salvage":18,"data":5},   "req":[]},
  {"id":"drones",   "name":"Recon Drones",       "tag":"speed",
   "desc":"Expeditions resolve 25% faster.",
   "cost":{"credits":60,"salvage":24,"data":7},   "req":[]},
  {"id":"med",      "name":"Field Medicine",     "tag":"safety",
   "desc":"Injured operatives recover 2× faster.",
   "cost":{"credits":50,"salvage":16,"data":6},   "req":[]},
  {"id":"logistics","name":"Logistics Convoy",   "tag":"facility",
   "desc":"Supply cost per expedition -40%.",
   "cost":{"credits":80,"salvage":30,"data":9},   "req":["hydro"]},
  {"id":"beacon",   "name":"Beacon Network",     "tag":"safety",
   "desc":"Noclipped operatives return injured, not lost.",
   "cost":{"credits":90,"salvage":28,"data":12},  "req":["drones"]},
  {"id":"cart1",    "name":"Cartography I",      "tag":"unlock",
   "desc":"Stabilises routing. Unlocks levels beyond Tier 2.",
   "cost":{"credits":85,"salvage":30,"data":11},  "req":["drones"]},
  {"id":"contain",  "name":"Containment Wing",   "tag":"facility",
   "desc":"+0.7¤/sec passive income, +400¤ cap.",
   "cost":{"credits":120,"salvage":44,"data":16}, "req":["hydro"]},
  {"id":"gear2",    "name":"Hardened Kit",       "tag":"safety",
   "desc":"A further -30% encounter severity.",
   "cost":{"credits":160,"salvage":60,"data":22}, "req":["gear1"]},
  {"id":"cart2",    "name":"Cartography II",     "tag":"unlock",
   "desc":"Maps the deepest routes. Unlocks all remaining levels.",
   "cost":{"credits":230,"salvage":88,"data":32}, "req":["cart1"]},
  {"id":"psi",      "name":"Psi-Screen Helmets", "tag":"safety",
   "desc":"-60% noclip chance.",
   "cost":{"credits":200,"salvage":70,"data":28}, "req":["beacon","gear2"]},
  {"id":"deep",     "name":"Deep Survey Protocol","tag":"unlock",
   "desc":"+25% loot from levels Tier 5 and above.",
   "cost":{"credits":320,"salvage":115,"data":50},"req":["cart2"]},
  {"id":"psych",    "name":"Psych Support",      "tag":"safety",
   "desc":"Idle operatives shed stress 2× faster.",
   "cost":{"credits":160,"salvage":55,"data":22}, "req":["med"]},
  {"id":"anomreg",  "name":"Anomaly Register",   "tag":"facility",
   "desc":"Anomaly study +80%, containment cap → 5.",
   "cost":{"credits":180,"salvage":64,"data":26}, "req":["contain"]},
]
FACILITY_DEFS = [
  {"id":"cap",      "name":"Storage Expansion","desc":"+50% all caps",
   "base":{"credits":25,"salvage":8},  "mult":1.6},
  {"id":"barracks", "name":"Barracks",          "desc":"+1 max operatives",
   "base":{"credits":40,"salvage":15}, "mult":1.7},
  {"id":"train",    "name":"Training Sims",     "desc":"+1 starting Grit",
   "base":{"credits":38,"data":5},     "mult":1.8},
]
ANOMALY_NAMES = ["Amber Shard","Humming Tile","Cold Lamp","Folded Document",
  "Still Clock","Warm Wall","Paperclip Chain","Resonant Pipe","Damp Note",
  "Yellow Coin","Fluorescent Tube","Locked Drawer","Corkboard Pin","Carpet Fragment"]
CONDITIONS = [
  {"id":"quiet", "label":"QUIET", "danger_mult":.70, "loot_mult":1.05,"weight":18},
  {"id":"clear", "label":"CLEAR", "danger_mult":1.0, "loot_mult":1.0, "weight":50},
  {"id":"active","label":"ACTIVE","danger_mult":1.25,"loot_mult":1.2, "weight":20},
  {"id":"hot",   "label":"HOT",   "danger_mult":1.5, "loot_mult":1.35,"weight":12},
]

# ═════════════════════════════════════════════════ STATE ═════════════════════

G: dict = {}
FRAG: dict = {"depth":0,"total_runs":0,"total_lost":0}
OP_SEQ: int = 1

def _load_frag():
    try:
        if FRAG_FILE.exists(): FRAG.update(json.loads(FRAG_FILE.read_text()))
    except Exception: pass

def _save_frag():
    try:
        FRAG_FILE.parent.mkdir(parents=True, exist_ok=True)
        FRAG_FILE.write_text(json.dumps(FRAG))
    except Exception: pass

def new_game():
    global G, OP_SEQ
    OP_SEQ = 1
    G.clear()
    G.update({
        "res":{"aw":120.0,"credits":50.0,"salvage":16.0,"data":1.0},
        "cap":{"aw":280.0,"credits":260.0,"salvage":160.0,"data":100.0},
        "ops":[],"active":[],"tech":{},"fac":{"cap":0,"barracks":0,"train":0},
        "max_ops":4,"feed":[],"last_tick":time.time(),
        "exp_seq":1,"clearance":0,"dry_until":None,"game_over":False,
        "runs":0,"lost":0,"auto_runs":0,"wanderers_found":0,"play_time":0.0,
        "end_cleared":False,"rfyl_cleared":False,"memorial":[],"tutorial_step":0,
        "contracts":[],"clean_streak":0,"anomalies":[],"anom_studied":0,
        "research_target":None,"research_progress":0.0,
        "intel":[],"achievements":{},"hub_cleared":False,
        "start_time":time.time(),
    })
    G["ops"].append(_mk_op()); G["ops"].append(_mk_op())
    _gen_contracts(); _gen_contracts()

def _mk_op():
    global OP_SEQ
    grit = 2 + G.get("fac",{}).get("train",0)
    trait = None
    if random.random() < 0.45:
        trait = random.choice(list(TRAITS.keys()))
        if trait == "vet": grit += 2
    op = {"id":OP_SEQ,"name":random.choice(FIRST)+" "+random.choice(LAST),
          "grit":grit,"xp":0,"cond":100,"status":"idle","recover":0,
          "trait":trait,"runs":0,"spec":None,"stress":0.0}
    OP_SEQ += 1
    return op

# ═══════════════════════════════════════════════ HELPERS ═════════════════════

def _rand(a,b): return random.randint(int(a),int(b))
def _pick(seq): return random.choice(seq)
def _clamp(v,a,b): return max(a,min(b,v))
def _fmt(n):
    n=int(n)
    if n>=10000: return f"{n/1000:.1f}k"
    return str(n)
def _has(t): return bool(G.get("tech",{}).get(t))
def _gain(r,amt):
    if amt>0:
        G["res"][r]=_clamp(G["res"][r]+amt,0,G["cap"][r])
def _can_afford(cost): return all(G["res"].get(k,0)>=v for k,v in cost.items())
def _spend(cost):
    if not _can_afford(cost): return False
    for k,v in cost.items(): G["res"][k]-=v
    return True
def _op_by_id(oid): return next((o for o in G["ops"] if o["id"]==oid),None)
def _lv(lid): return next((l for l in LEVELS if l["id"]==lid),None)
def _lv_tag(L): return f"LVL {L['code']}" if L.get("code") else f"LEVEL {L['id']}"
def _get_cond(cid): return next((c for c in CONDITIONS if c["id"]==cid),CONDITIONS[1])

def _log(msg,cls=""):
    t=time.localtime()
    stamp=f"{t.tm_hour:02d}:{t.tm_min:02d}:{t.tm_sec:02d}"
    G["feed"].insert(0,{"stamp":stamp,"msg":msg,"cls":cls})
    if len(G["feed"])>150: G["feed"].pop()

def _lv_unlocked(L):
    if G["clearance"]<L["unlock"]: return False
    c1={3,4,9,10,11,12,16,17,18}; c2={5,6,8,13,14,15,19,20}
    if L["id"] in c1 and not _has("cart1"): return False
    if L["id"] in c2 and not _has("cart2"): return False
    return True

def _roll_cond():
    tot=sum(c["weight"] for c in CONDITIONS); r=random.random()*tot
    for c in CONDITIONS:
        r-=c["weight"]
        if r<=0: return c["id"]
    return "clear"

def _supply_cost(L,n):
    per=max(2,round(L["time"]/30)+1)
    m=0.6 if _has("logistics") else 1.0
    return max(1,math.ceil(n*per*m))

def _speed_mult(ops):
    m=0.75 if _has("drones") else 1.0
    if any(o.get("trait")=="swift" for o in ops): m*=0.85
    if any(o.get("spec")=="scout" for o in ops): m*=0.8
    return m

def _sev_mult():
    m=1.0
    if _has("gear1"): m*=0.7
    if _has("gear2"): m*=0.7
    return m

def _research_rate():
    idle=sum(1 for o in G["ops"] if o["status"]=="idle")
    return 0.5+min(idle,4)*0.5

# ════════════════════════════════════════════ MECHANICS ══════════════════════

def dispatch(lid, op_ids, auto=False):
    L=_lv(lid)
    if not L or not _lv_unlocked(L): return False
    if len(G["active"])>=2: return False
    ops=[_op_by_id(oid) for oid in op_ids if _op_by_id(oid)]
    if not ops: return False
    cost=_supply_cost(L,len(ops))
    if G["res"]["aw"]<cost: return False
    G["res"]["aw"]-=cost
    for o in ops: o["status"]="deployed"
    dur=L["time"]*_speed_mult(ops)
    cid=_roll_cond()
    exp={"id":G["exp_seq"],"level":lid,"ops":[o["id"] for o in ops],
         "start":time.time(),"end":time.time()+dur,"duration":dur,
         "cond":cid,"auto":auto}
    G["exp_seq"]+=1; G["active"].append(exp)
    _log(f"Team → {L['name']} [{_get_cond(cid)['label']}] ~{int(dur)}s","report")
    return True

def _finish_resolve(exp):
    L=_lv(exp["level"])
    ops_alive=[_op_by_id(oid) for oid in exp["ops"] if _op_by_id(oid)]
    survivors=[o for o in ops_alive if o["status"]=="deployed"]
    cond=_get_cond(exp["cond"])
    team_grit=sum(o["grit"] for o in ops_alive)
    lm=(1.12 if _has("gear1") else 1.0)
    lm*=(1.25 if (L["id"]>=5 and _has("deep")) else 1.0)
    lm*=(1+team_grit*0.05)*cond["loot_mult"]
    lm+=sum(0.15 for o in ops_alive if o.get("trait")=="lucky")
    lm*=(0.5+0.5*len(survivors)/max(1,len(ops_alive)))
    idle_ops=[o for o in G["ops"] if o["status"]!="deployed"]
    avg_stress=sum(o.get("stress",0) for o in idle_ops)/max(1,len(idle_ops))
    lm*=(1-avg_stress*0.002)
    # intel boost
    intel=next((x for x in G.get("intel",[]) if x.get("level_id")==exp["level"] and x["expires"]>time.time()),None)
    if intel: lm*=1.25; G["intel"].remove(intel)
    salvager=any(o.get("spec")=="salvager" for o in survivors)
    haul={}
    for r in ["aw","salvage","credits","data"]:
        rng=L["loot"][r]; v=_rand(rng[0],rng[1]); m=lm
        if r=="aw":
            if any(o.get("trait")=="mule" for o in survivors): m*=1.2
            if _has("hydro"): m*=1.25
        if r in ("salvage","credits") and salvager: m*=1.3
        haul[r]=max(0,round(v*m))
        _gain(r,haul[r])
    if random.random()<0.07: b=_rand(20,60)+L["id"]*10; _gain("credits",b); haul["credits"]=haul.get("credits",0)+b
    if random.random()<0.12: _gain("aw",_rand(4,9))
    now=time.time(); injured_run=False
    for o in list(survivors):
        sev=L["danger"]*_sev_mult()*cond["danger_mult"]
        grit_eff=o["grit"]+(2 if o.get("spec")=="enforcer" else 0)
        tf=_clamp(1-(team_grit-len(ops_alive))*0.06-(len(ops_alive)-1)*0.05,0.22,1.0)
        risk=sev*tf*(1-grit_eff*0.04)
        if o.get("trait")=="tough": risk*=0.5
        if o.get("spec")=="enforcer": risk*=0.5
        risk*=(1+o.get("stress",0)*0.004)
        roll=random.random()
        if roll<risk*0.22 and L["danger"]>0.35:
            if _has("beacon"):
                o["status"]="injured"; o["cond"]=_clamp(o["cond"]-_rand(40,70),5,100)
                rt=_rand(60,120); rt=int(rt*(0.5 if _has("med") else 1))
                if any(x.get("spec")=="medic" for x in survivors if x is not o): rt=int(rt*0.4)
                o["recover"]=now+rt
                _log(f"{o['name']} noclipped — beacon recalled (injured).","bad")
            else:
                G["ops"].remove(o)
                G["memorial"].append({"name":o["name"],"detail":f"Lost in {L['name']}","runs":o["runs"]})
                G["lost"]+=1; FRAG["total_lost"]=FRAG.get("total_lost",0)+1
                _log(f"{o['name']} did not return from {L['name']}.","bad")
            injured_run=True
        elif roll<risk:
            o["status"]="injured"; o["cond"]=_clamp(o["cond"]-_rand(25,55),5,100)
            rt=_rand(40,95); rt=int(rt*(0.5 if _has("med") else 1))
            if any(x.get("spec")=="medic" for x in survivors if x is not o): rt=int(rt*0.4)
            o["recover"]=now+rt
            _log(f"{o['name']} injured in {L['name']}.","bad")
            injured_run=True
        else:
            o["status"]="idle"; o["cond"]=min(100,o["cond"]+5)
    if not injured_run: G["clean_streak"]=G.get("clean_streak",0)+1
    else: G["clean_streak"]=0
    xp_gain=4+L["id"]*2
    for o in [x for x in ops_alive if x in G["ops"]]:
        o["xp"]=o.get("xp",0)+xp_gain; o["runs"]=o.get("runs",0)+1
        o["stress"]=_clamp(o.get("stress",0)+_rand(10,22),0,100)
        while o["xp"]>=o["grit"]*12:
            o["xp"]-=o["grit"]*12; o["grit"]+=1
            if o["grit"]==4 and not o.get("spec"):
                o["_needs_spec"]=True
                _log(f"{o['name']} reached Grit 4 — ready for specialization.","achieve")
    G["runs"]+=1; FRAG["total_runs"]=FRAG.get("total_runs",0)+1
    G["clearance"]+=1+L["id"]//2
    _update_contracts(exp,haul)
    # anomaly chance
    anom_cap=5 if _has("anomreg") else 3
    if random.random()<0.22 and len(G["anomalies"])<anom_cap:
        used={a["name"] for a in G["anomalies"]}
        pool=[n for n in ANOMALY_NAMES if n not in used] or ANOMALY_NAMES
        G["anomalies"].append({"id":int(time.time()*1000),
            "name":_pick(pool),"level_id":L["id"],"progress":0.0,
            "reward":{"data":_rand(8,20)+L["id"],"credits":_rand(25,60)+L["id"]*4},
            "risk":0.08+L["danger"]*0.12})
    if L["id"]==6: G["end_cleared"]=True
    if L["id"]==8: G["rfyl_cleared"]=True
    if L["id"]==13: G["hub_cleared"]=True
    haul_str=" ".join(f"+{v}{k[0]}" for k,v in haul.items() if v>0)
    _log(f"← {L['name']}: {haul_str}","good")
    # noclip dread
    if random.random()<0.3: _log(_pick(DREAD),"dread")
    if random.random()<0.12: _log(_pick(DISCOVERIES),"report")
    if exp.get("auto"):
        still_alive=[oid for oid in exp["ops"] if _op_by_id(oid) and _op_by_id(oid)["status"]=="idle"]
        if still_alive: dispatch(exp["level"],still_alive,auto=True); G["auto_runs"]=G.get("auto_runs",0)+1
    G["active"].remove(exp)

def tick():
    if not G or G.get("game_over"): return
    now=time.time(); dt=_clamp(now-G["last_tick"],0,2.0); G["last_tick"]=now
    G["play_time"]=G.get("play_time",0)+dt
    G["res"]["aw"]=max(0,G["res"]["aw"]-len(G["ops"])*0.03*dt)
    if _has("contain"): _gain("credits",0.7*dt)
    # dry check
    if G["res"]["aw"]<=0.01:
        if G["dry_until"] is None:
            G["dry_until"]=now+35
            _log("CRITICAL: Almond water depleted. Site coherence failing.","bad")
        elif now>=G["dry_until"]:
            G["game_over"]=True; return
    else:
        G["dry_until"]=None
    for exp in list(G["active"]):
        if now>=exp["end"]: _finish_resolve(exp)
    for o in G["ops"]:
        if o["status"]=="injured" and now>=o.get("recover",0):
            o["status"]="idle"; o["cond"]=_clamp(o.get("cond",50)+40,0,100)
        if o["status"]=="idle":
            rate=3.0 if _has("psych") else 1.5
            o["stress"]=max(0.0,o.get("stress",0)-rate*dt)
            if o.get("stress",0)>=90 and random.random()<0.001*dt*25:
                o["status"]="injured"; o["recover"]=now+90
                o["stress"]=max(0,o["stress"]-30)
                _log(f"{o['name']} burned out — stand-down.","bad")
    if G.get("research_target"):
        G["research_progress"]=G.get("research_progress",0)+_research_rate()*dt
        if G["research_progress"]>=100:
            tid=G["research_target"]; G["tech"][tid]=True
            G["research_target"]=None; G["research_progress"]=0.0
            td=next((t for t in TECH if t["id"]==tid),None)
            _log(f"Research complete: {td['name'] if td else tid}.","achieve")
    idle_c=sum(1 for o in G["ops"] if o["status"]=="idle")
    ar=idle_c*0.4*(1.8 if _has("anomreg") else 1.0)
    for anom in list(G["anomalies"]):
        anom["progress"]=min(100.0,anom["progress"]+ar*dt)
        if anom["progress"]>=100:
            _gain("data",anom["reward"]["data"]); _gain("credits",anom["reward"]["credits"])
            if random.random()<anom["risk"]:
                idle_ops=[o for o in G["ops"] if o["status"]=="idle"]
                if idle_ops:
                    v=_pick(idle_ops); v["status"]="injured"; v["recover"]=now+_rand(30,60)
                    _log(f"Anomaly breach: {anom['name']} — {v['name']} injured.","bad")
            else:
                _log(f"Anomaly studied: {anom['name']}.","good")
            G["anomalies"].remove(anom); G["anom_studied"]=G.get("anom_studied",0)+1
    G["intel"]=[x for x in G.get("intel",[]) if x["expires"]>now]
    for ct in list(G.get("contracts",[])):
        if ct.get("expires",0)<now and not ct.get("done"): G["contracts"].remove(ct)
    while len(G.get("contracts",[]))<2: _gen_contracts()

def _gen_contracts():
    tier=max(0,G["runs"]//8)
    opts=[
        {"title":"Field Exercise","desc":f"Complete {2+tier//2} expeditions.",
         "gt":"runs","goal":2+tier//2,"reward":{"credits":22+tier*10,"salvage":4+tier*3}},
        {"title":"Water Detail",   "desc":f"Haul {18+tier*14} AW.",
         "gt":"aw","goal":18+tier*14,"reward":{"credits":18+tier*8,"data":2+tier*2}},
        {"title":"Salvage Drive",  "desc":f"Recover {10+tier*10} salvage.",
         "gt":"salvage","goal":10+tier*10,"reward":{"aw":12+tier*7,"data":2+tier*3}},
        {"title":"Data Harvest",   "desc":f"Recover {4+tier*3} data.",
         "gt":"data","goal":4+tier*3,"reward":{"credits":28+tier*12,"salvage":8+tier*5}},
    ]
    o=_pick(opts)
    G.setdefault("contracts",[]).append({
        "id":int(time.time()*1000+random.randint(0,999)),
        "title":o["title"],"desc":o["desc"],"goal_type":o["gt"],"goal":o["goal"],
        "progress":0,"reward":o["reward"],"done":False,"expires":time.time()+200})

def _update_contracts(exp,haul):
    for ct in G.get("contracts",[]):
        if ct.get("done"): continue
        gt=ct["goal_type"]
        if gt=="runs": ct["progress"]+=1
        elif gt=="aw": ct["progress"]+=haul.get("aw",0)
        elif gt=="salvage": ct["progress"]+=haul.get("salvage",0)
        elif gt=="data": ct["progress"]+=haul.get("data",0)
        if ct["progress"]>=ct["goal"]:
            ct["done"]=True; ct["progress"]=ct["goal"]
            for k,v in ct["reward"].items(): _gain(k,v)
            _log(f"Contract done: {ct['title']} — rewards paid.","achieve")

def hire():
    cost={"credits":20+G["runs"]}
    if len(G["ops"])>=G["max_ops"]: return False,"Barracks full."
    if not _can_afford(cost): return False,f"Need {_fmt(cost['credits'])}¤."
    _spend(cost); o=_mk_op()
    if random.random()<0.08:
        G["wanderers_found"]=G.get("wanderers_found",0)+1
        _log(f"Wanderer found: {o['name']}.","good")
    else:
        tr=(" ["+TRAITS[o["trait"]]["name"]+"]") if o.get("trait") else ""
        _log(f"Recruit: {o['name']} G{o['grit']}{tr}.","report")
    G["ops"].append(o); return True,None

def buy_tech(tid):
    if G.get("research_target"): return False,"Research in progress."
    t=next((t for t in TECH if t["id"]==tid),None)
    if not t or G["tech"].get(tid): return False,"Already researched."
    if not all(G["tech"].get(r) for r in t["req"]): return False,"Prereqs not met."
    if not _can_afford(t["cost"]): return False,"Insufficient resources."
    _spend(t["cost"]); G["research_target"]=tid; G["research_progress"]=0.0
    _log(f"Research started: {t['name']}.","report")
    return True,None

def buy_facility(fid):
    fd=next((f for f in FACILITY_DEFS if f["id"]==fid),None)
    if not fd: return False
    lvl=G["fac"].get(fid,0)
    cost={k:round(v*(fd["mult"]**lvl)) for k,v in fd["base"].items()}
    if not _spend(cost): return False
    G["fac"][fid]=lvl+1
    if fid=="cap":
        for k in G["cap"]: G["cap"][k]=round(G["cap"][k]*1.5)
    elif fid=="barracks": G["max_ops"]+=1
    _log(f"Facility: {fd['name']} → L{lvl+1}.","report")
    return True

def save_game():
    try:
        SAVE_FILE.parent.mkdir(parents=True,exist_ok=True)
        SAVE_FILE.write_text(json.dumps({"G":G,"OP_SEQ":OP_SEQ,"FRAG":FRAG},default=str))
        _save_frag(); return True
    except Exception: return False

def load_game():
    global G, OP_SEQ, FRAG
    try:
        if not SAVE_FILE.exists(): return False
        d=json.loads(SAVE_FILE.read_text())
        G.update(d.get("G",{})); OP_SEQ=d.get("OP_SEQ",1); FRAG.update(d.get("FRAG",{}))
        now=time.time()
        for o in G.get("ops",[]):
            if o["status"]=="deployed" and not any(o["id"] in e["ops"] for e in G.get("active",[])):
                o["status"]="idle"
            if o["status"]=="injured" and o.get("recover",0)<=now:
                o["status"]="idle"; o["cond"]=min(100,o.get("cond",50)+40)
        G["last_tick"]=now; G["dry_until"]=None
        return True
    except Exception: return False

# ═══════════════════════════════════════════════════ UI ══════════════════════

# curses color pairs
CA=1; CG=2; CR=3; CW=4; CD=5; CB=6   # amber, green, red, wall, dim, blue

UI={
    "lv_cursor":0,   # which unlocked level is highlighted
    "log_scroll":0,
    "status":"",
    "status_until":0,
    "last_save":0,
}

MODAL={"type":None,"data":{},"cursor":0}

def _cp(pair,bold=False,dim=False):
    a=curses.color_pair(pair)
    if bold: a|=curses.A_BOLD
    if dim:  a|=curses.A_DIM
    return a

def _put(win,y,x,s,attr=0):
    try:
        h,w=win.getmaxyx()
        if y<0 or y>=h or x<0: return
        s=str(s)[:max(0,w-x-1)]
        if s: win.addstr(y,x,s,attr)
    except curses.error: pass

def _bar(v,cap,w=10):
    pct=_clamp(v/max(1,cap),0,1)
    f=round(pct*w)
    return "█"*f+"░"*(w-f)

def _init_colors():
    curses.start_color(); curses.use_default_colors()
    curses.init_pair(CA,curses.COLOR_YELLOW,-1)
    curses.init_pair(CG,curses.COLOR_GREEN,-1)
    curses.init_pair(CR,curses.COLOR_RED,-1)
    curses.init_pair(CW,curses.COLOR_WHITE,-1)
    curses.init_pair(CD,curses.COLOR_WHITE,-1)
    curses.init_pair(CB,curses.COLOR_CYAN,-1)

def _render_left(win):
    h,w=win.getmaxyx(); row=0
    _put(win,row,0,f" ROSTER [{len(G['ops'])}/{G['max_ops']}]".ljust(w-1)[:w-1],_cp(CW,bold=True))
    row+=1
    for i,o in enumerate(G["ops"]):
        if row>=h-7: break
        st=o["status"]; sc=CG
        if st=="deployed": sc=CA; stxt="field"
        elif st=="injured": sc=CR; stxt=f"{max(0,int(o.get('recover',0)-time.time()))}s"
        else: stxt="idle"
        sp=SPECS[o["spec"]]["icon"] if o.get("spec") else ("·"+TRAITS[o["trait"]]["name"][:3] if o.get("trait") else "")
        name=o["name"][:13]
        _put(win,row,0,f" {name:<13}G{o['grit']}{sp}"[:w-1],_cp(CD if st=="injured" else 0))
        _put(win,row,w-6,f"{stxt:>5}",_cp(sc,dim=True))
        row+=1
        cpct=o.get("cond",100); cc=CR if cpct<30 else (CA if cpct<60 else CG)
        _put(win,row,1,f"[{_bar(cpct,100,8)}]",_cp(cc,dim=True))
        stress=o.get("stress",0)
        if stress>25:
            sc2=CR if stress>75 else CA
            _put(win,row,13,f"[{_bar(stress,100,5)}]",_cp(sc2,dim=True))
        row+=1
    hc=20+G["runs"]; hok=_can_afford({"credits":hc})
    _put(win,row,0,f" [H]ire ({_fmt(hc)}¤)",_cp(CA if hok else CD)); row+=2
    _put(win,row,0," FACILITY".ljust(w-1)[:w-1],_cp(CW,bold=True)); row+=1
    for fd in FACILITY_DEFS:
        if row>=h-1: break
        lvl=G["fac"].get(fd["id"],0)
        cost={k:round(v*(fd["mult"]**lvl)) for k,v in fd["base"].items()}
        cs=" ".join(f"{v}{k[0]}" for k,v in cost.items())
        can=_can_afford(cost)
        _put(win,row,0,f" {fd['name'][:12]} L{lvl} [{cs}]"[:w-1],_cp(CA if can else CD))
        row+=1

def _render_middle(win):
    h,w=win.getmaxyx(); now=time.time(); row=0
    # Active expeditions
    _put(win,row,0,f" ACTIVE [{len(G['active'])}/2]".ljust(w-1)[:w-1],_cp(CW,bold=True)); row+=1
    if not G["active"]:
        _put(win,row,1,"No teams in field.",_cp(CD)); row+=1
    else:
        for exp in G["active"]:
            L=_lv(exp["level"]); ops=[_op_by_id(oid) for oid in exp["ops"] if _op_by_id(oid)]
            pct=_clamp((now-exp["start"])/(exp["end"]-exp["start"]),0,1)
            eta=max(0,exp["end"]-now); cond=_get_cond(exp["cond"])
            names=",".join(o["name"].split()[0] for o in ops)
            cc=CG if exp["cond"]=="quiet" else (CR if exp["cond"]=="hot" else CA)
            _put(win,row,0,f" {L['name'][:15]} [{cond['label']}]"[:w-1],_cp(CA))
            _put(win,row,w-5,f"{int(eta):>3}s",_cp(CD,dim=True))
            row+=1
            bw=max(4,w-4); bf=round(pct*bw)
            prog="[" + "="*bf + (">" if bf<bw else "") + " "*(bw-bf-1) + "]"
            _put(win,row,0,prog[:w-1],_cp(CG if pct>0.8 else CA)); row+=1
    row+=1
    # Contracts
    _put(win,row,0," CONTRACTS".ljust(w-1)[:w-1],_cp(CW,bold=True)); row+=1
    for ct in G.get("contracts",[])[:3]:
        if row>=h-12: break
        done=ct.get("done",False)
        pct=min(1.0,ct["progress"]/max(1,ct["goal"]))
        cc=CG if done else CA
        _put(win,row,0,f" {'✓' if done else '○'} {ct['title']}"[:w-1],_cp(cc)); row+=1
        rew=" ".join(f"{v}{k[0]}" for k,v in ct["reward"].items())
        _put(win,row,2,f"{int(ct['progress'])}/{ct['goal']}  → {rew}"[:w-3],_cp(CD,dim=True))
        _put(win,row,w-int(pct*(w-4)-1)," " if pct<0.01 else "",0)
        row+=1
    row+=1
    # Levels list
    unlocked=[L for L in LEVELS if _lv_unlocked(L)]
    _put(win,row,0,f" LEVELS [{len(unlocked)}/{len(LEVELS)}]".ljust(w-1)[:w-1],_cp(CW,bold=True)); row+=1
    for i,L in enumerate(unlocked):
        if row>=h-2: break
        sel=(i==UI["lv_cursor"])
        danger_col=CR if L["danger"]>0.35 else (CA if L["danger"]>0.15 else CG)
        cur="▶" if sel else " "
        tag=_lv_tag(L)
        dstr=f"{int(L['danger']*100)}%"
        line=f"{cur}{tag:<9}{L['name'][:16]}"
        _put(win,row,0,line[:w-6],_cp(CA,bold=sel) if sel else _cp(CD))
        _put(win,row,w-5,f"{dstr:>4}",_cp(danger_col,dim=True))
        row+=1
    _put(win,row,0," ↑↓=select  D=dispatch  Enter=dispatch",_cp(CD,dim=True))

def _render_right(win):
    h,w=win.getmaxyx(); row=0
    # Research bar
    if G.get("research_target"):
        t=next((t for t in TECH if t["id"]==G["research_target"]),None)
        nm=t["name"][:14] if t else G["research_target"]
        prog=G.get("research_progress",0); eta=(100-prog)/max(0.1,_research_rate())
        _put(win,row,0,f" ▶ {nm} [{_bar(prog,100,10)}] {int(eta)}s"[:w-1],_cp(CB)); row+=1
    elif not any(G["tech"].get(t["id"]) for t in TECH if all(G["tech"].get(r) for r in t["req"])):
        _put(win,row,0," [R] Research tech available",_cp(CA,dim=True)); row+=1
    # Anomalies
    if G.get("anomalies"):
        _put(win,row,0,f" ANOMALY [{len(G['anomalies'])}]".ljust(w-1)[:w-1],_cp(CW,bold=True)); row+=1
        for a in G["anomalies"][:3]:
            if row>=h-3: break
            _put(win,row,0,f" {a['name'][:13]} [{_bar(a['progress'],100,8)}]"[:w-1],_cp(CA))
            row+=1
        row+=1
    # Feed
    _put(win,row,0," LOG".ljust(w-1)[:w-1],_cp(CW,bold=True)); row+=1
    feed=G.get("feed",[]); scroll=UI["log_scroll"]
    visible=feed[scroll:scroll+(h-row-1)]
    for e in visible:
        if row>=h-1: break
        cls=e.get("cls","")
        cc=CG if cls=="good" else (CR if cls=="bad" else (CA if cls in ("achieve","report") else CD))
        bold=(cls=="achieve")
        msg=e.get("msg",""); stamp=e.get("stamp","??:??:??")
        _put(win,row,0,f" {stamp} {msg}"[:w-1],_cp(cc,bold=bold,dim=(cls=="")))
        row+=1

def render(stdscr):
    h,w=stdscr.getmaxyx()
    if h<18 or w<60:
        stdscr.clear()
        _put(stdscr,0,0,"Terminal too small (need 60×18)",_cp(CR))
        stdscr.refresh(); return
    stdscr.clear()
    now=time.time()
    # Header
    dry=G.get("dry_until")
    if dry:
        remaining=max(0,dry-now)
        _put(stdscr,0,0,f" PARSEC · INITIATIVE  ▸ AW CRITICAL — {remaining:.0f}s  "[:w-1],_cp(CR,bold=True))
    else:
        _put(stdscr,0,0," PARSEC · INITIATIVE",_cp(CW,bold=True))
        _put(stdscr,0,22,f"  CLEARANCE {G['clearance']}",_cp(CA))
    res=G["res"]; cap=G["cap"]
    aw_col=CR if res["aw"]/max(1,cap["aw"])<0.15 else (CA if res["aw"]/max(1,cap["aw"])<0.5 else CG)
    rstr=(f"  🥛{_fmt(res['aw'])}/{_fmt(cap['aw'])}  "
          f"¤{_fmt(res['credits'])}  ⚙{_fmt(res['salvage'])}  ▦{_fmt(res['data'])}")
    _put(stdscr,1,0,rstr[:w-1],_cp(aw_col))
    # Status message
    if UI["status"] and now<UI["status_until"]:
        _put(stdscr,1,len(rstr)+2,UI["status"],_cp(CA,dim=True))
    # Panel layout
    lw=min(28,w//4); mw=min(36,w//3); rw=max(20,w-lw-mw-2)
    for row in range(2,h-1):
        try: stdscr.addch(row,lw,"│",_cp(CD))
        except curses.error: pass
        try: stdscr.addch(row,lw+mw+1,"│",_cp(CD))
        except curses.error: pass
    lwin=stdscr.derwin(h-3,lw,2,0)
    mwin=stdscr.derwin(h-3,mw,2,lw+1)
    rwin=stdscr.derwin(h-3,rw,2,lw+mw+2)
    for pw in (lwin,mwin,rwin): pw.erase()
    _render_left(lwin); _render_middle(mwin); _render_right(rwin)
    for pw in (lwin,mwin,rwin): pw.noutrefresh()
    # Footer
    dry_warn="▲AW!" if dry else ""
    footer=f" {dry_warn}[D]ispatch [H]ire [R]esearch [F]acility [S]ave [Q]uit  runs:{G['runs']} clr:{G['clearance']}"
    _put(stdscr,h-1,0,footer[:w-1],_cp(CD))
    # Modal
    if MODAL["type"]: _render_modal(stdscr,h,w)
    curses.doupdate()

def _render_modal(stdscr,h,w):
    mt=MODAL["type"]; md=MODAL["data"]
    mw=min(62,w-4); mh=min(22,h-4)
    mx=(w-mw)//2; my=(h-mh)//2
    win=curses.newwin(mh,mw,my,mx)
    win.erase()
    try: win.border()
    except curses.error: pass
    if mt=="level_pick":
        unlocked=[L for L in LEVELS if _lv_unlocked(L)]
        _put(win,0,2," SELECT LEVEL ",_cp(CA,bold=True))
        for i,L in enumerate(unlocked):
            if 2+i>=mh-2: break
            sel=(i==MODAL["cursor"])
            tag=_lv_tag(L); dc=CR if L["danger"]>0.35 else (CA if L["danger"]>0.15 else CG)
            cur="▶" if sel else " "
            _put(win,2+i,1,f"{cur}{tag:<9}{L['name'][:18]}  {int(L['danger']*100)}%"[:mw-2],
                _cp(CA,bold=True) if sel else _cp(CD))
        _put(win,mh-1,1,"↑↓=select  Enter=dispatch  Esc=cancel",_cp(CD,dim=True))
    elif mt=="dispatch":
        L=md.get("L"); sel=md.get("sel",[])
        _put(win,0,2,f" {_lv_tag(L)} · {L['name'][:28]} ",_cp(CA,bold=True))
        _put(win,1,1,L["flav"][:mw-3],_cp(CD,dim=True))
        _put(win,2,1,f"Danger {int(L['danger']*100)}%  Duration ~{L['time']}s  Supply: {_supply_cost(L,max(1,len(sel)))} AW",
             _cp(CR if L["danger"]>0.35 else CD))
        _put(win,3,1,"─"*(mw-2),_cp(CD,dim=True))
        idle=[o for o in G["ops"] if o["status"]=="idle"]
        if not idle:
            _put(win,4,2,"No idle operatives.",_cp(CR))
        else:
            _put(win,4,1,"Assign operatives (Space=toggle, A=all):",_cp(CW))
            for i,o in enumerate(idle):
                if 5+i>=mh-3: break
                chk="■" if o["id"] in sel else "□"
                cur="▶" if i==MODAL["cursor"] else " "
                tr=(" ["+TRAITS[o["trait"]]["name"][:4]+"]") if o.get("trait") else ""
                sp=(" "+SPECS[o["spec"]]["icon"]) if o.get("spec") else ""
                _put(win,5+i,1,f"{cur}{chk} {o['name'][:15]} G{o['grit']}{sp}{tr}"[:mw-2],
                    _cp(CA) if o["id"] in sel else _cp(CD))
        cost=_supply_cost(L,len(sel)); ok=G["res"]["aw"]>=cost and bool(sel)
        _put(win,mh-2,1,f"Supply: {cost} AW  {len(sel)} assigned  —  Enter=deploy  Esc=cancel",
             _cp(CG if ok else CR))
    elif mt=="spec":
        op=md.get("op"); specs=list(SPECS.items())
        _put(win,0,2," FIELD PROMOTION ",_cp(CA,bold=True))
        _put(win,2,2,f"{op['name']} reached Grit {op['grit']}.",_cp(CW))
        _put(win,3,2,"Choose specialization (permanent):",_cp(CD,dim=True))
        for i,(sid,s) in enumerate(specs):
            cur="▶" if i==MODAL["cursor"] else " "
            _put(win,5+i*2,1,f"{cur}{s['icon']} {s['name']}",_cp(CA,bold=(i==MODAL["cursor"])))
            _put(win,6+i*2,4,s["desc"][:mw-5],_cp(CD,dim=True))
        _put(win,mh-1,2,"↑↓=select  Enter=confirm",_cp(CD,dim=True))
    elif mt=="gameover":
        _put(win,0,2," CONTAINMENT LOST ",_cp(CR,bold=True))
        _put(win,3,2,md.get("reason","Site coherence failed.")[:mw-4],_cp(CR))
        _put(win,5,2,f"Runs: {G['runs']}  Lost: {G['lost']}  Clearance: {G['clearance']}",_cp(CD))
        opts=["▶ Reinitialise site","  Quit"]
        for i,o in enumerate(opts):
            _put(win,7+i,2,o,_cp(CA,bold=(i==MODAL["cursor"])) if i==MODAL["cursor"] else _cp(CD))
        _put(win,mh-1,2,"↑↓=select  Enter=confirm",_cp(CD,dim=True))
    elif mt=="facility":
        _put(win,0,2," FACILITY UPGRADES ",_cp(CW,bold=True))
        for i,fd in enumerate(FACILITY_DEFS):
            if 2+i*2>=mh-2: break
            lvl=G["fac"].get(fd["id"],0)
            cost={k:round(v*(fd["mult"]**lvl)) for k,v in fd["base"].items()}
            cs=" ".join(f"{v}{k[0]}" for k,v in cost.items())
            can=_can_afford(cost)
            cur="▶" if i==MODAL["cursor"] else " "
            _put(win,2+i*2,1,f"{cur}{fd['name']} L{lvl} [{cs}]"[:mw-2],_cp(CA if can else CD,bold=(i==MODAL["cursor"])))
            _put(win,3+i*2,3,fd["desc"],_cp(CD,dim=True))
        _put(win,mh-1,2,"↑↓=select  Enter=buy  Esc=cancel",_cp(CD,dim=True))
    elif mt=="tech_pick":
        avail=md.get("avail",[])
        _put(win,0,2," RESEARCH ",_cp(CB,bold=True))
        for i,t in enumerate(avail):
            if 2+i*2>=mh-2: break
            can=_can_afford(t["cost"])
            cur="▶" if i==MODAL["cursor"] else " "
            cs=" ".join(f"{v}{k[0]}" for k,v in t["cost"].items())
            _put(win,2+i*2,1,f"{cur}[{t['tag'][:3]}] {t['name'][:18]} {cs}"[:mw-2],
                _cp(CA if can else CD,bold=(i==MODAL["cursor"])))
            _put(win,3+i*2,4,t["desc"][:mw-5],_cp(CD,dim=True))
        _put(win,mh-1,2,"↑↓=select  Enter=research  Esc=cancel",_cp(CD,dim=True))
    win.refresh()

# ════════════════════════════════════════════════ INPUT ══════════════════════

def _status(msg):
    UI["status"]=msg; UI["status_until"]=time.time()+2.5

def handle_key(key):
    mt=MODAL["type"]
    if mt=="level_pick":
        unlocked=[L for L in LEVELS if _lv_unlocked(L)]
        if key==curses.KEY_UP: MODAL["cursor"]=max(0,MODAL["cursor"]-1)
        elif key==curses.KEY_DOWN: MODAL["cursor"]=min(len(unlocked)-1,MODAL["cursor"]+1)
        elif key in (10,13,curses.KEY_ENTER):
            if 0<=MODAL["cursor"]<len(unlocked):
                L=unlocked[MODAL["cursor"]]
                MODAL["type"]="dispatch"; MODAL["data"]={"L":L,"sel":[]}; MODAL["cursor"]=0
        elif key==27: MODAL["type"]=None
        return None
    if mt=="dispatch":
        md=MODAL["data"]; L=md.get("L"); sel=md.get("sel",[]); idle=[o for o in G["ops"] if o["status"]=="idle"]
        if key==curses.KEY_UP: MODAL["cursor"]=max(0,MODAL["cursor"]-1)
        elif key==curses.KEY_DOWN: MODAL["cursor"]=min(max(0,len(idle)-1),MODAL["cursor"]+1)
        elif key==ord(' '):
            if 0<=MODAL["cursor"]<len(idle):
                oid=idle[MODAL["cursor"]]["id"]
                if oid in sel: sel.remove(oid)
                else: sel.append(oid)
        elif key in (ord('a'),ord('A')): md["sel"]=[o["id"] for o in idle]
        elif key in (10,13,curses.KEY_ENTER):
            if sel and dispatch(L["id"],sel): MODAL["type"]=None
            elif not sel: _status("Select at least one operative.")
            else: _status("Not enough almond water.")
        elif key==27: MODAL["type"]=None
        return None
    if mt=="spec":
        specs=list(SPECS.keys()); op=MODAL["data"].get("op")
        if key==curses.KEY_UP: MODAL["cursor"]=max(0,MODAL["cursor"]-1)
        elif key==curses.KEY_DOWN: MODAL["cursor"]=min(len(specs)-1,MODAL["cursor"]+1)
        elif key in (10,13,curses.KEY_ENTER):
            if op and 0<=MODAL["cursor"]<len(specs):
                op["spec"]=specs[MODAL["cursor"]]; op["_needs_spec"]=False
                _log(f"{op['name']} → {SPECS[op['spec']]['name']}.","achieve"); MODAL["type"]=None
        return None
    if mt=="gameover":
        if key==curses.KEY_UP: MODAL["cursor"]=0
        elif key==curses.KEY_DOWN: MODAL["cursor"]=1
        elif key in (10,13,curses.KEY_ENTER):
            if MODAL["cursor"]==0: MODAL["type"]=None; new_game()
            else: return "quit"
        return None
    if mt=="facility":
        if key==curses.KEY_UP: MODAL["cursor"]=max(0,MODAL["cursor"]-1)
        elif key==curses.KEY_DOWN: MODAL["cursor"]=min(len(FACILITY_DEFS)-1,MODAL["cursor"]+1)
        elif key in (10,13,curses.KEY_ENTER):
            fd=FACILITY_DEFS[MODAL["cursor"]]
            if not buy_facility(fd["id"]): _status("Cannot afford.")
        elif key==27: MODAL["type"]=None
        return None
    if mt=="tech_pick":
        avail=MODAL["data"].get("avail",[])
        if key==curses.KEY_UP: MODAL["cursor"]=max(0,MODAL["cursor"]-1)
        elif key==curses.KEY_DOWN: MODAL["cursor"]=min(max(0,len(avail)-1),MODAL["cursor"]+1)
        elif key in (10,13,curses.KEY_ENTER):
            if 0<=MODAL["cursor"]<len(avail):
                ok,err=buy_tech(avail[MODAL["cursor"]]["id"])
                if ok: MODAL["type"]=None
                else: _status(err or "Cannot research.")
        elif key==27: MODAL["type"]=None
        return None
    # Global keys
    if key in (ord('q'),ord('Q')): save_game(); return "quit"
    if key in (ord('s'),ord('S')):
        if save_game(): _status("Saved.")
        else: _status("Save failed.")
    elif key in (ord('h'),ord('H')):
        ok,err=hire()
        if not ok: _status(err or "Cannot hire.")
    elif key in (ord('d'),ord('D'),10,13,curses.KEY_ENTER):
        if len(G["active"])>=2: _status("Max 2 concurrent expeditions."); return None
        if not any(o["status"]=="idle" for o in G["ops"]): _status("No idle operatives."); return None
        unlocked=[L for L in LEVELS if _lv_unlocked(L)]
        if not unlocked: _status("No levels accessible."); return None
        if key in (10,13,curses.KEY_ENTER):
            # dispatch with current level cursor
            L=unlocked[min(UI["lv_cursor"],len(unlocked)-1)]
            MODAL["type"]="dispatch"; MODAL["data"]={"L":L,"sel":[]}; MODAL["cursor"]=0
        else:
            MODAL["type"]="level_pick"; MODAL["cursor"]=UI["lv_cursor"]
    elif key in (ord('r'),ord('R')):
        if G.get("research_target"): _status("Research in progress."); return None
        avail=[t for t in TECH if not G["tech"].get(t["id"]) and all(G["tech"].get(r) for r in t["req"])]
        if not avail: _status("Nothing to research."); return None
        MODAL["type"]="tech_pick"; MODAL["data"]={"avail":avail}; MODAL["cursor"]=0
    elif key in (ord('f'),ord('F')):
        MODAL["type"]="facility"; MODAL["cursor"]=0
    elif key==curses.KEY_UP:
        UI["lv_cursor"]=max(0,UI["lv_cursor"]-1)
        UI["log_scroll"]=max(0,UI["log_scroll"]-1)
    elif key==curses.KEY_DOWN:
        unlocked=[L for L in LEVELS if _lv_unlocked(L)]
        UI["lv_cursor"]=min(max(0,len(unlocked)-1),UI["lv_cursor"]+1)
        UI["log_scroll"]=min(max(0,len(G["feed"])-5),UI["log_scroll"]+1)
    return None

# ══════════════════════════════════════════════ MAIN LOOP ════════════════════

def run_game(stdscr):
    curses.curs_set(0); stdscr.nodelay(True); stdscr.keypad(True)
    _init_colors(); _load_frag()
    if not load_game():
        new_game()
        _log("PARSEC Dispatch Terminal initialised.","report")
        _log("Dispatch to The Lobby first — recover almond water.","report")
    UI["last_save"]=time.time()
    last_render=0.0
    while True:
        now=time.time()
        if G and not G.get("game_over"):
            tick()
        # spec prompt
        if G and not MODAL["type"]:
            for o in G["ops"]:
                if o.get("_needs_spec"):
                    MODAL["type"]="spec"; MODAL["data"]={"op":o}; MODAL["cursor"]=0; break
        # game over
        if G and G.get("game_over") and not MODAL["type"]:
            MODAL["type"]="gameover"; MODAL["cursor"]=0
            MODAL["data"]={"reason":"Almond water depleted. Site coherence lost."}
        # autosave
        if now-UI["last_save"]>=8:
            save_game(); UI["last_save"]=now
        # render ~25fps
        if now-last_render>=0.04:
            render(stdscr); last_render=now
        key=stdscr.getch()
        if key!=-1:
            result=handle_key(key)
            if result=="quit": break
        time.sleep(0.02)

def main():
    curses.wrapper(run_game)

if __name__=="__main__":
    main()
