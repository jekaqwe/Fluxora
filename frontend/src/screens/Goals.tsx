import { useState } from "react";
import { Plus, Plane, CarFront, Laptop, X } from "lucide-react";
import { PageHeading, Panel, Stat, Progress, money } from "./ui";
import "./Pure.css";
type Goal={id:number;name:string;current:number;target:number;date:string;kind:"plane"|"car"|"laptop"};
const initial:Goal[]=[{id:1,name:"На отпуск",current:62000,target:100000,date:"01.08.2025",kind:"plane"},{id:2,name:"На машину",current:120000,target:300000, date:"01.12.2025",kind:"car"},{id:3,name:"Новый ПК",current:25000,target:100000,date:"01.03.2026",kind:"laptop"}];
function Goals(){
 const [goals,setGoals]=useState<Goal[]>(()=>{try{return JSON.parse(localStorage.getItem("fluxora_demo_goals")||"null")||initial}catch{return initial}});
 const [show,setShow]=useState(false);const [name,setName]=useState("");const [target,setTarget]=useState("");
 const save=(next:Goal[])=>{setGoals(next);localStorage.setItem("fluxora_demo_goals",JSON.stringify(next));};
 const add=()=>{const n=Number(target);if(!name.trim()||n<=0)return;save([...goals,{id:Date.now(),name:name.trim(),current:0,target:n,date:"Срок не задан",kind:"laptop"}]);setName("");setTarget("");setShow(false);};
 const contribute=(id:number)=>save(goals.map(g=>g.id===id?{...g,current:Math.min(g.target,g.current+5000)}:g));
 const total=goals.reduce((s,g)=>s+g.current,0);
 return <div className="pure-page"><PageHeading title="Цели" subtitle="Планируйте накопления и приближайте мечты" action={<button className="pure-button" onClick={()=>setShow(v=>!v)}><Plus size={16}/>Создать цель</button>}/>
 <div className="pure-grid two"><Stat label="Всего накоплено" value={money(total)} change="По всем целям" icon={<Plane size={17}/>}/><Stat label="Активные цели" value={String(goals.length)} change="Ваш план накоплений" tone="mint"/></div>
 {show&&<Panel title="Новая цель"><div className="form-row"><label className="form-label" style={{flex:2}}>Название<input className="pure-input" value={name} onChange={e=>setName(e.target.value)} placeholder="Например, Путешествие"/></label><label className="form-label" style={{flex:1}}>Сумма, ₽<input className="pure-input" type="number" min="1" value={target} onChange={e=>setTarget(e.target.value)} placeholder="100000"/></label></div><div className="form-row" style={{marginTop:13}}><button className="pure-button" onClick={add}>Добавить цель</button><button className="pure-button ghost" onClick={()=>setShow(false)}><X size={14}/>Отмена</button></div></Panel>}
 <div className="pure-grid three" style={{marginTop:16}}>{goals.map(g=>{const Icon=g.kind==="plane"?Plane:g.kind==="car"?CarFront:Laptop;const pct=Math.min(100,Math.round(g.current/g.target*100));return <Panel key={g.id} className="goal-card"><div className="goal-art"><Icon/></div><div className="goal-content"><h3>{g.name}</h3><p>Цель накоплений</p><strong style={{fontSize:21}}>{pct}%</strong><div className="goal-meta"><span>{money(g.current)}</span><span>{money(g.target)}</span></div><Progress value={pct}/><div className="goal-meta"><span>Срок: {g.date}</span><span>{money(Math.max(0,g.target-g.current))} осталось</span></div><button className="pure-button secondary" style={{width:"100%",marginTop:12}} onClick={()=>contribute(g.id)} disabled={pct>=100}>+ Пополнить на 5 000 ₽</button></div></Panel>})}</div>
 <p className="muted" style={{fontSize:10,marginTop:14}}>Созданные цели и пополнения сохраняются локально в этом браузере.</p>
 </div>;
}
export default Goals;
