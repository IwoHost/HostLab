"use strict";
// ┌─ constants & data ─────────────────────────────────────────
"use strict";
const SAVE_KEY="parsec_save_v3";
const FIRST=["Vance","Okoye","Reyes","Halloran","Sato","Mbeki","Kovac","Lindqvist","Ferreira","Adeyemi","Novak","Petrov","Chen","Dubois","Aaltonen","Marek","Iqbal","Sorensen","Romano","Bauer","Costa","Ng","Haddad","Ortega","Yakimov","Thorne","Mwangi","Sundaram","Voss","Papadopoulos"];
const LAST=["D.","M.","R.","K.","T.","L.","S.","V.","E.","N.","J.","P.","A.","B.","O.","W.","C.","F.","H.","I."];
const ENTITY_NAMES=["a Hound","a Smiler","a Skin-Stealer","a Clump","a Death Moth swarm","a Wretch","a Partygoer","something wearing a colleague's face","a Bacterium bloom","a Faceling","a Hunting Frenzy","a Crawler","a Blur","something that doesn't cast a shadow"];
const DREAD=[
  "The wallpaper was warm. It is always warm in the newer sections.",
  "Someone had written 'NOT YET' on the wall in chalk. The advance team did not write it.",
  "The fluorescents hummed at a frequency that made two operatives stop and stand still without knowing why.",
  "A door that wasn't on the map. Someone had already labelled it UNIT 4.",
  "The team reports the ceiling tiles were warm to the touch. Like something lived above them.",
  "Audio log recovered: a family argument about curtains. No families deployed.",
  "Operative count on exit matched count on entry. The Sync requests we stop noting this.",
  "The hum changed pitch for three seconds. Everyone remembers it differently.",
  "A vending machine, recently restocked. The receipts inside are dated six weeks forward.",
  "The team found their own names on a list nailed to a door. Under the heading RESIDENTS.",
  "Three radios switched on simultaneously. The same voice: 'We're ready for the first wave.'",
  "An operative kept making plans aloud — furniture arrangement, route to work. They don't remember.",
  "The exit was exactly where the entrance was. The team marked it HOME before they noticed.",
  "The shadows fall toward something. Not the lights. Something else.",
  "Someone had laid out a welcome mat at a threshold no team had cleared. It faced inward.",
  "The corridor felt shorter on the way back. The team hopes that is a good sign.",
];
const DISCOVERIES=[
  "Mapped a clean corridor network — viable settlement throughway. Routing updated.",
  "Found a previous advance team's supply cache, seals intact. Tagged for settler use.",
  "Textbook clearance run. Sector documented before anything noticed them.",
  "An operative pocketed a humming coin. Filed under 'settler curios — examine before handover.'",
  "Found a stairwell connecting two isolated sectors. The Sync will want to reroute the access plan.",
  "A wall of sticky notes, all in the same handwriting, all reading 'you made it here.'",
  "Quiet sector. The hum stayed even the whole way. Marked as PRIORITY SETTLEMENT ZONE.",
  "Someone left a cold thermos on a shelf. Label read 'FOR THE FIRST ONES.' The team drank it.",
  "A room staged like a classroom. Chairs facing a blank wall. The chalk was fresh.",
  "The team found a welcome mat at the threshold. Not by any logged team. It faced inward.",
];

const TRAITS={
  vet:  {name:"Veteran",   desc:"+2 starting Grit."},
  lucky:{name:"Lucky",     desc:"+15% loot on their team."},
  tough:{name:"Tough",     desc:"Half the chance of injury."},
  path: {name:"Pathfinder",desc:"Rarely lost; reduces team noclip risk."},
  swift:{name:"Swift",     desc:"−15% expedition duration for their team."},
  mule: {name:"Pack Mule", desc:"+20% almond water yield for their team."},
};

const SPECS={
  scout:   {name:"Scout",    icon:"◈",desc:"Team runs 20% faster. Noclip risk −40%.",        color:"var(--green)"},
  enforcer:{name:"Enforcer", icon:"◆",desc:"Self injury risk −50%. +2 Grit vs encounters.", color:"var(--red)"},
  salvager:{name:"Salvager", icon:"⚙",desc:"Team earns +30% credits & salvage.",            color:"var(--amber)"},
  medic:   {name:"Medic",    icon:"✚",desc:"Team recovery time −60%.",                      color:"var(--blue)"},
};

function levelById(id){return LEVELS.find(l=>l.id===id);}
const LEVELS=[
  {id:0, name:"The Lobby",        danger:.03, time:30,  unlock:0,
   flav:"Endless yellow rooms, mono-yellow wallpaper, the wet smell of old carpet and the buzz of fluorescent lights. The first and the last place.",
   loot:{aw:[14,22],salvage:[1,4],credits:[3,7],data:[0,2]}},
  {id:1, name:"Habitable Zone",   danger:.08, time:55,  unlock:0,
   flav:"A concrete warehouse the size of grief. Pallets, forklifts, working sinks. People have lived here. Some still do.",
   loot:{aw:[6,12],salvage:[5,11],credits:[7,16],data:[1,3]}},
  {id:2, name:"Pipe Dreams",      danger:.15, time:85,  unlock:1,
   flav:"Dark tunnels of groaning pipe. The temperature swings without reason. Bring a light. Bring two.",
   loot:{aw:[4,9],salvage:[9,18],credits:[12,24],data:[2,5]}},
  {id:7, name:"The Window",       code:"188", danger:.05,time:70, unlock:4,
   flav:"A single endless room, soft brown light, and one window looking out on nothing in particular. People feel calm here. It may be the only kind level.",
   loot:{aw:[10,16],salvage:[3,7],credits:[16,30],data:[2,4]}},
  {id:3, name:"The Office",       danger:.22, time:120, unlock:3,
   flav:"Cubicles to the horizon. The phones ring. If you answer, do not give your real name.",
   loot:{aw:[3,8],salvage:[7,15],credits:[22,42],data:[3,7]}},
  {id:9, name:"The Pool Rooms",   danger:.18, time:100, unlock:5,
   flav:"Warm tiles, no exit on the first try. The water is the right colour but the wrong temperature. Sometimes it's higher than it was.",
   loot:{aw:[8,14],salvage:[6,13],credits:[18,36],data:[2,6]}},
  {id:4, name:"Cave System",      danger:.30, time:175, unlock:7,
   flav:"Wet rock, no fluorescents — only what you carry. The dark is patient and it is very, very old.",
   loot:{aw:[2,6],salvage:[15,28],credits:[26,50],data:[4,9]}},
  {id:10,name:"Suburbs",          danger:.26, time:140, unlock:8,
   flav:"Identical streets, identical houses. The cars are cold. Nobody is driving. The street lights turn on as you pass and won't turn off.",
   loot:{aw:[3,7],salvage:[10,20],credits:[30,58],data:[3,8]}},
  {id:5, name:"Terror Hotel",     danger:.42, time:240, unlock:11,
   flav:"A grand hotel run by things that smile too wide. The rooms are warm. The hospitality is the trap.",
   loot:{aw:[4,10],salvage:[11,24],credits:[48,88],data:[6,12]}},
  {id:11,name:"Electrical Station",danger:.36,time:200,unlock:14,
   flav:"Banks of humming generators, catwalks over nothing. The smell of ozone and fear. The power here goes somewhere. The team couldn't find where.",
   loot:{aw:[2,5],salvage:[18,34],credits:[42,78],data:[8,16]}},
  {id:6, name:"The End",          danger:.55, time:330, unlock:17,
   flav:"A small office at the end of everything. A desk. A door marked EXIT. No team has confirmed what is behind it.",
   loot:{aw:[6,14],salvage:[20,36],credits:[85,160],data:[10,20]}},
  {id:8, name:"Run For Your Life",code:"!", danger:.72,time:300,unlock:24,
   flav:"There is no exploring here. The moment you arrive, it is already chasing. Whatever you grab on the way out is all you get.",
   loot:{aw:[8,18],salvage:[24,44],credits:[120,220],data:[14,28]}},
  {id:12, name:"The Parking Lot", danger:.21, time:115, unlock:6,
   flav:"Endless concrete under humming strip lights. Cold cars that haven't moved in decades. The exits are always behind you.",
   loot:{aw:[5,10],salvage:[9,19],credits:[18,38],data:[2,5]}},
  {id:13, name:"The Hub", danger:.33, time:165, unlock:10,
   flav:"The convergence of many levels. Thousands of corridors branch outward from this point. Things use it as a highway. Do not stop moving.",
   loot:{aw:[3,7],salvage:[13,25],credits:[32,62],data:[5,11]}},
  {id:15, name:"The Grey Rooms", danger:.40, time:225, unlock:15,
   flav:"Featureless grey corridors, identical from every angle. The compass doesn't work. The team navigated entirely by memory.",
   loot:{aw:[4,9],salvage:[15,28],credits:[44,82],data:[6,14]}},
  {id:14, name:"The False Light", danger:.50, time:270, unlock:18,
   flav:"Warm amber light from no visible source. The team reported feeling inexplicably safe until they didn't.",
   loot:{aw:[5,13],salvage:[17,32],credits:[58,108],data:[8,16]}},
  {id:16, name:"The Void Lounge",    danger:.25, time:155, unlock:8,
   flav:"Comfortable armchairs, soft amber light, jazz from nowhere. The staff here know your team's names. You've never met them.",
   loot:{aw:[5,11],salvage:[9,20],credits:[26,50],data:[3,8]}},
  {id:17, name:"Sub-Basement ∞",     danger:.37, time:215, unlock:12,
   flav:"Concrete stairs going down, then down again — always the same stairwell. The loop is closed. Whatever lives down here has been circling for a very long time.",
   loot:{aw:[3,7],salvage:[13,26],credits:[36,68],data:[5,11]}},
  {id:18, name:"The Greenhouse",     danger:.18, time:130, unlock:6,
   flav:"An impossible span of glowing flora, bioluminescent and silent. The air is thick and the humidity is wrong. Nothing here appears hostile — one operative noted 'yet' in the margin.",
   loot:{aw:[10,22],salvage:[7,15],credits:[20,42],data:[3,8]}},
  {id:19, name:"Catacombs",          danger:.44, time:255, unlock:16,
   flav:"Stone corridors carved with text no one can read. Cold, constant, patient. The team was certain something walked behind them — always just outside the light.",
   loot:{aw:[2,6],salvage:[17,34],credits:[52,96],data:[7,15]}},
  {id:20, name:"The Quiet Room",     danger:.60, time:370, unlock:21,
   flav:"One room. One desk. One chair, occupied by someone who looks like you. A window onto the lobby — empty, but for a figure standing at the far end. The team did not stay long.",
   loot:{aw:[7,17],salvage:[20,38],credits:[95,175],data:[11,23]}},
];

