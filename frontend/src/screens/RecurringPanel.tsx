import { useEffect, useState } from "react";
import { Panel } from "./ui";
import { Plus, Pause, Play, Trash2, Check, X, Pencil, History } from "lucide-react";

type Account={id:number;name:string};
type Item={id:number;title:string;type:"income"|"expense";amount:string;category:string;accountId?:number;frequency:string;nextDate:string;endDate?:string;status:"active"|"paused"|"completed"|"deleted"};
type Missed={id:number;recurringId:number;title:string;type:"income"|"expense";amount:string;category:string;accountId?:number;date:string};
type HistoryItem={id:number;date:string;status:"pending"|"confirmed"|"skipped";transactionId?:number;createdAt:string};
const frequencies:Record<string,string>={daily:"Ежедневно",weekly:"Еженедельно",monthly:"Ежемесячно",yearly:"Ежегодно"};

export default function RecurringPanel({accounts}:{accounts:Account[]}){
 const [items,setItems]=useState<Item[]>([]);
 const [accountSelections,setAccountSelections]=useState<Record<number,string>>({});
 const [missed,setMissed]=useState<Missed[]>([]);
 const [history,setHistory]=useState<HistoryItem[]>([]);
 const [historyOpenId,setHistoryOpenId]=useState<number|null>(null);
 const [show,setShow]=useState(false);
 const [editingId,setEditingId]=useState<number|null>(null);
 const [title,setTitle]=useState("");
 const [type,setType]=useState<"income"|"expense">("expense");
 const [amount,setAmount]=useState("");
 const [category,setCategory]=useState("Регулярные платежи");
 const [accountId,setAccountId]=useState("");
 const [frequency,setFrequency]=useState("monthly");
 const [nextDate,setNextDate]=useState(new Date().toISOString().slice(0,10));
 const [endDate,setEndDate]=useState("");
 const [error,setError]=useState("");
 const [saving,setSaving]=useState(false);
 const load=async()=>{try{const r=await fetch("/api/recurring");const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось загрузить расписание");setItems(d.items||[]);setMissed(d.missed||[]);}catch(e){setError(e instanceof Error?e.message:"Ошибка расписания");}};
 useEffect(()=>{void load();},[]);
 const activeItems=items.filter(item=>item.status!=="deleted");
 const deletedItems=items.filter(item=>item.status==="deleted");
 const showHistory=async(id:number)=>{if(historyOpenId===id){setHistoryOpenId(null);return;}try{const r=await fetch(`/api/recurring/${id}/history`);const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось загрузить историю");setHistory(d);setHistoryOpenId(id);}catch(e){setError(e instanceof Error?e.message:"Ошибка истории");}};
 const resetForm=()=>{setShow(false);setEditingId(null);setTitle("");setAmount("");setCategory("Регулярные платежи");setEndDate("");};
 const create=async()=>{setSaving(true);setError("");try{const url=editingId===null?"/api/recurring":`/api/recurring/${editingId}`;const r=await fetch(url,{method:editingId===null?"POST":"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({title,type,amount:amount.replace(",", "."),category,accountId:Number(accountId),frequency,nextDate,endDate,status:editingId===null?"active":(items.find(item=>item.id===editingId)?.status==="paused"?"paused":"active")})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось сохранить расписание");resetForm();await load();}catch(e){setError(e instanceof Error?e.message:"Ошибка сохранения");}finally{setSaving(false);}};
 const edit=(item:Item)=>{setEditingId(item.id);setTitle(item.title);setType(item.type);setAmount(item.amount);setCategory(item.category);setAccountId(String(item.accountId||""));setFrequency(item.frequency);setNextDate(item.nextDate);setEndDate(item.endDate||"");setShow(true);};
 const update=async(item:Item,status:"active"|"paused")=>{const selected=item.accountId||Number(accountSelections[item.id]||accountId);try{const r=await fetch(`/api/recurring/${item.id}`,{method:"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({status,accountId:selected})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось обновить расписание");await load();}catch(e){setError(e instanceof Error?e.message:"Ошибка обновления");}};
 const remove=async(id:number)=>{if(!window.confirm("Удалить регулярную операцию?"))return;try{const r=await fetch(`/api/recurring/${id}`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить расписание");await load();}catch(e){setError(e instanceof Error?e.message:"Ошибка удаления");}};
 const resolve=async(id:number,action:"confirm"|"skip")=>{try{const r=await fetch(`/api/recurring/occurrences/${id}/${action}`,{method:"POST"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось обработать дату");await load();}catch(e){setError(e instanceof Error?e.message:"Ошибка обработки даты");}};
 const confirmAll=async()=>{for(const item of missed){try{const r=await fetch(`/api/recurring/occurrences/${item.id}/confirm`,{method:"POST"});if(!r.ok){const d=await r.json();throw new Error(d.error||"Не удалось подтвердить операцию");}}catch(e){setError(e instanceof Error?e.message:"Ошибка подтверждения");break;}}await load();};
 return <>
  <Panel title="Регулярные операции" subtitle="Операции по расписанию">
   <div className="form-row" style={{justifyContent:"flex-end"}}><button className="pure-button" onClick={()=>{if(show){resetForm();}else{setEditingId(null);setShow(true);}}}><Plus size={15}/>Добавить расписание</button></div>
   {show&&<div className="pure-grid three" style={{marginTop:12}}>
    <label className="form-label">Название<input className="pure-input" value={title} onChange={e=>setTitle(e.target.value)}/></label>
    <label className="form-label">Тип<select className="pure-select" value={type} onChange={e=>setType(e.target.value as "income"|"expense")}><option value="expense">Расход</option><option value="income">Доход</option></select></label>
    <label className="form-label">Сумма, ₽<input className="pure-input" value={amount} onChange={e=>setAmount(e.target.value)}/></label>
    <label className="form-label">Категория<input className="pure-input" value={category} onChange={e=>setCategory(e.target.value)}/></label>
    <label className="form-label">Счёт<select className="pure-select" value={accountId} onChange={e=>setAccountId(e.target.value)}>{accounts.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</select></label>
    <label className="form-label">Периодичность<select className="pure-select" value={frequency} onChange={e=>setFrequency(e.target.value)}>{Object.entries(frequencies).map(([key,label])=><option key={key} value={key}>{label}</option>)}</select></label>
    <label className="form-label">Следующая дата<input className="pure-input" type="date" value={nextDate} onChange={e=>setNextDate(e.target.value)}/></label>
    <label className="form-label">Последняя дата<input className="pure-input" type="date" value={endDate} onChange={e=>setEndDate(e.target.value)}/></label>
    <div className="form-row" style={{alignItems:"end"}}><button className="pure-button" disabled={saving||!title.trim()||!accountId||!/^\d+(?:[.,]\d{1,2})?$/.test(amount)} onClick={()=>void create()}>{saving?"Сохранение…":editingId===null?"Создать":"Сохранить изменения"}</button><button className="pure-button ghost" onClick={resetForm}>Отмена</button></div>
   </div>}
   {activeItems.length===0?<div className="empty-note">Регулярных операций пока нет.</div>:<div className="transaction-list">{activeItems.map(item=><div className="transaction-row" key={item.id}>
    <span className={`transaction-icon ${item.type==="income"?"mint":""}`}>{item.type==="income"?"+":"↻"}</span>
    <span className="transaction-copy"><strong>{item.title}</strong><small>{frequencies[item.frequency]} · Следующая дата: {item.nextDate} · {item.status==="active"?"Активна":item.status==="paused"?"Приостановлена":"Завершена"}</small></span>
    <strong className="transaction-amount">{item.amount} ₽</strong>
    {item.status==="paused"&&!item.accountId&&<select className="pure-select" aria-label="Выберите счёт для возобновления" value={accountSelections[item.id]||""} onChange={e=>setAccountSelections(old=>({...old,[item.id]:e.target.value}))}><option value="">Счёт для возобновления</option>{accounts.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</select>}{item.status!=="completed"&&<button className="pure-button ghost" aria-label={item.status==="active"?"Приостановить":"Возобновить"} disabled={item.status==="paused"&&!item.accountId&&!accountSelections[item.id]&&!accountId} onClick={()=>void update(item,item.status==="active"?"paused":"active")}>{item.status==="active"?<Pause size={14}/>:<Play size={14}/>}</button>}
    {item.status!=="completed"&&<button className="pure-button ghost" aria-label="История расписания" onClick={()=>void showHistory(item.id)}><History size={14}/></button><button className="pure-button ghost" aria-label="Изменить расписание" onClick={()=>edit(item)}><Pencil size={14}/></button>}<button className="pure-button ghost" aria-label="Удалить расписание" onClick={()=>void remove(item.id)}><Trash2 size={14}/></button>
   </div>)}</div>}
  </Panel>
  {deletedItems.length>0&&<Panel title="Архив регулярных операций" subtitle="Удалённые расписания сохранены вместе с историей">
   <div className="transaction-list">{deletedItems.map(item=><div className="transaction-row" key={item.id}><span className="transaction-copy"><strong>{item.title}</strong><small>{frequencies[item.frequency]} · Архивировано</small></span><strong className="transaction-amount">{item.amount} ₽</strong><button className="pure-button ghost" onClick={()=>void showHistory(item.id)}><History size={14}/>История</button></div>)}</div>
  </Panel>}
  {historyOpenId!==null&&<Panel title="История регулярной операции" subtitle="Созданные, пропущенные и подтверждённые даты">
   {history.length===0?<div className="empty-note">История пока пуста.</div>:<div className="transaction-list">{history.map(item=><div className="transaction-row" key={item.id}><span className="transaction-copy"><strong>{item.date}</strong><small>{item.status==="confirmed"?"Подтверждена":item.status==="skipped"?"Пропущена":"Ожидает подтверждения"}</small></span>{item.transactionId&&<small>Операция #{item.transactionId}</small>}</div>)}</div>}
  </Panel>}
  {missed.length>0&&<Panel title={`Пропущенные даты: ${missed.length}`} subtitle="Они не были созданы автоматически. Подтвердите каждую или все сразу." action={<button className="pure-button" onClick={()=>void confirmAll()}><Check size={14}/>Подтвердить все</button>}>
   <div className="transaction-list">{missed.map(item=><div className="transaction-row" key={item.id}>
    <span className="transaction-copy"><strong>{item.title}</strong><small>{item.date} · {item.type==="income"?"Доход":"Расход"} · {item.category}</small></span>
    <strong className="transaction-amount">{item.amount} ₽</strong>
    <button className="pure-button" onClick={()=>void resolve(item.id,"confirm")}><Check size={14}/>Подтвердить</button>
    <button className="pure-button ghost" onClick={()=>void resolve(item.id,"skip")}><X size={14}/>Пропустить</button>
   </div>)}</div>
  </Panel>}
  {error&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
 </>;
}
