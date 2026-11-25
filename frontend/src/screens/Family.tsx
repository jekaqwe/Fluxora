import { useEffect, useState, type ChangeEvent } from "react";
import { Plus, Users, Home as HomeIcon, Trash2, UserPlus, LogOut, Pencil, Target, Image } from "lucide-react";
import { ConfirmPopover, PageHeading, Panel, Stat } from "./ui";
import "./Pure.css";

type FamilyInfo={id:number;name:string;initialBalance:string;currentBalance:string;role:"owner"|"admin"|"member"};
type Member={id:number;userId:number;firstName:string;lastName:string;username:string;role:string;joinedAt:string};
type FamilyTx={id:number;title:string;type:"income"|"expense";amount:string;category:string;date:string;comment:string;author:string;createdAt:string};
type Account={id:number;name:string};
type FamilyGoal={id:number;title:string;description:string;targetAmount:string;startingAmount:string;currentAmount:string;dueDate?:string;priority:string;status:string;imageUrl?:string;createdBy?:number};
type FamilyGoalEntry={id:number;amount:string;entryType:"contribution"|"withdrawal";comment:string;author:string;createdAt:string};
type FamilyData={family:FamilyInfo|null;members:Member[]};
const cents=(v:string)=>{const [whole,fraction=""]=v.split(".");return BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));};
const moneyCents=(value:bigint)=>{const neg=value<0n;const abs=neg?-value:value;return (neg?"−":"")+new Intl.NumberFormat("ru-RU").format(abs/100n)+","+(abs%100n).toString().padStart(2,"0")+" ₽";};
const money=(v:string)=>{const neg=v.startsWith("-");const [whole,fraction=""]=v.replace("-","").split(".");const value=BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));return moneyCents(neg?-value:value);};
const today=()=>{const d=new Date();return [d.getFullYear(),String(d.getMonth()+1).padStart(2,"0"),String(d.getDate()).padStart(2,"0")].join("-");};