const TECH=[
  {id:"hydro",    name:"Hydroponics Bay",       tag:"facility",desc:"Water reclamation loop. Boosts AW yield from all expeditions by +25%.",          cost:{credits:35,salvage:12,data:3},  req:[]},
  {id:"gear1",    name:"Reinforced Kit",         tag:"safety",  desc:"Field armour and beacons. −30% encounter severity, +12% loot.",                  cost:{credits:45,salvage:18,data:5},  req:[]},
  {id:"drones",   name:"Recon Drones",           tag:"speed",   desc:"Scout ahead. Expeditions resolve 25% faster.",                                    cost:{credits:60,salvage:24,data:7},  req:[]},
  {id:"med",      name:"Field Medicine",         tag:"safety",  desc:"Injured operatives recover twice as fast.",                                       cost:{credits:50,salvage:16,data:6},  req:[]},
  {id:"logistics",name:"Logistics Convoy",       tag:"facility",desc:"Pre-positioned caches. Supply cost per expedition −40%.",                        cost:{credits:80,salvage:30,data:9},  req:["hydro"]},
  {id:"beacon",   name:"Beacon Network",         tag:"safety",  desc:"Operatives lost to noclip are traced; return injured instead of gone.",          cost:{credits:90,salvage:28,data:12}, req:["drones"]},
  {id:"cart1",    name:"Cartography I",          tag:"unlock",  desc:"Stabilises deep routing. Unlocks levels beyond Tier 2.",                        cost:{credits:85,salvage:30,data:11}, req:["drones"]},
  {id:"contain",  name:"Containment Wing",       tag:"facility",desc:"License secured anomalies. +0.7 credits / sec, +400 credit cap.",               cost:{credits:120,salvage:44,data:16},req:["hydro"]},
  {id:"gear2",    name:"Hardened Kit",           tag:"safety",  desc:"Class-III suits. A further −30% encounter severity.",                            cost:{credits:160,salvage:60,data:22},req:["gear1"]},
  {id:"cart2",    name:"Cartography II",         tag:"unlock",  desc:"Maps the deepest routes. Required for The End and beyond.",                     cost:{credits:230,salvage:88,data:32},req:["cart1"]},
  {id:"psi",      name:"Psi-Screen Helmets",     tag:"safety",  desc:"−60% noclip chance. Some operatives refuse to wear them.",                       cost:{credits:200,salvage:70,data:28},req:["beacon","gear2"]},
  {id:"deep",     name:"Deep Survey Protocol",   tag:"unlock",  desc:"+25% loot from all levels Tier 5 and above.",                                    cost:{credits:320,salvage:115,data:50},req:["cart2"]},
  {id:"psych",   name:"Psychological Support",  tag:"safety",  desc:"Idle operatives shed stress 2× faster. Reduces long-run burnout.",                    cost:{credits:160,salvage:55,data:22}, req:["medkit"]},
  {id:"anomreg", name:"Anomaly Register",       tag:"facility",desc:"Anomaly study speed +80%. Containment capacity increases to 5.",                       cost:{credits:180,salvage:64,data:26}, req:["contain"]},
];

const FACILITY=[
  {id:"cap",     name:"Storage Expansion", desc:"Raise all resource caps by 50%.", base:{credits:25,salvage:8},  mult:1.6},
  {id:"barracks",name:"Barracks",          desc:"+1 max operatives on site.",       base:{credits:40,salvage:15}, mult:1.7},
  {id:"train",   name:"Training Sims",     desc:"New recruits start with +1 Grit.",base:{credits:38,data:5},     mult:1.8},
];

