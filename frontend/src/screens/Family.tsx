import { useEffect, useState } from "react";
import { Plus, Users, Home as HomeIcon, Trash2, UserPlus, LogOut, Pencil } from "lucide-react";
import { PageHeading, Panel, Stat } from "./ui";
import "./Pure.css";

type FamilyInfo={id:number;name:string;initialBalance:string;currentBalance:string;role:"owner"|"admin"|"member"};
type Member={id:number;userId:number;firstName:string;lastName:string;username:string;role:string;joinedAt:string};
type FamilyTx={id:number;title:string;type:"income"|"expense";amount:string;category:string;date:string;comment:string;author:string;createdAt:string};
type FamilyData={family:FamilyInfo|null;members:Member[]};
const cents=(v:string)=>{const [whole,fraction=""]=v.split(".");return BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));};
const moneyCents=(value:bigint)=>{const neg=value<0n;const abs=neg?-value:value;return (neg?"−":"")+new Intl.NumberFormat("ru-RU").format(abs/100n)+","+(abs%100n).toString().padStart(2,"0")+" ₽";};
const money=(v:string)=>{const neg=v.startsWith("-");const [whole,fraction=""]=v.replace("-","").split(".");const value=BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));return moneyCents(neg?-value:value);};
const today=()=>{const d=new Date();return [d.getFullYear(),String(d.getMonth()+1).padStart(2,"0"),String(d.getDate()).padStart(2,"0")].join("-");};