function Family(){
 const [family,setFamily]=useState<FamilyInfo|null>(null);
 const [members,setMembers]=useState<Member[]>([]);
 const [transactions,setTransactions]=useState<FamilyTx[]>([]);
 const [authorFilter,setAuthorFilter]=useState("");
 const [fromFilter,setFromFilter]=useState("");
 const [toFilter,setToFilter]=useState("");
 const [categoryFilter,setCategoryFilter]=useState("");
 const [familyGoals,setFamilyGoals]=useState<FamilyGoal[]>([]);
 const [accounts,setAccounts]=useState<Account[]>([]);
 const [familyGoalAccountId,setFamilyGoalAccountId]=useState("");
 const [showGoalForm,setShowGoalForm]=useState(false);
 const [goalTitle,setGoalTitle]=useState("");
 const [goalDescription,setGoalDescription]=useState("");
 const [goalTarget,setGoalTarget]=useState("");
 const [goalStarting,setGoalStarting]=useState("0");
 const [goalDueDate,setGoalDueDate]=useState("");
 const [goalPriority,setGoalPriority]=useState("medium");
 const [contributionGoalId,setContributionGoalId]=useState<number|null>(null);
 const [contributionAmount,setContributionAmount]=useState("");
 const [goalEntryType,setGoalEntryType]=useState<"contribution"|"withdrawal">("contribution");
 const [contributionKey,setContributionKey]=useState("");
 const [goalHistory,setGoalHistory]=useState<Record<number,FamilyGoalEntry[]>>({});
 const [goalHistoryId,setGoalHistoryId]=useState<number|null>(null);
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
 const [confirmAction,setConfirmAction]=useState<{title:string;message:string;confirmLabel?:string;onConfirm:()=>void}|null>(null);
 const askConfirmation=(title:string,message:string,onConfirm:()=>void,confirmLabel="Удалить")=>setConfirmAction({title,message,onConfirm,confirmLabel});

 const load=async()=>{
  setError("");
  try{
   const r=await fetch("/api/family");const d:FamilyData=await r.json();if(!r.ok)throw new Error("Не удалось загрузить семейный бюджет");
   setFamily(d.family);setMembers(d.members||[]);
   if(d.family){const [tx,goals,accountResponse]=await Promise.all([fetch("/api/family/transactions"),fetch("/api/family/goals"),fetch("/api/accounts")]);const [list,goalList,accountList]=await Promise.all([tx.json(),goals.json(),accountResponse.json()]);if(!tx.ok||!goals.ok||!accountResponse.ok)throw new Error("Не удалось загрузить семейные данные");setTransactions(list);setFamilyGoals(goalList);setAccounts(accountList);if(accountList.length&&!accountList.some((a:Account)=>String(a.id)===familyGoalAccountId))setFamilyGoalAccountId(String(accountList[0].id));}else{setTransactions([]);setFamilyGoals([]);}
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
 const createFamilyGoal=async()=>{
  try{const r=await fetch("/api/family/goals",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:goalTitle,description:goalDescription,targetAmount:goalTarget.replace(",", "."),startingAmount:goalStarting.replace(",", "."),dueDate:goalDueDate,priority:goalPriority})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось создать семейную цель");setFamilyGoals(old=>[d,...old]);setShowGoalForm(false);setGoalTitle("");setGoalDescription("");setGoalTarget("");setGoalStarting("0");}
  catch(e){setError(e instanceof Error?e.message:"Ошибка семейной цели");}
 };
 const contributeFamilyGoal=async()=>{
  if(contributionGoalId===null)return;
  try{const r=await fetch(`/api/family/goals/${contributionGoalId}/entries`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({amount:contributionAmount.replace(",", "."),entryType:goalEntryType,comment:"",accountId:Number(familyGoalAccountId),idempotencyKey:contributionKey})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось провести операцию по семейной цели");setContributionGoalId(null);setContributionAmount("");setContributionKey("");setGoalEntryType("contribution");await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка операции по цели");}
 };
 const showFamilyGoalHistory=async(id:number)=>{
  if(goalHistoryId===id){setGoalHistoryId(null);return;}
  try{const r=await fetch(`/api/family/goals/${id}/entries`);const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось загрузить историю");setGoalHistory(old=>({...old,[id]:d}));setGoalHistoryId(id);}
  catch(e){setError(e instanceof Error?e.message:"Ошибка истории");}
 };
 const uploadFamilyGoalImage=async(id:number,event:ChangeEvent<HTMLInputElement>)=>{
  const file=event.target.files?.[0];event.target.value="";if(!file)return;
  try{const form=new FormData();form.append("image",file);const r=await fetch(`/api/family/goals/${id}/image`,{method:"PUT",body:form});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось загрузить изображение");setFamilyGoals(old=>old.map(g=>g.id===id?{...g,imageUrl:d.imageUrl}:g));}
  catch(e){setError(e instanceof Error?e.message:"Ошибка изображения");}
 };
 const removeTransaction=async(id:number)=>{
  try{const r=await fetch(`/api/family/transactions/${id}`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить операцию");setTransactions(old=>old.filter(x=>x.id!==id));await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка удаления");}
 };
 const updateRole=async(id:number,role:"admin"|"member")=>{
  try{const r=await fetch(`/api/family/members/${id}`,{method:"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({role})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось изменить роль");await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка роли");}
 };
 const removeMember=async(id:number)=>{
  try{const r=await fetch(`/api/family/members/${id}`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить участника");await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка удаления участника");}
 };
 const leave=async()=>{
  try{const r=await fetch("/api/family/leave",{method:"POST"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось покинуть семейный бюджет");await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка выхода");}
 };
 const currentMonth=today().slice(0,7);
 const monthExpense=transactions.filter(x=>x.type==="expense"&&x.date.startsWith(currentMonth)).reduce((s,x)=>s+cents(x.amount),0n);
 const authors=Array.from(new Set(transactions.map(item=>item.author))).sort((a,b)=>a.localeCompare(b,"ru"));
 const familyCategories=Array.from(new Set(transactions.map(item=>item.category))).sort((a,b)=>a.localeCompare(b,"ru"));
 const filteredTransactions=transactions.filter(item=>(!authorFilter||item.author===authorFilter)&&(!fromFilter||item.date>=fromFilter)&&(!toFilter||item.date<=toFilter)&&(!categoryFilter||item.category===categoryFilter));
 const filteredIncome=filteredTransactions.filter(item=>item.type==="income").reduce((sum,item)=>sum+cents(item.amount),0n);
 const filteredExpense=filteredTransactions.filter(item=>item.type==="expense").reduce((sum,item)=>sum+cents(item.amount),0n);

 if(loading)return <div className="pure-page" style={{padding:16}}><div className="empty-note">Загружаем семейный бюджет…</div></div>;
 return <div className="pure-page" style={{padding:16}}>
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
      {family.role==="owner"&&m.role!=="owner"&&<select className="pure-select" aria-label="Роль участника" value={m.role} onChange={e=>void updateRole(m.id,e.target.value as "admin"|"member")}><option value="member">Участник</option><option value="admin">Администратор</option></select>}{(family.role==="owner"||family.role==="admin")&&m.role!=="owner"&&<button className="pure-button ghost" aria-label="Удалить участника" onClick={()=>askConfirmation("Удалить участника?","Участник будет удалён из семейного бюджета.",()=>removeMember(m.id))}><Trash2 size={14}/></button>}
     </div>)}
     {family.role!=="owner"&&<button className="pure-button ghost" style={{marginTop:12}} onClick={()=>askConfirmation("Покинуть семейный бюджет?","Вы потеряете доступ к общему бюджету и семейным операциям.",leave,"Покинуть")}><LogOut size={14}/>Покинуть семью</button>}
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
     <div className="form-row" style={{alignItems:"end"}}><button className="pure-button" disabled={!title.trim()||!/^\d+(?:[.,]\d{1,2})?$/.test(amount)||cents(amount.replace(",", "."))<=0n} onClick={()=>void addTransaction()}>{editingTxId===null?"Сохранить":"Сохранить изменения"}</button><button className="pure-button ghost" onClick={()=>{setShowForm(false);setEditingTxId(null);}}>Отмена</button></div>
    </div>}
    <div className="pure-grid three" style={{marginBottom:12}}>
     <label className="form-label">Участник<select className="pure-select" value={authorFilter} onChange={e=>setAuthorFilter(e.target.value)}><option value="">Все участники</option>{authors.map(author=><option key={author} value={author}>{author}</option>)}</select></label>
     <label className="form-label">С даты<input className="pure-input" type="date" value={fromFilter} max={toFilter||undefined} onChange={e=>setFromFilter(e.target.value)}/></label>
     <label className="form-label">По дату<input className="pure-input" type="date" value={toFilter} min={fromFilter||undefined} onChange={e=>setToFilter(e.target.value)}/></label>
     <label className="form-label">Категория<select className="pure-select" value={categoryFilter} onChange={e=>setCategoryFilter(e.target.value)}><option value="">Все категории</option>{familyCategories.map(item=><option key={item} value={item}>{item}</option>)}</select></label>
    </div>
    <div className="pure-grid two" style={{marginBottom:12}}>
     <Stat label="Доходы по фильтру" value={money(filteredIncome.toString())} tone="mint"/>
     <Stat label="Расходы по фильтру" value={money(filteredExpense.toString())} tone="violet"/>
    </div>
    {filteredTransactions.length?<div className="transaction-list">{filteredTransactions.map(t=><div className="transaction-row" key={t.id}>
     <span className={`transaction-icon ${t.type==="income"?"mint":""}`}><HomeIcon size={16}/></span>
     <span className="transaction-copy"><strong>{t.title}</strong><small>{t.category} · {t.author} · {t.date}</small></span>
     <strong className={`transaction-amount ${t.type==="income"?"positive":"negative"}`}>{t.type==="income"?"+":"−"}{money(t.amount)}</strong>
     <button className="pure-button ghost" aria-label="Изменить операцию" onClick={()=>editTransaction(t)}><Pencil size={14}/></button><button className="pure-button ghost" aria-label="Удалить операцию" onClick={()=>askConfirmation("Удалить семейную операцию?","Операция будет удалена из истории.",()=>removeTransaction(t.id))}><Trash2 size={14}/></button>
    </div>)}</div>:<div className="empty-note">Операций по выбранным фильтрам нет.</div>}
   </Panel>
   <Panel title="Общие семейные цели" subtitle="Участники могут пополнять накопления и снимать средства с учётом остатка" action={<button className="pure-button" onClick={()=>setShowGoalForm(v=>!v)}><Plus size={15}/>Создать цель</button>}>
    {showGoalForm&&<div className="pure-grid three" style={{marginBottom:14}}>
     <label className="form-label">Название<input className="pure-input" value={goalTitle} onChange={e=>setGoalTitle(e.target.value)}/></label>
     <label className="form-label">Целевая сумма<input className="pure-input" value={goalTarget} onChange={e=>setGoalTarget(e.target.value)}/></label>
     <label className="form-label">Уже накоплено<input className="pure-input" value={goalStarting} onChange={e=>setGoalStarting(e.target.value)}/></label>
     <label className="form-label">Срок<input className="pure-input" type="date" value={goalDueDate} onChange={e=>setGoalDueDate(e.target.value)}/></label>
     <label className="form-label">Приоритет<select className="pure-select" value={goalPriority} onChange={e=>setGoalPriority(e.target.value)}><option value="low">Низкий</option><option value="medium">Средний</option><option value="high">Высокий</option></select></label>
     <label className="form-label">Описание<input className="pure-input" value={goalDescription} onChange={e=>setGoalDescription(e.target.value)}/></label>
     <button className="pure-button" disabled={!goalTitle.trim()||!/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(goalTarget)||cents(goalTarget.replace(",", "."))<=0n} onClick={()=>void createFamilyGoal()}>Сохранить цель</button>
    </div>}
    {familyGoals.length===0?<div className="empty-note">Семейных целей пока нет.</div>:<div className="pure-grid two">{familyGoals.map(g=>{const current=cents(g.currentAmount);const targetCents=cents(g.targetAmount);const pct=Math.max(0,Math.min(100,Number(current*10000n/targetCents)/100));return <div className="goal-card" key={g.id}>
     <div className="goal-art">{g.imageUrl?<img src={g.imageUrl} alt={g.title} style={{width:"100%",height:"100%",objectFit:"cover"}}/>:<Target/>}</div>
     <div className="goal-content"><h3>{g.title}</h3><p>{g.description||"Семейная цель"}</p><strong>{pct.toFixed(1).replace(".",",")}%</strong><div className="goal-meta"><span>{money(g.currentAmount)}</span><span>{money(g.targetAmount)}</span></div><div className="progress-track"><div className="progress-fill" style={{width:pct+"%"}}/></div><small className="muted">Срок: {g.dueDate||"не задан"} · {g.status==="completed"?"Завершена":g.status==="paused"?"Приостановлена":"Активна"}</small>
      <div className="form-row" style={{marginTop:10,flexWrap:"wrap"}}><button className="pure-button" disabled={g.status!=="active"} onClick={()=>{setContributionGoalId(g.id);setGoalEntryType("contribution");setContributionKey(crypto.randomUUID());}}>Пополнить</button><button className="pure-button ghost" disabled={g.status==="paused"} onClick={()=>{setContributionGoalId(g.id);setGoalEntryType("withdrawal");setContributionKey(crypto.randomUUID());}}>Снять</button><button className="pure-button ghost" onClick={()=>void showFamilyGoalHistory(g.id)}>История</button><label className="pure-button ghost"><Image size={14}/><input type="file" hidden accept="image/jpeg,image/png,image/webp,image/gif" onChange={e=>void uploadFamilyGoalImage(g.id,e)}/></label></div>
      {contributionGoalId===g.id&&<div style={{marginTop:10}}><strong>{goalEntryType==="withdrawal"?"Снятие с семейной цели":"Пополнение семейной цели"}</strong><label className="form-label">Сумма<input className="pure-input" value={contributionAmount} onChange={e=>setContributionAmount(e.target.value)}/></label><label className="form-label" style={{marginTop:8}}>Личный счёт<select className="pure-select" value={familyGoalAccountId} onChange={e=>setFamilyGoalAccountId(e.target.value)}>{accounts.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</select></label><div className="form-row" style={{marginTop:8}}><button className="pure-button" disabled={!familyGoalAccountId||!/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(contributionAmount)||cents(contributionAmount.replace(",", "."))<=0n} onClick={()=>void contributeFamilyGoal()}>{goalEntryType==="withdrawal"?"Подтвердить снятие":"Подтвердить пополнение"}</button><button className="pure-button ghost" onClick={()=>setContributionGoalId(null)}>Отмена</button></div></div>}
      {goalHistoryId===g.id&&<div className="empty-note" style={{marginTop:10}}>{(goalHistory[g.id]||[]).length?(goalHistory[g.id]||[]).map(entry=><div className="legend-row" key={entry.id}><span>{entry.author} · {new Date(entry.createdAt).toLocaleDateString("ru-RU")}</span><strong>{entry.entryType==="withdrawal"?"−":"+"}{money(entry.amount)} <small>{entry.entryType==="withdrawal"?"Снятие":"Пополнение"}</small></strong></div>):"История пуста."}</div>}
     </div></div>})}</div>}
   </Panel>
   <div className="empty-note">Начальный семейный баланс и общие операции учитываются отдельно от личных счетов участников.</div>
  </>}
  {confirmAction&&<ConfirmPopover title={confirmAction.title} message={confirmAction.message} confirmLabel={confirmAction.confirmLabel} onCancel={()=>setConfirmAction(null)} onConfirm={()=>{const action=confirmAction;setConfirmAction(null);void action.onConfirm();}}/>}
  {error&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
 </div>;
}
export default Family;