const LORE=[
  {id:"memo001",title:"The Sync — Settlement Directive 001",body:`THE SYNC — SETTLEMENT DIRECTIVE 001\nFROM: The Coordination Board, Settlement Authority\nTO: All PARSEC Advance Personnel\nDATE: [REDACTED]\n\nThe Backrooms are not an anomaly. They are a space. Spaces can be inhabited.\n\nThe Sync was formed for one purpose: to prepare the Backrooms for human settlement. The resource phase is over. The cartography phase is over. We are in the advance phase now.\n\nPARSEC — Pioneer Advance & Reconnaissance Settlement Corps — is the forward instrument of the Sync. Your job is not to survive the Backrooms. Your job is to make them survivable for the people who come after you.\n\nSome of you will not come back from that work. The Sync acknowledges this. It is in the contract. It is also, we believe, worth it.\n\nThe first wave of settlers is waiting.\n\n— The Coordination Board`},
  {id:"entity01",title:"Entity Hazard Briefing — Settler Safety",body:`ENTITY HAZARD BRIEFING — PARSEC ADVANCE TEAMS\nFor distribution to Forward Settlement Teams — Clearance 5+\n\nBefore any sector is approved for settler habitation, entity threats must be assessed and documented. Known active threats:\n\nPartygoers: high-speed ambush predators. Do not make eye contact. Run laterally, not away. SECTOR CANNOT BE SETTLED until population is reduced or rerouted.\n\nSkin-Stealers: mimics. Any settler showing behavioural deviation after entering a hot zone must be placed under immediate quarantine. This is non-negotiable.\n\nSmilers: ambient pressure. Sustained settler exposure is contraindicated. Sectors with confirmed Smiler presence are SUSPENDED pending study.\n\nFacelings: generally passive. Limited coexistence with human settlement is theoretically viable. Do not provoke.\n\nNOTE: No sector is cleared for settler intake until entity density drops below AMBER threshold. The people we bring in are not trained for this. That is why we are here first.`},
  {id:"awrep01",title:"Almond Water — Settler Supply Brief",body:`ALMOND WATER — SETTLER SUPPLY SUMMARY\nPARSEC Resource Division\n\nThe settlers need to understand this before they arrive.\n\nAlmond water is not optional. It is not a comfort supply. It is a metabolic requirement for sustained Backrooms habitation. Prolonged absence causes rapid cognitive and physical deterioration.\n\nRecommended daily intake: 0.4L per adult. 0.6L for elderly settlers and children.\n\nThe advance base exists, in part, to ensure sufficient water is stockpiled before the first wave arrives. Every PARSEC expedition's primary mandate includes almond water survey and reserve building.\n\nDo not authorise settler intake for any sector with less than 30 days' water in reserve.\n\nWe have made this mistake before. We are not making it again.\n\n— Resource Division, The Sync`},
  {id:"journal1",title:"Pioneer Journal — K. Morrow, First Wave",body:`Day 1: The briefing said it would smell like carpet. They were right. It smells exactly like carpet.\n\nDay 4: I understand why the Sync wants people to live here. It's so quiet. After everything outside, it is so, so quiet.\n\nDay 9: I'm meant to be assessing livability. I keep writing down things I'd want to change. Better lighting. Something growing. Ventilation that doesn't hum like someone thinking.\n\nDay 14: New sector. The wallpaper was peeling in spirals. Someone had written names under the strips. I checked for mine. I didn't find it. I'm not sure what to make of that.\n\nDay 21: The advance report is filed. Sector B-12 is provisionally habitable. I wrote that. I meant it.\n\nDay 22: I've requested reassignment to the non-advance programme. The Sync approved it.\n\nI won't explain why. The report stands.\n\n— K.M., PARSEC First Wave`},
  {id:"level0s",title:"Level 0 — Settlement Viability Assessment",body:`LEVEL 0 — THE LOBBY — SETTLEMENT VIABILITY ASSESSMENT\n\nPreliminary rating: VIABLE (conditional)\n\nEnvironment: Infinite office-style corridors. Fluorescent hum, consistent. Yellow-beige wallpaper throughout. Wet carpet (source undetermined — may be manageable with drainage infrastructure).\n\nEntity density: VERY LOW. Suitable for early settler intake with standard precautions.\n\nInfrastructure potential: HIGH. Level 0's stable geometry and low entity count make it the strongest candidate for the Sync's first permanent settlement cluster.\n\nWater: Vending machines present and functional (~40% operational). Natural spring suspected in sector 7-C. Survey ongoing.\n\nRecommendation: Begin settlement preparation. Designate as Settlement Zone Alpha.\n\nNote from the advance lead: Level 0 is not calm because it is safe. It is calm because something in here is choosing to let it be calm. I recommend we factor this into the long-term settlement plan.\n\n— Advance Lead, Carto Team 1`},
  {id:"incid01",title:"Incident Report — Expedition 44-C",body:`SYNC INCIDENT REPORT — Expedition 44-C\nSector: The Pool Rooms\nSettlement Status: SUSPENDED\n\nTeam entered sector for advance settlement assessment. At 90 minutes, operative Vance reported the pool water level was rising. Confirmed: 11cm rise in 8 minutes.\n\nExtraction was ordered. Vance did not comply.\n\n[Helmet cam transcript]\nVance: "I know this place. I've been here before."\nOps. Chen: "Vance, we're going."\nVance [to camera]: "Bring the first wave here. They'll love it."\n\nVance was not recovered.\n\nSettlement Status: Pool Rooms reclassified to SUSPENDED. Advance teams may continue resource gathering but settler clearance is denied pending further assessment.\n\nNote: We have not been able to determine what Vance meant by 'they'll love it.'`},
  {id:"trans01",title:"Cryptic Transmission — Signal Log",body:`[SIGNAL RECEIVED ON SYNC COMM BAND — SOURCE UNLOCATED]\n\n...if the Sync is listening, I need you to understand something before you send the settlers.\n\nThe route through The End isn't an exit. It's a reset. You come back. You don't know you came back.\n\nThe desk is always there. The papers describe whoever reads them. The EXIT sign is always lit.\n\nWe were the fourth advance team. We found the third team's notes. They found the second team's notes. I don't know who left the first set.\n\nThe settlement plans were on the desk. They were good plans. I recognised the handwriting.\n\nDon't send the settlers until you understand what's behind that door.\n\n[END OF TRANSMISSION — 14 min, 29 sec — ORIGIN: UNKNOWN]`},
  {id:"endrep1",title:"The End — Settlement Status: HOLD",body:`LEVEL 6 — "THE END" — SETTLEMENT STATUS: HOLD\n\nA small office. One desk. One lamp. Papers on the desk describe the office — and whoever is reading them. Page 7 is redacted in every copy.\n\nOne door. Marked EXIT. All recording equipment fails within 4 metres of it.\n\nThree PARSEC advance teams have passed through. All three returned to the lobby threshold with no memory of the other side.\n\nThe Sync's position: The End is not a settlement zone. It is a boundary.\n\nCurrent working theory: it is not the end of the Backrooms. It is the end of this section of the Backrooms. What lies beyond is the next advance team's problem.\n\nWe are still working on what that means for the settlement programme.`},
  {id:"awshort",title:"Almond Water Shortage — Emergency Protocol",body:`SYNC PROTOCOL — ADVANCE BASE SUPPLY FAILURE\n\nIf almond water reserves fall below CRITICAL threshold, apply immediately:\n\n1. ALL non-essential operations suspended\n2. No new advance runs until reserves exceed 20%\n3. Level 0 priority dispatch — fastest water recovery\n4. Sync Authority: mandatory extraction of all field teams\n\nThis is not an administrative protocol. This is survival.\n\nWithout a minimum water reserve, the advance base cannot sustain personnel. Without the advance base, there is no path forward for settlement. Without settlement, the first wave never arrives.\n\nWe are the only thing standing between them and a closed door.\n\nDo not let the water run out.`},
  {id:"pers001",title:"Personal Note — 'The Window'",body:`This is not an official document.\n\nI found a window in Level 11. That shouldn't be possible — Level 11 is underground. There's nothing outside a window here.\n\nThere was a street. I know that street. It's the street I grew up on. The sun was setting. My mother was in the yard.\n\nI was in there for six hours before Reyes pulled me out. I don't remember any of it except the window.\n\nI've requested reassignment to administrative. The Sync approved it.\n\nThe settlement feasibility report for Level 11 is on the shared drive. I filled it out. I recommended VIABLE.\n\nI meant it. I just can't be the one who goes back.\n\n— R.`},
  {id:"frag001",title:"Fragment Survey Order — The Sync",body:`THE SYNC — COORDINATION BOARD — INTERNAL ORDER\nFROM: The Board\nTO: PARSEC Advance Command, Cartography Division\n\nThe facility is larger than projected.\n\nEvery advance team that pushed past The End found a new section. New wallpaper. New hum. New entity types. Structural analysis suggests this is not drift or decay — the Backrooms extend in a direction we do not yet have a name for.\n\nWe are calling these new regions Fragments. Each is a new operational environment. Each is, for all purposes, a new facility waiting for a settlement programme.\n\nThe protocol: when an advance base is fully operational and a Fragment boundary is identified, we commission a new advance team. We take one person who knows the path. We prepare the next base.\n\nThe settlers will follow when the path is ready.\n\nThe operatives who agree to lead Fragment advances have been told they may not recognise themselves when they arrive. Every one of them agreed to go.\n\n— The Coordination Board`},
  {id:"stress01",title:"Psychological Assessment — Advance Teams",body:`PARSEC MEDICAL — PSYCHOLOGICAL DIVISION\n\nAfter reviewing 34 advance operatives across 8 deployments, we have identified a pattern we are calling Cumulative Exposure Syndrome.\n\nSymptoms by stage:\n— Stage 1: Heightened alertness, mild sleep disruption, increased productivity (paradoxically)\n— Stage 2: Intrusive geometric visualisation, discomfort in normal lighting, social withdrawal\n— Stage 3: Inability to distinguish internal monologue from external sound. Operatives begin making unprompted settlement plans.\n— Stage 4: Full dissociation. Operative requires immediate extraction.\n\nWe recommend mandatory downtime after high-exposure runs.\n\nNote: Stage 2 and above operatives are statistically more effective in the field. We have chosen not to include this finding in the brief we send to the Sync.\n\n— Dr. Vásquez, PARSEC Medical`},
  {id:"anom001",title:"Anomaly Log — Object Class: Amber",body:`ANOMALY LOG #0044\nObject designation: Amber Shard\nRecovered from: Level 0, corridor 7-C\nRetrieved by: Operative Mwangi, K.\n\nPhysical description: Crystalline fragment, warm to the touch. Does not cool. Emits a faint hum at 38Hz (confirmed via spectrometer). Casts no shadow.\n\nBehavioural notes:\n— Six researchers have reported feeling "watched" in the study room.\n— One researcher left the shard on their desk overnight. The next morning, it had moved 3.4cm. No one entered the room.\n— Mwangi refuses to enter the containment lab since retrieval. Will not explain why.\n\nSettlement relevance: Unknown. Recommend study before clearing any sector as settler-safe where similar objects are found.\n\nUpdate: the hum has changed pitch. We are recalibrating.`},
  {id:"parking",title:"The Parking Lot — Settlement Assessment",body:`LEVEL 12 — THE PARKING LOT — SETTLEMENT VIABILITY ASSESSMENT\n\nRating: VIABLE (medium-term)\n\nEnvironment: Concrete multi-storey structure of indeterminate height. Vehicles present (1968-2001, approximately). All cold. All keys in the ignition.\n\nOne vehicle had a receipt dated two days ago.\n\nNavigation: The exits loop. We walked exit C-7 three times before realising. The Sync will need to fund directional infrastructure before settler intake.\n\nEntity density: LOW. Footsteps with no visible source have been reported. The footsteps sometimes stop when addressed. This is noted as curious, not threatening.\n\nHazard rating: moderate. Easier than it looks, stranger than it seems. Recommended for settler intake in second-wave expansion.\n\n— Carto Team 4`},
  {id:"hub001",title:"The Hub — Advance Note",body:`This is not a formal report. I don't know how to write what I saw at The Hub as one.\n\nThe cartography team can map corridors. I can describe what a corridor looks like. Neither of us can explain why standing at that intersection felt like standing in front of something that was considering our paperwork.\n\nThings use The Hub as a transit point. We watched from above for forty minutes. Most moved like things that had learned to move by watching people. Close enough that you'd accept it at distance.\n\nOne stopped in the intersection. Stood still for six minutes. Then it looked up.\n\nWe had been very quiet.\n\nThe Hub is not ready for settlers. It may never be. But I filed my report as DEFERRED rather than DENIED. I want to believe there's an answer.\n\n— Unsigned (PARSEC, Clearance 8+)`},
];

