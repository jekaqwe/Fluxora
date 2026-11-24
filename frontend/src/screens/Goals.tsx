import { useEffect, useState, type ChangeEvent } from "react";
import { Plus, Target, Plane, CarFront, Laptop, X, Pause, Play, Trash2, Pencil, History, Image, XCircle } from "lucide-react";
import { PageHeading, Panel, Stat, Progress } from "./ui";
import "./Pure.css";

type Priority="low"|"medium"|"high";
type Goal={id:number;title:string;description:string;targetAmount:string;startingAmount:string;currentAmount:string;dueDate?:string;priority:Priority;status:"active"|"paused"|"completed";createdAt:string;imageUrl?:string};
type Account={id:number;name:string};
type Entry={id:number;type:"contribution"|"withdrawal";amount:string;comment:string;createdAt:string;transactionId?:number};
const toCents=(value:string)=>{const negative=value.startsWith("-");const [whole,fraction=""]=value.replace("-","").replace(",",".").split(".");const cents=BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));return negative?-cents:cents;};
const moneyCents=(value:bigint)=>{const negative=value<0n;const abs=negative?-value:value;return (negative?"−":"")+new Intl.NumberFormat("ru-RU").format(abs/100n)+","+(abs%100n).toString().padStart(2,"0")+" ₽";};
const money=(value:string)=>moneyCents(toCents(value));
const validAmount=(value:string)=>/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(value)&&toCents(value)>0n;