function Family(){
 const [family,setFamily]=useState<FamilyInfo|null>(null);
 const [members,setMembers]=useState<Member[]>([]);
 const [transactions,setTransactions]=useState<FamilyTx[]>([]);
 const [loading,setLoading]=useState(true);
 const [error,setError]=useState("");
 const [familyName,setFamilyName]=useState("Семейный бюджет");
 const [initialBalance,setInitialBalance]=useState("0");
 const [inviteEmail,setInviteEmail]=useState("");
 const [inviteToken,setInviteToken]=useState("");
 const [acceptToken,setAcceptToken]=useState("");
 const [title,setTitle]=useState("");
 const [amount,setAmount]=useState("");
 const [category,setCategory]=useState("Покупки");
 const [date,setDate]=useState(today());
 const [kind,setKind]=useState<"income"|"expense">("expense");
 const [showForm,setShowForm]=useState(false);
 const [editingTxId,setEditingTxId]=useState<number|null>(null);

 const load=async()=>{
  setError("");
  try{
   const r=await fetch("/api/family");const d:FamilyData=await r.json();if(!r.ok)throw new Error("Не удалось загрузить семейный бюджет");
   setFamily(d.family);setMembers(d.members||[]);
   if(d.family){const tx=await fetch("/api/family/transactions");const list=await tx.json();if(!tx.ok)throw new Error(list.error||"Не удалось загрузить семейные операции");setTransactions(list);}
  }catch(e){setError(e instanceof Error?e.message:"Ошибка загрузки");}finally{setLoading(false);}
 };
 useEffect(()=>{void load();},[]);

 const createFamily=async()=>{
  setError("");
  try{const r=await fetch("/api/family",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({name:familyName,initialBalance:initialBalance.replace(",",".")})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось создать семейный бюджет");await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка создания");}
 };
 const invite=async()=>{
  setError("");
  try{const r=await fetch("/api/family/invitations",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({email:inviteEmail})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось создать приглашение");setInviteToken(d.token);setInviteEmail("");}
  catch(e){setError(e instanceof Error?e.message:"Ошибка приглашения");}
 };
 const accept=async()=>{
  setError("");
  try{const r=await fetch("/api/family/accept",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({token:acceptToken})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось принять приглашение");setAcceptToken("");await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка принятия приглашения");}
 };
 const addTransaction=async()=>{
  setError("");
  try{const r=await fetch(editingTxId===null?"/api/family/transactions":`/api/family/transactions/${editingTxId}`,{method:editingTxId===null?"POST":"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({title,type:kind,amount:amount.replace(",", "."),category,date,comment:"",...(editingTxId===null?{idempotencyKey:crypto.randomUUID()}:{})})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось сохранить семейную операцию");setTransactions(old=>[d,...old.filter(x=>x.id!==d.id)]);setTitle("");setAmount("");setShowForm(false);setEditingTxId(null);await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка операции");}
 };
 const editTransaction=(t:FamilyTx)=>{
  setEditingTxId(t.id);setTitle(t.title);setKind(t.type);setAmount(t.amount);setCategory(t.category);setDate(t.date);setShowForm(true);
 };
 const removeTransaction=async(id:number)=>{
  if(!window.confirm("Удалить семейную операцию?"))return;
  try{const r=await fetch(`/api/family/transactions/${id}`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить операцию");setTransactions(old=>old.filter(x=>x.id!==id));await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка удаления");}
 };
 const removeMember=async(id:number)=>{
  if(!window.confirm("Удалить участника из семейного бюджета?"))return;
  try{const r=await fetch(`/api/family/members/${id}`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить участника");await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка удаления участника");}
 };
 const leave=async()=>{
  if(!window.confirm("Покинуть семейный бюджет?"))return;
  try{const r=await fetch("/api/family/leave",{method:"POST"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось покинуть семейный бюджет");await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка выхода");}
 };
 const currentMonth=today().slice(0,7);
 const monthExpense=transactions.filter(x=>x.type==="expense"&&x.date.startsWith(currentMonth)).reduce((s,x)=>s+cents(x.amount),0n);

 if(loading)return <div className="pure-page"><div className="empty-note">Загружаем семейный бюджет…</div></div>;
 return <div className="pure-page">
  <PageHeading title="Семья" subtitle="Общий бюджет и финансовые планы семьи"/>
  {!family?<div className="pure-grid two">
   <Panel title="Создать семейный бюджет" subtitle="Начальный баланс задаётся владельцем">
    <label className="form-label">Название<input className="pure-input" value={familyName} onChange={e=>setFamilyName(e.target.value)} maxLength={100}/></label>
    <label className="form-label" style={{marginTop:12}}>Начальный общий баланс, ₽<input className="pure-input" value={initialBalance} onChange={e=>setInitialBalance(e.target.value)}/></label>
    <button className="pure-button" style={{marginTop:14}} onClick={()=>void createFamily()} disabled={!familyName.trim()||!/^-?(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(initialBalance.trim())}><Plus size={15}/>Создать</button>
   </Panel>
   <Panel title="Принять приглашение" subtitle="Вставьте токен, полученный от владельца семьи">
    <label className="form-label">Токен приглашения<input className="pure-input" value={acceptToken} onChange={e=>setAcceptToken(e.target.value)} placeholder="Токен"/></label>
    <button className="pure-button" style={{marginTop:14}} disabled={!acceptToken.trim()} onClick={()=>void accept()}>Принять приглашение</button>
   </Panel>
  </div>:<>
   <div className="pure-grid three">
    <Stat label="Семейный баланс" value={money(family.currentBalance)} icon={<HomeIcon size={17}/>}/>
    <Stat label="Расходы за месяц" value={moneyCents(monthExpense)} tone="violet"/>
    <Stat label="Участники" value={String(members.length)} tone="mint" icon={<Users size={17}/>}/>
   </div>
   <div className="pure-grid two" style={{marginTop:16}}>
    <Panel title="Участники семьи" subtitle={family.name}>
     {members.map(m=><div className="member-row" key={m.id}>
      <div className="member-avatar">{(m.firstName[0]||"").toUpperCase()}{(m.lastName[0]||"").toUpperCase()}</div>
      <div className="member-copy"><strong>{m.firstName} {m.lastName}</strong><small>{m.role==="owner"?"Владелец":m.role==="admin"?"Администратор":"Участник"} · @{m.username}</small></div>
      {(family.role==="owner"||family.role==="admin")&&m.role!=="owner"&&<button className="pure-button ghost" aria-label="Удалить участника" onClick={()=>void removeMember(m.id)}><Trash2 size={14}/></button>}
     </div>)}
     {family.role!=="owner"&&<button className="pure-button ghost" style={{marginTop:12}} onClick={()=>void leave()}><LogOut size={14}/>Покинуть семью</button>}
    </Panel>
    <Panel title="Пригласить участника" subtitle="Приглашение действует 7 дней">
     {family.role==="owner"||family.role==="admin"?<>
      <label className="form-label">Email участника<input className="pure-input" value={inviteEmail} onChange={e=>setInviteEmail(e.target.value)} placeholder="name@example.com"/></label>
      <button className="pure-button" style={{marginTop:12}} disabled={!inviteEmail.includes("@")} onClick={()=>void invite()}><UserPlus size={15}/>Создать приглашение</button>
      {inviteToken&&<div className="empty-note" style={{marginTop:12}}>Передайте этот токен участнику вручную: <strong>{inviteToken}</strong></div>}
     </>:<div className="empty-note">Приглашать участников могут владелец и администраторы.</div>}
    </Panel>
   </div>
   <Panel title="Семейные операции" subtitle="Доходы и расходы общего бюджета" action={<button className="pure-button" onClick={()=>setShowForm(v=>!v)}><Plus size={15}/>Добавить</button>}>
    {showForm&&<div className="pure-grid three" style={{marginBottom:16}}>
     <label className="form-label">Тип<select className="pure-select" value={kind} onChange={e=>setKind(e.target.value as "income"|"expense")}><option value="expense">Расход</option><option value="income">Доход</option></select></label>
     <label className="form-label">Название<input className="pure-input" value={title} onChange={e=>setTitle(e.target.value)}/></label>
     <label className="form-label">Сумма, ₽<input className="pure-input" value={amount} onChange={e=>setAmount(e.target.value)}/></label>
     <label className="form-label">Категория<input className="pure-input" value={category} onChange={e=>setCategory(e.target.value)}/></label>
     <label className="form-label">Дата<input className="pure-input" type="date" value={date} onChange={e=>setDate(e.target.value)}/></label>
     <div className="form-row" style={{alignItems:"end"}}><button className="pure-button" disabled={!title.trim()||!/^\d+(?:[.,]\d{1,2})?$/.test(amount)||cents(amount.replace(",", "."))<=0n} onClick={()=>void addTransaction()}>Сохранить</button><button className="pure-button ghost" onClick={()=>{setShowForm(false);setEditingTxId(null);}}>Отмена</button></div>
    </div>}
    {transactions.length?<div className="transaction-list">{transactions.map(t=><div className="transaction-row" key={t.id}>
     <span className={`transaction-icon ${t.type==="income"?"mint":""}`}><HomeIcon size={16}/></span>
     <span className="transaction-copy"><strong>{t.title}</strong><small>{t.category} · {t.author} · {t.date}</small></span>
     <strong className={`transaction-amount ${t.type==="income"?"positive":"negative"}`}>{t.type==="income"?"+":"−"}{money(t.amount)}</strong>
     <button className="pure-button ghost" aria-label="Изменить операцию" onClick={()=>editTransaction(t)}><Pencil size={14}/></button><button className="pure-button ghost" aria-label="Удалить операцию" onClick={()=>void removeTransaction(t.id)}><Trash2 size={14}/></button>
    </div>)}</div>:<div className="empty-note">Семейных операций пока нет.</div>}
   </Panel>
   <div className="empty-note">Начальный семейный баланс и общие операции учитываются отдельно от личных счетов участников.</div>
  </>}
  {error&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
 </div>;
}
export default Family;