const ACHIEVEMENTS=[
  {id:"first_run",  name:"First Contact",       desc:"Complete your first expedition.",           check:g=>g.runs>=1},
  {id:"ten_runs",   name:"Routine",             desc:"Complete 10 expeditions.",                  check:g=>g.runs>=10},
  {id:"fifty_runs", name:"Veteran Outfit",      desc:"Complete 50 expeditions.",                  check:g=>g.runs>=50},
  {id:"no_losses",  name:"Nobody Left Behind",  desc:"Reach 20 runs without losing anyone.",      check:g=>g.runs>=20&&g.lost===0},
  {id:"full_roster",name:"Full House",          desc:"Have 6 operatives on site at once.",        check:g=>g.ops.length>=6},
  {id:"grit5",      name:"Iron Core",           desc:"Have an operative reach Grit 5.",           check:g=>g.ops.some(o=>o.grit>=5)},
  {id:"grit8",      name:"Unbreakable",         desc:"Have an operative reach Grit 8.",           check:g=>g.ops.some(o=>o.grit>=8)},
  {id:"all_tech",   name:"Fully Equipped",      desc:"Research all technologies.",               check:g=>TECH.every(t=>g.tech[t.id])},
  {id:"the_end",    name:"Exit Found?",         desc:"Complete The End.",                         check:g=>!!g.endCleared},
  {id:"rfyl",       name:"Beyond The Map",      desc:"Complete Run For Your Life.",               check:g=>!!g.rfylCleared},
  {id:"credits1k",  name:"Flush",               desc:"Hold 1,000 credits at once.",              check:g=>g.res.credits>=1000},
  {id:"aw500",      name:"Deep Reserve",        desc:"Hold 500 almond water at once.",           check:g=>g.res.aw>=500},
  {id:"auto5",      name:"Set It, Forget It",   desc:"Run 5 auto-repeat expeditions.",           check:g=>(g.autoRuns||0)>=5},
  {id:"wanderer",   name:"They Know The Walls", desc:"Successfully recruit a wanderer.",         check:g=>(g.wanderersFound||0)>=1},
  {id:"survivor",   name:"Still Standing",      desc:"Accumulate 30 minutes of play time.",      check:g=>(g.playTime||0)>=1800},
  {id:"stressed",    name:"Fraying",          desc:"Have an operative reach 80 stress.",           check:g=>g.ops.some(o=>(o.stress||0)>=80)},
  {id:"zen",         name:"Clear Heads",      desc:"Have all operatives at zero stress.",           check:g=>g.ops.length>0&&g.ops.every(o=>(o.stress||0)===0)},
  {id:"anom3",       name:"Curious Cabinet",  desc:"Contain 3 anomalies simultaneously.",          check:g=>(g.anomalies||[]).length>=3},
  {id:"anom10",      name:"Museum",           desc:"Complete study on 10 anomalies.",              check:g=>(g.anomStudied||0)>=10},
  {id:"fragment1",   name:"New Ground",       desc:"Establish a new fragment.",                    check:g=>FRAG.depth>=1},
  {id:"fragment3",   name:"Endless Facility", desc:"Establish three fragments.",                   check:g=>FRAG.depth>=3},
  {id:"hub",         name:"Transit Hub",      desc:"Complete an expedition to The Hub.",           check:g=>!!(g.hubCleared)},
  {id:"parking_lot", name:"Long Term",        desc:"Complete an expedition to The Parking Lot.",   check:g=>!!(g.parkingCleared)},
  {id:"century",     name:"Century",         desc:"Complete 100 expeditions.",                    check:g=>g.runs>=100},
  {id:"data200",     name:"Data Hoarder",    desc:"Hold 200 data at once.",                       check:g=>g.res.data>=200},
  {id:"credits5k",   name:"Compound Interest",desc:"Hold 5,000 credits at once.",                check:g=>g.res.credits>=5000},
  {id:"clearance50", name:"Deep Diver",      desc:"Reach clearance level 50.",                    check:g=>g.clearance>=50},
  {id:"clearance100",name:"The Deepest One", desc:"Reach clearance level 100.",                   check:g=>g.clearance>=100},
  {id:"grit10",      name:"Immovable",       desc:"Have an operative reach Grit 10.",              check:g=>g.ops.some(o=>o.grit>=10)},
  {id:"greenhouse",  name:"Biologist",       desc:"Complete an expedition to The Greenhouse.",     check:g=>!!(g.greenhouseCleared)},
  {id:"catacombs",   name:"Archaeologist",   desc:"Complete an expedition to the Catacombs.",     check:g=>!!(g.cataCleared)},
  {id:"voidlounge",  name:"Do Not Sit Down", desc:"Complete an expedition to The Void Lounge.",   check:g=>!!(g.voidLoungeCleared)},
  {id:"anom5",       name:"Sample Collection",desc:"Study 5 anomalies.",                         check:g=>(g.anomStudied||0)>=5},
];