function Goals(){
 const [goals,setGoals]=useState<Goal[]>([]);
 const [accounts,setAccounts]=useState<Account[]>([]);
 const [entryAccountId,setEntryAccountId]=useState("");
 const [loading,setLoading]=useState(true);
 const [error,setError]=useState("");
 const [show,setShow]=useState(false);
 const [editingId,setEditingId]=useState<number|null>(null);
 const [title,setTitle]=useState("");
 const [target,setTarget]=useState("");
 const [starting,setStarting]=useState("0");
 const [dueDate,setDueDate]=useState("");
 const [priority,setPriority]=useState<Priority>("medium");
 const [description,setDescription]=useState("");
 const [saving,setSaving]=useState(false);
 const [entryGoalId,setEntryGoalId]=useState<number|null>(null);
 const [entryAmount,setEntryAmount]=useState("");
 const [entryType,setEntryType]=useState<"contribution"|"withdrawal">("contribution");
 const [entryKey,setEntryKey]=useState("");
 const [historyGoalId,setHistoryGoalId]=useState<number|null>(null);
 const [history,setHistory]=useState<Record<number,Entry[]>>({});

 const load=async()=>{try{const [r,a]=await Promise.all([fetch("/api/goals"),fetch("/api/accounts")]);const [d,accountData]=await Promise.all([r.json(),a.json()]);if(!r.ok||!a.ok)throw new Error("Не удалось загрузить цели и счета");setGoals(d);setAccounts(accountData);if(accountData.length&&!entryAccountId)setEntryAccountId(String(accountData[0].id));}catch(e){setError(e instanceof Error?e.message:"Ошибка загрузки");}finally{setLoading(false);}};
 useEffect(()=>{void load();},[]);

 const resetForm=()=>{setShow(false);setEditingId(null);setTitle("");setTarget("");setStarting("0");setDueDate("");setPriority("medium");setDescription("");};
 const beginEdit=(g:Goal)=>{setEditingId(g.id);setTitle(g.title);setTarget(g.targetAmount);setStarting(g.startingAmount);setDueDate(g.dueDate||"");setPriority(g.priority);setDescription(g.description);setShow(true);};
 const save=async()=>{
  setError("");setSaving(true);
  try{
   const response=await fetch(editingId===null?"/api/goals":`/api/goals/${editingId}`,{method:editingId===null?"POST":"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({title,description,targetAmount:target.replace(",", "."),startingAmount:starting.replace(",", "."),dueDate,priority,status:editingId===null?"active":(goals.find(g=>g.id===editingId)?.status||"active")})});
   const data=await response.json();if(!response.ok)throw new Error(data.error||"Не удалось сохранить цель");
   setGoals(old=>editingId===null?[data,...old]:old.map(g=>g.id===data.id?data:g));resetForm();
  }catch(e){setError(e instanceof Error?e.message:"Ошибка сохранения");}finally{setSaving(false);}
 };
 const recordEntry=async()=>{
  if(entryGoalId===null)return;setError("");
  try{
   const response=await fetch(`/api/goals/${entryGoalId}/entries`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({type:entryType,amount:entryAmount.replace(",", "."),comment:"",accountId:Number(entryAccountId),idempotencyKey:entryKey})});
   const data=await response.json();if(!response.ok)throw new Error(data.error||"Не удалось записать движение");
   const savedGoalId=entryGoalId;setEntryGoalId(null);setEntryAmount("");setEntryKey("");await load();
   const historyResponse=await fetch(`/api/goals/${savedGoalId}/entries`);const historyData=await historyResponse.json();if(historyResponse.ok)setHistory(old=>({...old,[savedGoalId]:historyData}));
  }catch(e){setError(e instanceof Error?e.message:"Ошибка операции");}
 };
 const uploadGoalImage=async(id:number,event:ChangeEvent<HTMLInputElement>)=>{
  const file=event.target.files?.[0];event.target.value="";if(!file)return;setError("");
  try{const form=new FormData();form.append("image",file);const r=await fetch(`/api/goals/${id}/image`,{method:"PUT",body:form});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось загрузить изображение");setGoals(old=>old.map(g=>g.id===id?{...g,imageUrl:d.imageUrl}:g));}
  catch(e){setError(e instanceof Error?e.message:"Ошибка изображения");}
 };
 const deleteGoalImage=async(id:number)=>{
  try{const r=await fetch(`/api/goals/${id}/image`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить изображение");setGoals(old=>old.map(g=>g.id===id?{...g,imageUrl:undefined}:g));}
  catch(e){setError(e instanceof Error?e.message:"Ошибка удаления изображения");}
 };
 const remove=async(id:number)=>{if(!window.confirm("Удалить цель и её историю?"))return;try{const r=await fetch(`/api/goals/${id}`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить цель");setGoals(old=>old.filter(g=>g.id!==id));}catch(e){setError(e instanceof Error?e.message:"Ошибка удаления");}};
 const togglePause=async(g:Goal)=>{try{const r=await fetch(`/api/goals/${g.id}`,{method:"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({...g,status:g.status==="paused"?"active":"paused"})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось изменить статус");setGoals(old=>old.map(x=>x.id===g.id?d:x));}catch(e){setError(e instanceof Error?e.message:"Ошибка статуса");}};
 const showHistory=async(id:number)=>{if(historyGoalId===id){setHistoryGoalId(null);return;}try{const r=await fetch(`/api/goals/${id}/entries`);const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось загрузить историю");setHistory(old=>({...old,[id]:d}));setHistoryGoalId(id);}catch(e){setError(e instanceof Error?e.message:"Ошибка истории");}};
 const total=goals.reduce((sum,g)=>sum+toCents(g.currentAmount),0n);
 const active=goals.filter(g=>g.status==="active").length;

 return <div className="pure-page">
  <PageHeading title="Цели" subtitle="Планируйте накопления и приближайте мечты" action={<button className="pure-button" onClick={()=>{resetForm();setShow(true);}}><Plus size={16}/>Создать цель</button>}/>
  <div className="pure-grid two"><Stat label="Всего накоплено" value={moneyCents(total)} change="По всем целям" icon={<Target size={17}/>}/><Stat label="Активные цели" value={String(active)} change="Ваш план накоплений" tone="mint"/></div>
  {show&&<Panel title={editingId===null?"Новая цель":"Изменить цель"}>
   <div className="pure-grid two">
    <label className="form-label">Название<input className="pure-input" value={title} onChange={e=>setTitle(e.target.value)} maxLength={120} placeholder="Например, Путешествие"/></label>
    <label className="form-label">Целевая сумма, ₽<input className="pure-input" value={target} onChange={e=>setTarget(e.target.value)} placeholder="100000"/></label>
    <label className="form-label">Уже накоплено, ₽<input className="pure-input" value={starting} onChange={e=>setStarting(e.target.value)} placeholder="0"/></label>
    <label className="form-label">Срок<input className="pure-input" type="date" value={dueDate} onChange={e=>setDueDate(e.target.value)}/></label>
    <label className="form-label">Приоритет<select className="pure-select" value={priority} onChange={e=>setPriority(e.target.value as Priority)}><option value="low">Низкий</option><option value="medium">Средний</option><option value="high">Высокий</option></select></label>
    <label className="form-label">Описание<input className="pure-input" value={description} onChange={e=>setDescription(e.target.value)} maxLength={1000}/></label>
   </div>
   <div className="form-row" style={{marginTop:13}}><button className="pure-button" disabled={saving||!title.trim()||!validAmount(target)||!(/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(starting))} onClick={()=>void save()}>{saving?"Сохранение…":editingId===null?"Добавить цель":"Сохранить изменения"}</button><button className="pure-button ghost" onClick={resetForm}><X size={14}/>Отмена</button></div>
  </Panel>}
  {loading?<div className="empty-note">Загружаем цели…</div>:goals.length===0?<div className="empty-note">Целей пока нет. Создайте первую и начните копить на важное.</div>:
   <div className="pure-grid three" style={{marginTop:16}}>{goals.map(g=>{
    const current=toCents(g.currentAmount);const targetCents=toCents(g.targetAmount);
    const pct=Math.max(0,Math.min(100,Number(current*10000n/targetCents)/100));
    const Icon=g.title.toLowerCase().includes("авто")||g.title.toLowerCase().includes("машин")?CarFront:g.title.toLowerCase().includes("отпуск")||g.title.toLowerCase().includes("путеше")?Plane:Laptop;
    return <Panel key={g.id} className="goal-card">
     <div className="goal-art">{g.imageUrl?<img src={g.imageUrl} alt={g.title} style={{width:"100%",height:"100%",objectFit:"cover"}}/>:<Icon/>}</div><div className="goal-content">
      <div className="form-row" style={{justifyContent:"space-between"}}><h3>{g.title}</h3><span className="muted">{g.priority==="high"?"Высокий":g.priority==="low"?"Низкий":"Средний"}</span></div>
      <p>{g.description||"Цель накоплений"}</p><strong style={{fontSize:21}}>{pct.toFixed(1).replace(".",",")}%</strong>
      <div className="goal-meta"><span>{money(g.currentAmount)}</span><span>{money(g.targetAmount)}</span></div><Progress value={pct}/>
      <div className="goal-meta"><span>Срок: {g.dueDate||"не задан"}</span><span>{moneyCents(targetCents>current?targetCents-current:0n)}</span></div>
      <span className="muted">{g.status==="paused"?"Приостановлена":g.status==="completed"?"Завершена":"Активна"}</span>
      <div className="form-row" style={{marginTop:10,flexWrap:"wrap"}}>
       <button className="pure-button secondary" onClick={()=>{setEntryGoalId(g.id);setEntryType("contribution");setEntryKey(crypto.randomUUID());}} disabled={g.status!=="active"}>Пополнить</button>
       <button className="pure-button ghost" onClick={()=>{setEntryGoalId(g.id);setEntryType("withdrawal");setEntryKey(crypto.randomUUID());}} disabled={g.status!=="active"||current<=0n}>Снять</button>
       <button className="pure-button ghost" onClick={()=>beginEdit(g)}><Pencil size={14}/></button>
       <button className="pure-button ghost" onClick={()=>void togglePause(g)} disabled={g.status==="completed"}>{g.status==="paused"?<Play size={14}/>:<Pause size={14}/>}</button>
       <button className="pure-button ghost" onClick={()=>void showHistory(g.id)}><History size={14}/></button><label className="pure-button ghost" title="Изменить изображение"><Image size={14}/><input type="file" accept="image/jpeg,image/png,image/webp,image/gif" hidden onChange={e=>void uploadGoalImage(g.id,e)}/></label>{g.imageUrl&&<button className="pure-button ghost" title="Удалить изображение" onClick={()=>void deleteGoalImage(g.id)}><XCircle size={14}/></button>}
       <button className="pure-button ghost" onClick={()=>void remove(g.id)}><Trash2 size={14}/></button>
      </div>
      {entryGoalId===g.id&&<div style={{marginTop:12}}><label className="form-label">{entryType==="contribution"?"Пополнение, ₽":"Снятие, ₽"}<input className="pure-input" value={entryAmount} onChange={e=>setEntryAmount(e.target.value)} placeholder="0.00"/></label><label className="form-label" style={{marginTop:8}}>Счёт<select className="pure-select" value={entryAccountId} onChange={e=>setEntryAccountId(e.target.value)}>{accounts.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</select></label><div className="form-row" style={{marginTop:8}}><button className="pure-button" disabled={!validAmount(entryAmount)||!entryAccountId} onClick={()=>void recordEntry()}>Подтвердить</button><button className="pure-button ghost" onClick={()=>setEntryGoalId(null)}>Отмена</button></div></div>}
      {historyGoalId===g.id&&<div className="empty-note" style={{marginTop:12}}>{(history[g.id]||[]).length?(history[g.id]||[]).map(e=><div key={e.id} className="legend-row"><span>{e.type==="contribution"?"Пополнение":"Снятие"} · {new Date(e.createdAt).toLocaleDateString("ru-RU")}</span><strong>{money(e.amount)}</strong></div>):"История пока пуста."}</div>}
     </div>
    </Panel>;
   })}</div>}
  {error&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
 </div>;
}
export default Goals;