// Expedition conditions
const CONDITIONS=[
  {id:"quiet", label:"QUIET",   dangerMult:.7,  lootMult:1.05, color:"var(--green)",    weight:18},
  {id:"clear", label:"CLEAR",   dangerMult:1.0, lootMult:1.0,  color:"var(--amber-dim)",weight:50},
  {id:"active",label:"ACTIVE",  dangerMult:1.25,lootMult:1.2,  color:"var(--amber)",    weight:20},
  {id:"hot",   label:"HOT",     dangerMult:1.5, lootMult:1.35, color:"var(--red)",      weight:12},
];
function rollCondition(){
  const total=CONDITIONS.reduce((s,c)=>s+c.weight,0);
  let r=Math.random()*total;
  for(const c of CONDITIONS){r-=c.weight;if(r<=0)return c.id;}
  return "clear";
}
function getCondition(id){return CONDITIONS.find(c=>c.id===id)||CONDITIONS[1];}

// ┌─ world events & entities ──────────────────────────────────
const EVENTS=[
  {title:"Sealed door",flav:"A welded door, scratching behind it. A cache, or something that wants out.",q:"Instruct the team?",
   choices:[
    {label:"Force it open",rk:"Risk · big salvage or an injury",run(){if(Math.random()<0.6){const v=rand(22,48);gain("salvage",v);log("Sealed cache cracked: +"+v+" salvage.","good");}else{hurtRandom();log("It was occupied. An operative was pulled out injured.","bad");}}},
    {label:"Mark it and leave",rk:"Safe · small data",run(){const v=rand(3,6);gain("data",v);log("Door logged and bypassed. +"+v+" data.","report");}}]},
  {title:"Pooled water",flav:"A still pool of clean almond water, far more than a level should hold. Too clean.",q:"Harvest it?",
   choices:[
    {label:"Fill every container",rk:"Greedy · big haul, small contamination risk",run(){if(Math.random()<0.72){const v=rand(30,60);gain("aw",v);log("Harvest successful: +"+v+" almond water.","good");}else{hurtRandom();log("The water was wrong. Operative in recovery and rambling.","bad");}}},
    {label:"Measured ration",rk:"Safe · guaranteed almond water",run(){const v=rand(14,24);gain("aw",v);log("Measured ration: +"+v+" almond water.","report");}}]},
  {title:"A long-term resident",flav:"A person who has lived in the Backrooms for years — voluntarily, they say. They know routes no advance team has mapped. They want to join.",q:"Decision?",
   choices:[
    {label:"Recruit them",rk:"Free operative · they may not be what they seem",run(){if(G.ops.length<G.maxOps){if(Math.random()<0.7){const o=mkOp();o.grit+=2;o.name="(found) "+o.name;G.ops.push(o);G.wanderersFound=(G.wanderersFound||0)+1;log("Recruited wanderer: "+o.name+", Grit "+o.grit+".","good");}else{hurtRandom();log("It wore a person's shape until it didn't. Operative hurt.","bad");}}else{const v=rand(5,9);gain("data",v);log("No room. Debriefed them: +"+v+" data.","report");}}},
    {label:"Debrief and release",rk:"Safe · solid data",run(){const v=rand(7,13);gain("data",v);log("Wanderer debriefed: +"+v+" data on deep routing.","report");}}]},
  {title:"License offer",flav:"A buyer wants first refusal on the next secured anomaly. Good price. Binding contract.",q:"Sign?",
   choices:[
    {label:"Sign the contract",rk:"Credits now",run(){const v=rand(34,75);gain("credits",v);log("Contract signed: +"+v+" credits.","good");}},
    {label:"Stay independent",rk:"Keep options open",run(){const v=rand(4,7);gain("data",v);log("Offer declined. +"+v+" data.","report");}}]},
  {title:"Almond spring",flav:"Survey drones found a self-replenishing spring. The Sync will fund a tap — it goes on the settlement infrastructure list.",q:"Build the tap?",
   choices:[
    {label:"Build it",rk:"Costs 10 salvage · big AW surge",run(){if(G.res.salvage>=10){G.res.salvage-=10;const v=rand(40,80);gain("aw",v);log("Tap installed: +"+v+" almond water.","good");}else{const v=rand(6,12);gain("aw",v);log("No salvage for tap. Hauled by hand: +"+v+" AW.","report");}}},
    {label:"Cap it for later",rk:"Safe · small data",run(){const v=rand(3,6);gain("data",v);log("Spring capped and mapped: +"+v+" data.","report");}}]},
  {title:"Auction lot",flav:"A rival group is liquidating salvage cheap. Cash up front, sight unseen.",q:"Bid?",
   choices:[
    {label:"Buy the lot",rk:"Spend 25 credits · usually a profit",run(){if(G.res.credits>=25){G.res.credits-=25;const v=rand(18,46);gain("salvage",v);log("Lot won: +"+v+" salvage for 25¤.","good");}else{toast("Need 25 credits.");log("Couldn't cover the bid.","report");}}},
    {label:"Pass",rk:"Keep your credits",run(){log("Passed on the auction.","report");}}]},
  {title:"Distress signal",flav:"A looping distress call from a deep level. Could be survivors. Could be bait.",q:"Respond?",
   choices:[
    {label:"Send help",rk:"Risk · rescue an operative or lose one trying",run(){if(Math.random()<0.55&&G.ops.length<G.maxOps){const o=mkOp();G.ops.push(o);log("Rescued a survivor: "+o.name+" joins the roster.","good");}else{hurtRandom();log("The signal was bait. Operative injured.","bad");}}},
    {label:"Log and ignore",rk:"Safe · small data",run(){const v=rand(3,6);gain("data",v);log("Signal triangulated and logged: +"+v+" data.","report");}}]},
  {title:"Equipment cache",flav:"Intact lockers, sealed since before the site was established. Could be anything.",q:"Open them?",
   choices:[
    {label:"Open all",rk:"Varied loot · minor risk",run(){const r=Math.random();if(r<0.5){const s=rand(15,35);gain("salvage",s);log("Cache opened: +"+s+" salvage.","good");}else if(r<0.75){const d=rand(8,16);gain("data",d);log("Cache held research files: +"+d+" data.","good");}else{hurtRandom();log("The cache was a biohazard. Operative contaminated.","bad");}}},
    {label:"Catalogue without opening",rk:"Safe · data only",run(){const v=rand(5,10);gain("data",v);log("Cache catalogued without opening: +"+v+" data.","report");}}]},
  {title:"The phonecall",flav:"One of the field phones rings. You answer it. The voice on the line says your clearance number back to you, digit by digit, then waits.",q:"Respond?",
   choices:[
    {label:"Hang up immediately",rk:"Nothing happens. Probably.",run(){log("The line was dead before you could hang up. Maybe it already was.","dread");}},
    {label:"Stay on the line",rk:"Unsettling · possibly valuable",run(){if(Math.random()<0.6){const v=rand(12,28);gain("data",v);log("The voice read out coordinates. Real ones. +"+v+" data.","good");}else{log("The voice kept reading the number. For eleven minutes. Then stopped.","dread");}}}]},
  {title:"Black market",flav:"An anonymous channel offering salvage for data, no questions asked. The rate is good. Too good.",q:"Trade?",
   choices:[
    {label:"Trade 15 data",rk:"If genuine: 40+ salvage",run(){if(G.res.data>=15){G.res.data-=15;if(Math.random()<0.75){const v=rand(38,68);gain("salvage",v);log("Trade completed: +"+v+" salvage for 15 data.","good");}else{log("They took the data. Nothing arrived. Channel is dead.","bad");}}else{toast("Need 15 data.");}}},
    {label:"Report the channel",rk:"Small data bonus",run(){const v=rand(5,9);gain("data",v);log("Channel reported. Finder's fee: +"+v+" data.","report");}}]},
  {title:"Rogue machine",flav:"A maintenance robot still running old protocols. It can be redirected, for a cost.",q:"Repurpose it?",
   choices:[
    {label:"Spend 12 salvage to redirect",rk:"Ongoing credits",run(){if(G.res.salvage>=12){G.res.salvage-=12;const v=rand(50,90);gain("credits",v);log("Machine redirected to anomaly sorting: +"+v+" ¤.","good");}else{toast("Need 12 salvage.");log("Couldn't redirect the machine.","report");}}},
    {label:"Disable it",rk:"Safe · frees up data",run(){const v=rand(6,11);gain("data",v);log("Machine disabled. Schematics extracted: +"+v+" data.","report");}}]},
  {title:"Contamination risk",flav:"A returning operative seems 'off.' Different. Can't explain it.",q:"Protocol?",
   choices:[
    {label:"Quarantine for 60s",rk:"Lose one operative temporarily · full safety",run(){const idle=G.ops.filter(o=>o.status==="idle");if(idle.length){const o=pick(idle);o.status="injured";o.recover=Date.now()+60000*recoverMult();log(o.name+" quarantined as a precaution. Back in 60s.","report");}else log("No idle operatives to quarantine.","report");}},
    {label:"Clear them immediately",rk:"Risky",run(){if(Math.random()<0.8)log("Operative cleared. Probably fine.","report");else{hurtRandom();log("They weren't fine. Operative stabilised.","bad");}}}]},
  // New events
  {title:"Map fragment",flav:"A crumpled survey map, someone's handwriting all over it. Routes marked that no team has confirmed.",q:"What do you do with it?",
   choices:[
    {label:"Transcribe it",rk:"Costs 5 data · clearance bonus",run(){if(G.res.data>=5){G.res.data-=5;G.clearance+=3;log("Map fragment transcribed. +3 clearance.","good");}else{toast("Need 5 data.");log("Not enough data to transcribe the map.","report");}}},
    {label:"File it",rk:"Safe · data reward",run(){const v=rand(3,6);gain("data",v);log("Map filed in the archive. +"+v+" data.","report");}}]},
  {title:"Supply drop",flav:"A parachute-rigged crate hit the upper walkways. PARSEC batch delivery, six months late.",q:"Recover the crate?",
   choices:[
    {label:"Recover it",rk:"70% big haul, 30% unstable",run(){if(Math.random()<0.7){const aw=rand(30,60);const sal=rand(8,20);gain("aw",aw);gain("salvage",sal);log("Crate recovered: +"+aw+" almond water, +"+sal+" salvage.","good");}else{hurtRandom();log("The crate was unstable. Detonated on retrieval. Operative injured.","bad");}}},
    {label:"Leave it",rk:"Safe · small data",run(){const v=rand(4,8);gain("data",v);log("Crate left on the walkway, location logged. +"+v+" data.","report");}}]},
  {title:"Operative letter",flav:"A hand-written letter found in a field jacket. Addressed to someone on the roster who isn't there anymore. Or — is that the same handwriting as—",q:"What do you do with it?",
   choices:[
    {label:"Read it aloud",rk:"Data bonus · unsettling",run(){const v=rand(6,12);gain("data",v);log("The letter was dated three years ago. Impossible. +"+v+" data.","good");}},
    {label:"Seal it back up",rk:"Nothing gained",run(){log("No one spoke about it. The jacket went back on the rack.","dread");}}]},
  {title:"Power surge",flav:"The facility lights spiked and held. Something in the deep levels is drawing current.",q:"What do you do?",
   choices:[
    {label:"Trace the draw",rk:"Costs 8 salvage · data reward",run(){if(G.res.salvage>=8){G.res.salvage-=8;const v=rand(14,28);gain("data",v);log("Power source traced to sub-level conduit. +"+v+" data.","good");}else{toast("Need 8 salvage.");log("Not enough salvage to trace the draw.","report");}}},
    {label:"Isolate the circuit",rk:"Safe · nothing lost",run(){log("Circuit isolated. The lights returned to normal. No one investigated further.","dread");}}]},
  {title:"Supply pallet",flav:"Someone else's supply drop, still sealed. The branding is unfamiliar but the manifest is legible. It was meant for a team that didn't arrive.",q:"Claim it?",
   choices:[
    {label:"Full claim",rk:"AW + credits",run(){const aw=rand(20,40);const c=rand(14,28);gain("aw",aw);gain("credits",c);log("Pallet claimed: +"+aw+" almond water, +"+c+" ¤.","good");}},
    {label:"Log it and leave",rk:"Data",run(){const v=rand(4,8);gain("data",v);log("Pallet location logged — evidence of another operation. +"+v+" data.","report");}}]},
  {title:"Entity observation",flav:"Something is following the team at distance. Not hostile — curious. It hasn't tried to close the gap.",q:"How does the team respond?",
   choices:[
    {label:"Observe and document",rk:"Large data · risk",run(){const v=rand(14,28);gain("data",v);if(Math.random()<0.3)hurtRandom("it wasn't as passive as it seemed");log("Entity behaviour documented over 30 minutes. +"+v+" data.","good");}},
    {label:"Move away quickly",rk:"Safe",run(){log("Team increased pace. Entity did not follow past the junction.","dread");}}]},
  {title:"Signal on repeat",flav:"The comms unit picked up a loop broadcast — coordinates, four levels deep. Different from anything logged. Still transmitting.",q:"Log it?",
   choices:[
    {label:"Cross-reference and map",rk:"Clearance bonus",run(){const v=rand(2,5);G.clearance+=v;log("Signal cross-referenced. New routing opened. +"+v+" clearance.","good");}},
    {label:"Archive raw signal",rk:"Data",run(){const v=rand(5,10);gain("data",v);log("Signal archived. Frequency noted for analysis. +"+v+" data.","report");}}]},
  {title:"False wall",flav:"The acoustics are wrong. One section of wall sounds hollow — different material, recent installation. Someone put this here after the fact.",q:"Break it open?",
   choices:[
    {label:"Force it open",rk:"Salvage windfall · small risk",run(){if(Math.random()<0.75){const v=rand(18,38);gain("salvage",v);log("False wall removed. Stockpile behind it: +"+v+" salvage.","good");}else{hurtRandom("structural collapse");log("The wall took part of the ceiling with it.","bad");}}},
    {label:"Mark and leave",rk:"Safe · data",run(){const v=rand(3,6);gain("data",v);log("Wall marked on site map. +"+v+" data.","report");}}]},
  {title:"Familiar handwriting",flav:"A note tacked to a pillar. The handwriting belongs to someone on the roster. The date is nine days in the future.",q:"Take it?",
   choices:[
    {label:"Take the note",rk:"Unknown",run(){const v=rand(6,14);gain("data",v);log("The note was taken. Its contents were not shared in the debrief. +"+v+" data.","dread");}},
    {label:"Leave it",rk:"Nothing gained",run(){log("The note was left. The team agreed not to mention it.","dread");}}]},
];

const ANOMALY_NAMES=["Resonant Object","Humming Device","Mirrored Fragment","Null Compass","Recursive Map","Pale Transmission","Self-Indexing File","Amber Shard","Phase Object","Breathing Wall Section","Crystallised Static","Inverted Clock","Warm Stone","Signal Loop","Folded Map","Singing Stone","Null Portrait","Frozen Wristwatch","Double-Exposed Photograph","Inverted Pressure Gauge","Thread Without End","Mechanical Eye","Index Card (illegible)","Carbon Copy","Radio With No Station","Self-Addressed Envelope","Weight That Increases","Sound With No Source","Threshold Marker","Object That Casts No Shadow"];

const LEVEL_EVENTS={
  9:[{title:"The water is rising",flav:"The pool level climbed three inches while the team wasn't watching. It is still climbing.",q:"Call the extraction?",
     choices:[
       {label:"Extract immediately",rk:"Safe · lose 25% haul",run(exp){exp._earlyExtract=true;exp.end=Date.now()+8000;log("Team extracting early from The Poolrooms.","report");}},
       {label:"Push through",rk:"Risk injury · full haul",run(){if(Math.random()<0.55)hurtRandom("The water took them by surprise.");else log("Team outran the water. Haul intact.","report");}}]}],
  4:[{title:"Lights out",flav:"The last headlamp failed. The cave is absolutely dark. The team can hear movement.",q:"Navigate blind or retreat?",
     choices:[
       {label:"Navigate blind",rk:"High risk · double salvage if successful",run(exp){if(Math.random()<0.45){const v=rand(18,36);gain("salvage",v);log("Navigated blind. Found a cache in the dark: +"+v+" salvage.","good");}else{hurtRandom("couldn't see what they hit.");log("Something in the dark found them first.","bad");}}},
       {label:"Retreat and regroup",rk:"Safe · end run early",run(exp){exp.end=Date.now()+5000;log("Team retreating — lights failed.","report");}}]}],
  5:[{title:"They're being too accommodating",flav:"The staff have not left the team alone since they arrived. Smiling. Offering things. The food smells right.",q:"Accept the hospitality?",
     choices:[
       {label:"Accept",rk:"Large AW · possible trap",run(){if(Math.random()<0.6){const v=rand(28,55);gain("aw",v);log("The food was fine. This time. +"+v+" almond water.","good");}else{hurtRandom("what was in the food");log("Something was in the food.","bad");}}},
       {label:"Decline and stay professional",rk:"Safe · small data",run(){const v=rand(4,8);gain("data",v);log("Declined all offers. Team noted routes instead. +"+v+" data.","report");}}]}],
  6:[{title:"The door is open",flav:"The EXIT door is ajar. No team has passed through it and reported back coherently. There is light on the other side.",q:"Send someone through?",
     choices:[
       {label:"One operative steps through",rk:"Unknown · possibly transformative",run(){const v=rand(18,38);gain("data",v);log("They were gone for eleven seconds and came back calm. They won't say what was there. +"+v+" data.","dread");}},
       {label:"Document from the threshold",rk:"Safe · large data",run(){const v=rand(12,22);gain("data",v);log("The light through the door was documented at 14 lux. Same as a candle. +"+v+" data.","good");}}]}],
  8:[{title:"It's right behind them",flav:"The team has stopped communicating in full sentences. Short words only. Movement on the comms feed.",q:"",
     choices:[
       {label:"Drop the haul and run",rk:"Lose all salvage · everyone home safe",run(exp){exp._dropSalvage=true;log("Team dropped everything. Running on instinct now.","bad");}},
       {label:"Keep running, keep the haul",rk:"Risk bad injuries · full loot",run(){hurtRandom("couldn't outrun it carrying everything");if(Math.random()<0.4)hurtRandom("second operative tagged");log("They made it out, barely.","bad");}}]}],
  3:[{title:"All the phones are ringing",flav:"Every phone on every desk, simultaneously. The same ring. The team is frozen.",q:"Answer one?",
     choices:[
       {label:"Answer",rk:"Unsettling · data or trauma",run(){if(Math.random()<0.55){const v=rand(8,18);gain("data",v);log("A voice read numbers for four minutes then stopped. +"+v+" data.","good");}else{hurtRandom("what they heard on the line");log("They hung up immediately but it was too late.","bad");}}},
       {label:"Unplug them all",rk:"Safe · salvage",run(){const v=rand(6,14);gain("salvage",v);log("Unplugged every phone on the floor. Salvaged the copper. +"+v+" salvage.","report");}}]}],
  11:[{title:"The generators are cycling",flav:"The power output is spiking in a pattern. It looks intentional. Something is communicating.",q:"Try to respond?",
     choices:[
       {label:"Mirror the pattern",rk:"Data reward · unknown consequence",run(){const v=rand(10,22);gain("data",v);if(Math.random()<0.25)hurtRandom("the discharge");log("Pattern acknowledged. Something changed in the hum. +"+v+" data.","dread");}},
       {label:"Kill the power to that bank",rk:"Safe · salvage",run(){const v=rand(8,16);gain("salvage",v);log("Generator bank shut down. Salvaged the components. +"+v+" salvage.","report");}}]}],
  10:[{title:"A front door is open",flav:"The house at the end of the cul-de-sac. Light inside. Television on. Something is sitting in front of it.",q:"Go in?",
     choices:[
       {label:"Enter carefully",rk:"Good loot · risk",run(){if(Math.random()<0.55){const aw=rand(15,28);const c=rand(20,40);gain("aw",aw);gain("credits",c);log("House cleared. Supplies inside: +"+aw+" AW, +"+c+" ¤.","good");}else{hurtRandom("whoever was watching television");log("It wasn't a person.","bad");}}},
       {label:"Document and move on",rk:"Data",run(){const v=rand(5,10);gain("data",v);log("House exterior documented. Lights still on. +"+v+" data.","report");}}]}],
  13:[{title:"The highway is busy",flav:"Something is moving through the hub — fast, large, using the corridors as a transit route. The team pressed against the wall.",q:"What do the team do?",
     choices:[
       {label:"Observe and document",rk:"Large data · medium risk",run(){const v=rand(12,24);gain("data",v);if(Math.random()<0.35)hurtRandom("the transit struck the wall too close");log("Entity transit documented: 14 distinct forms, direction unknown. +"+v+" data.","dread");}},
       {label:"Hold still and wait",rk:"Safe · small data",run(){const v=rand(4,8);gain("data",v);log("Team waited 22 minutes. Whatever used the hub moved on. +"+v+" data.","report");}}]}],
  14:[{title:"The safe feeling intensifies",flav:"One operative sat down. Didn't want to get up. Reported that leaving felt wrong, like abandoning somewhere important.",q:"Force extraction?",
     choices:[
       {label:"Force extraction",rk:"Safe · stress relief",run(){G.ops.forEach(o=>{o.stress=Math.max(0,(o.stress||0)-15);});log("Team extracted. Stress was meaningfully lower after leaving the False Light.","good");}},
       {label:"Let them rest a while",rk:"Risk being trapped · large AW",run(){if(Math.random()<0.6){const v=rand(22,44);gain("aw",v);log("The rest restored something. +"+v+" almond water equivalent.","good");}else{hurtRandom("the false light convinced them to stay too long");log("The warmth was a trap. It always is.","bad");}}}]}],
  15:[{title:"All corridors look the same",flav:"The team has been walking for forty minutes and arrived at their starting point twice. The grey offers nothing to navigate by.",q:"Navigation approach?",
     choices:[
       {label:"Mark the walls and create a route",rk:"Costs 6 salvage · data",run(){if(G.res.salvage>=6){G.res.salvage-=6;const v=rand(8,16);gain("data",v);log("Route mapped by damage to the walls. +"+v+" data.","report");}else{toast("Need 6 salvage.");log("Couldn't mark route — not enough salvage.","report");}}},
       {label:"Trust instinct",rk:"Random outcome",run(){const r=Math.random();if(r<0.4){const v=rand(14,28);gain("credits",v);log("Instinct led somewhere useful. +"+v+" ¤.","good");}else if(r<0.7){log("The team circled for another hour. No progress.","report");}else{hurtRandom("disorientation");log("Disorientation set in.","bad");}}}]}],
  16:[{title:"They know your names",flav:"The lounge staff address every operative by name. No introductions were made. They are still smiling.",q:"Engage them?",
     choices:[
       {label:"Gather information",rk:"Large data · risk",run(){const v=rand(10,22);gain("data",v);if(Math.random()<0.3)hurtRandom("what they were told");log("The staff answered every question. The answers were too specific. +"+v+" data.","dread");}},
       {label:"Leave immediately",rk:"Safe · exit early",run(){log("Team left the lounge at pace. Not everyone wanted to go.","report");}}]}],
  17:[{title:"The stairs ended",flav:"After two hours of descent, the stairwell levelled into a concrete room. One door. Warm light underneath.",q:"Open the door?",
     choices:[
       {label:"Go through",rk:"High reward · unknown risk",run(){const s=rand(16,32);const d=rand(8,18);gain("salvage",s);gain("data",d);if(Math.random()<0.45)hurtRandom("what was in the room");log("Through the door, a military-grade cache. +"+s+" salvage, +"+d+" data.","good");}},
       {label:"Note and retreat",rk:"Safe · clearance bonus",run(){const v=rand(2,4);G.clearance+=v;log("Door sealed on record. Clearance updated: +"+v+".","report");}}]}],
  18:[{title:"The plants lean in",flav:"The bioluminescence pulsed in sequence as the team passed. The flora is responsive. One operative reported a warmth — like being noticed.",q:"Stay and study it?",
     choices:[
       {label:"Document the phenomenon",rk:"Excellent data",run(){const v=rand(12,24);gain("data",v);log("Flora behaviour documented over 40 minutes. +"+v+" data.","good");}},
       {label:"Harvest samples",rk:"AW reward · possible reaction",run(){const v=rand(20,40);gain("aw",v);if(Math.random()<0.25)hurtRandom("the flora's defence response");log("Bioluminescent samples extracted: +"+v+" AW equivalent.","good");}}]}],
  19:[{title:"A door with no handle",flav:"Set flush into the stone — no handle, no hinge, no script above it. The cold is more intense here than anywhere else in the catacombs.",q:"Investigate?",
     choices:[
       {label:"Chip at the surround",rk:"Costs 10 salvage · data or danger",run(){if(G.res.salvage>=10){G.res.salvage-=10;const d=rand(14,28);gain("data",d);if(Math.random()<0.35)hurtRandom("what came through when the seal broke");log("The door opened inward. Partially documented. +"+d+" data.","dread");}else{toast("Need 10 salvage.");log("Not enough salvage to investigate.","report");}}},
       {label:"Record and step back",rk:"Small data",run(){const v=rand(5,10);gain("data",v);log("The door photographed. Dimensions noted. It is not standard. +"+v+" data.","report");}}]}],
  20:[{title:"The figure in the chair",flav:"The figure is still. It does not breathe. It has the face of the last operative to go missing. The team recognised them.",q:"Approach it?",
     choices:[
       {label:"Speak to it",rk:"Possible contact · extreme risk",run(){if(Math.random()<0.4){const v=rand(20,40);gain("data",v);log("It spoke. The team refuses to transcribe what it said. +"+v+" data.","dread");}else{hurtRandom("what the figure did");hurtRandom("secondary response");log("They should not have gone closer.","bad");}}},
       {label:"Document from the door",rk:"Safe · large data",run(){const v=rand(14,26);gain("data",v);log("Photographed from the threshold. The figure did not react. +"+v+" data.","report");}}]}],
};

const ENTITIES={
  0: {name:"Bacterium Cluster",    icon:"◉", threat:0.10, flav:"Yellow-grey, pulsing. Walls of it. It moves when disturbed, and it has been disturbed."},
  1: {name:"Facelings",            icon:"◌", threat:0.18, flav:"Blank-faced wanderers. Passive unless cornered — and the team has cornered them."},
  2: {name:"Pipe Crawlers",        icon:"◈", threat:0.28, flav:"Something inside the pipes, moving parallel to the team, matching their pace exactly."},
  3: {name:"Deathmoths",           icon:"◎", threat:0.22, flav:"Large. Silent. Drawn to light and movement. The ceiling is moving."},
  4: {name:"Cave Spiders",         icon:"◆", threat:0.35, flav:"Not like surface spiders. Faster. Quieter. Far too many legs."},
  5: {name:"Smilers",              icon:"⌇", threat:0.52, flav:"Always smiling. Always watching. They have been waiting here for a very long time."},
  6: {name:"The Archivist",        icon:"⎔", threat:0.68, flav:"It knows the team is there. It has been patient about it. It is done being patient."},
  7: {name:"Glass Figures",        icon:"◇", threat:0.14, flav:"Translucent, slow-moving. They look fragile. They have survived here longer than the team has been alive."},
  8: {name:"The Hunter",           icon:"▲", threat:0.88, flav:"Nothing else has its full attention right now. Nothing."},
  9: {name:"Pool Presence",        icon:"◌", threat:0.24, flav:"Beneath the surface. Watching. It hasn't broken the waterline yet."},
  10:{name:"Suburbanites",         icon:"◉", threat:0.30, flav:"Human-shaped. Human-proportioned. Not human. They came out of the houses."},
  11:{name:"Static Wraith",        icon:"⊕", threat:0.42, flav:"Feeds on electrical discharge. Extremely fast. The instruments are going wrong."},
  12:{name:"Parking Stalker",      icon:"◈", threat:0.24, flav:"It has been in the parking lot since before the cars stopped. It remembers when they moved."},
  13:{name:"Transit Form",         icon:"▷", threat:0.46, flav:"Uses the hub as a highway. Does not slow down for obstacles. The team is an obstacle."},
  14:{name:"The Lure",             icon:"⌾", threat:0.55, flav:"It IS the warm light. It has always been the warm light. It has been waiting to be recognised."},
  15:{name:"Mirror Walker",        icon:"⊘", threat:0.44, flav:"Identical to one of the operatives. The team can't agree on which one."},
  16:{name:"Maître D'",            icon:"⌇", threat:0.28, flav:"Impeccably dressed. Very insistent. The team hasn't finished their reservation."},
  17:{name:"Stair Dweller",        icon:"◆", threat:0.38, flav:"Has lived in the stairwell for longer than memory allows. This stairwell is its home."},
  18:{name:"Root System",          icon:"◉", threat:0.20, flav:"The greenhouse is not a place. The greenhouse is the entity. It has noticed the team."},
  19:{name:"The Interred",         icon:"⎔", threat:0.50, flav:"They were buried here. They have not fully accepted that."},
  20:{name:"The Reflection",       icon:"⊘", threat:0.65, flav:"It has your face. It knows you noticed. It is walking toward you now."},
};