import { useEffect, useState, type ChangeEvent } from "react";
import { Plus, Home as HomeIcon, Trash2, UserPlus, LogOut, Pencil, Target, Image, X, ChevronDown, ChevronLeft, ChevronRight, Wallet, TrendingUp, TrendingDown, ArrowLeftRight, HandCoins, Baby, House, Heart, BriefcaseBusiness, Stethoscope, Utensils, Flame, Car, GraduationCap, Gift, Repeat, ShoppingBag, ShoppingCart, Plane, Gamepad2, Bus, CalendarDays, Users, Tag } from "lucide-react";
import { ConfirmPopover, PageHeading, Panel, Stat, Progress } from "./ui";
import "./Pure.css";

type FamilyInfo={id:number;name:string;initialBalance:string;currentBalance:string;role:"owner"|"admin"|"member"};
type Member={id:number;userId:number;firstName:string;lastName:string;username:string;role:string;joinedAt:string};
type FamilyTx={id:number;title:string;type:string;amount:string;category:string;date:string;comment:string;author:string;createdAt:string};
type Account={id:number;name:string};
type FamilyGoal={id:number;title:string;description:string;targetAmount:string;startingAmount:string;currentAmount:string;dueDate?:string;priority:string;status:string;imageUrl?:string;createdBy?:number};
type FamilyGoalEntry={id:number;amount:string;entryType:"contribution"|"withdrawal";comment:string;author:string;createdAt:string};
type FamilyData={family:FamilyInfo|null;members:Member[]};
const cents=(v:string)=>{const [whole,fraction=""]=v.split(".");return BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));};
const moneyCents=(value:bigint)=>{const neg=value<0n;const abs=neg?-value:value;return (neg?"−":"")+new Intl.NumberFormat("ru-RU").format(abs/100n)+","+(abs%100n).toString().padStart(2,"0")+" ₽";};
const money=(v:string)=>{const neg=v.startsWith("-");const [whole,fraction=""]=v.replace("-","").split(".");const value=BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));return moneyCents(neg?-value:value);};
const today=()=>{const d=new Date();return [d.getFullYear(),String(d.getMonth()+1).padStart(2,"0"),String(d.getDate()).padStart(2,"0")].join("-");};
const validFamilyGoalTarget=(value:string)=>/^(?:0|[1-9][0-9]{0,8})(?:[.,][0-9]{1,2})?$/.test(value)&&cents(value.replace(",", "."))>0n&&cents(value.replace(",", "."))<=99999999900n;
const validFamilyGoalStarting=(value:string)=>/^(?:0|[1-9][0-9]{0,8})(?:[.,][0-9]{1,2})?$/.test(value)&&cents(value.replace(",", "."))>=0n&&cents(value.replace(",", "."))<=99999999900n;

function Family(){
 const [family,setFamily]=useState<FamilyInfo|null>(null);
 const [members,setMembers]=useState<Member[]>([]);
 const [transactions,setTransactions]=useState<FamilyTx[]>([]);
 const [authorFilter,setAuthorFilter]=useState("");
 const [fromFilter,setFromFilter]=useState("");
 const [toFilter,setToFilter]=useState("");
 const [categoryFilter,setCategoryFilter]=useState("");
 const [typeFilter,setTypeFilter]=useState("");
 const [openFilter,setOpenFilter]=useState<"member"|"period"|"type"|"category"|null>(null);
 const [historyPage,setHistoryPage]=useState(1);
 const [draftFrom,setDraftFrom]=useState("");
 const [draftTo,setDraftTo]=useState("");
 const [familyGoals,setFamilyGoals]=useState<FamilyGoal[]>([]);
 const [accounts,setAccounts]=useState<Account[]>([]);
 const [familyGoalAccountId,setFamilyGoalAccountId]=useState("");
 const [showGoalForm,setShowGoalForm]=useState(false);
 const [editingFamilyGoalId,setEditingFamilyGoalId]=useState<number|null>(null);
 const [goalLimitExceeded,setGoalLimitExceeded]=useState(false);
 const [goalInvalidFields,setGoalInvalidFields]=useState<Record<string,boolean>>({});
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
 const [inviteTicketEmail,setInviteTicketEmail]=useState("");
 const [showInviteForm,setShowInviteForm]=useState(false);
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
  try{const r=await fetch("/api/family/invitations",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({email:inviteEmail})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось создать приглашение");setInviteToken(d.token);setInviteTicketEmail(inviteEmail);setInviteEmail("");}
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
 const resetFamilyGoalForm=()=>{setShowGoalForm(false);setEditingFamilyGoalId(null);setGoalLimitExceeded(false);setGoalInvalidFields({});setGoalTitle("");setGoalDescription("");setGoalTarget("");setGoalStarting("0");setGoalDueDate("");setGoalPriority("medium");};
 const beginEditFamilyGoal=(g:FamilyGoal)=>{setGoalInvalidFields({});setEditingFamilyGoalId(g.id);setGoalTitle(g.title);setGoalDescription(g.description);setGoalTarget(g.targetAmount);setGoalStarting(g.startingAmount);setGoalDueDate(g.dueDate||"");setGoalPriority(g.priority);setGoalLimitExceeded(false);setShowGoalForm(true);};
 const saveFamilyGoal=async()=>{
  if(editingFamilyGoalId===null&&familyGoals.length>=3){setGoalLimitExceeded(true);return;}
  const invalid={title:!goalTitle.trim()||goalTitle.length>24,description:goalDescription.length>30,target:!validFamilyGoalTarget(goalTarget),starting:!validFamilyGoalStarting(goalStarting)};
  setGoalInvalidFields(invalid);
  if(Object.values(invalid).some(Boolean)){setError("Проверьте выделенные поля: максимальная сумма 999 999 999 ₽.");return;}
  setError("");
  try{const editing=editingFamilyGoalId!==null;const current=editing?familyGoals.find(g=>g.id===editingFamilyGoalId):undefined;const r=await fetch(editing?`/api/family/goals/${editingFamilyGoalId}`:"/api/family/goals",{method:editing?"PATCH":"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({title:goalTitle,description:goalDescription,targetAmount:goalTarget.replace(",", "."),startingAmount:goalStarting.replace(",", "."),dueDate:goalDueDate,priority:goalPriority,status:current?.status||"active"})});const d=await r.json();if(!r.ok)throw new Error(d.error||(editing?"Не удалось изменить семейную цель":"Не удалось создать семейную цель"));resetFamilyGoalForm();await load();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка семейной цели");}
 };
 const removeFamilyGoal=async(id:number)=>{try{const r=await fetch(`/api/family/goals/${id}`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить семейную цель");setFamilyGoals(old=>old.filter(g=>g.id!==id));}catch(e){setError(e instanceof Error?e.message:"Ошибка удаления семейной цели");}};
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
 const familyCategoryOptions=[["Дети",Baby],["Дом/уют",House],["Забота о себе",Heart],["Зарплата",BriefcaseBusiness],["Здоровье",Stethoscope],["Кафе/рестораны",Utensils],["ЖКХ",Flame],["Машина",Car],["Образование",GraduationCap],["Подарки",Gift],["Подписки",Repeat],["Покупки",ShoppingBag],["Продукты",ShoppingCart],["Путешествия",Plane],["Развлечения",Gamepad2],["Общественный транспорт",Bus]] as const;
 const familyTypeOptions=[["Расход","expense",TrendingDown],["Доход","income",TrendingUp],["Перевод","transfer",ArrowLeftRight],["Долг","debt",HandCoins]] as const;
 const categoryIcon=(value:string)=>familyCategoryOptions.find(x=>x[0].toLowerCase()===value.toLowerCase())?.[1]||Tag;
 const typeIcon=(value:string)=>familyTypeOptions.find(x=>x[1]===value)?.[2]||Wallet;
 const formatDate=(value:string)=>value?new Date(value+"T00:00:00").toLocaleDateString("ru-RU"):"";

 const monthIncome=transactions.filter(x=>x.type==="income"&&x.date.startsWith(currentMonth)).reduce((s,x)=>s+cents(x.amount),0n);
 const monthExpense=transactions.filter(x=>x.type==="expense"&&x.date.startsWith(currentMonth)).reduce((s,x)=>s+cents(x.amount),0n);
 const authors=Array.from(new Set([...members.map(m=>m.firstName+" "+m.lastName),...transactions.map(item=>item.author)])).sort((a,b)=>a.localeCompare(b,"ru"));
 const filteredTransactions=transactions.filter(item=>(!authorFilter||item.author===authorFilter)&&(!fromFilter||item.date>=fromFilter)&&(!toFilter||item.date<=toFilter)&&(!categoryFilter||item.category.toLowerCase()===categoryFilter.toLowerCase())&&(!typeFilter||item.type===typeFilter));
 const filteredIncome=filteredTransactions.filter(item=>item.type==="income").reduce((sum,item)=>sum+cents(item.amount),0n);
 const filteredExpense=filteredTransactions.filter(item=>item.type==="expense").reduce((sum,item)=>sum+cents(item.amount),0n);
 const historyPageSize=20;
 const historyPages=Math.max(1,Math.ceil(filteredTransactions.length/historyPageSize));
 const pagedTransactions=filteredTransactions.slice((historyPage-1)*historyPageSize,historyPage*historyPageSize);
 useEffect(()=>{setHistoryPage(1);},[authorFilter,fromFilter,toFilter,categoryFilter,typeFilter]);
 useEffect(()=>{setHistoryPage(page=>Math.min(page,historyPages));},[historyPages]);

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
    <Stat label="Семейный доход" value={moneyCents(monthIncome)} tone="mint"/>
    <Stat label="Семейный расход" value={moneyCents(monthExpense)} tone="violet"/>
   </div>
   <div style={{marginTop:16}}>
    <Panel title="Участники семьи" subtitle={family.name}>
     <div style={{display:"grid",gridTemplateColumns:"repeat(3,minmax(0,1fr))",borderTop:"1px solid #293953",borderLeft:"1px solid #293953"}}>
      {Array.from({length:6},(_,index)=>{const m=members[index];const canInvite=family.role==="owner"||family.role==="admin";return <div key={m?m.id:"empty-"+index} style={{minWidth:0,minHeight:94,padding:"10px 8px",display:"flex",alignItems:"center",gap:8,borderRight:"1px solid #293953",borderBottom:"1px solid #293953"}}>
       {m?<><div className="member-avatar" style={{flex:"0 0 34px"}}>{(m.firstName[0]||"").toUpperCase()}{(m.lastName[0]||"").toUpperCase()}</div><div className="member-copy"><strong>{m.firstName} {m.lastName}</strong><small>{m.role==="owner"?"Владелец":m.role==="admin"?"Администратор":"Участник"} · @{m.username}</small></div>{family.role==="owner"&&m.role!=="owner"&&<select className="pure-select" aria-label="Роль участника" style={{width:96,minWidth:0,height:34,minHeight:34,padding:"0 5px",fontSize:10}} value={m.role} onChange={e=>void updateRole(m.id,e.target.value as "admin"|"member")}><option value="member">Участник</option><option value="admin">Админ</option></select>}{canInvite&&m.role!=="owner"&&<button className="pure-button ghost" aria-label="Удалить участника" style={{width:30,height:30,minWidth:30,minHeight:30,padding:0}} onClick={()=>askConfirmation("Удалить участника?","Участник будет удалён из семейного бюджета.",()=>removeMember(m.id))}><Trash2 size={15}/></button>}</>:canInvite?<button className="pure-button secondary" style={{width:"100%",minHeight:36,padding:"0 6px",fontSize:10}} onClick={()=>{setInviteEmail("");setInviteToken("");setInviteTicketEmail("");setShowInviteForm(true);}}><UserPlus size={14}/>Пригласить</button>:<span className="muted" style={{fontSize:10}}>Свободное место</span>}
      </div>;})}
     </div>
     {members.length>6&&<p className="muted" style={{fontSize:10,marginBottom:0}}>Показаны первые 6 участников</p>}
     {family.role!=="owner"&&<button className="pure-button ghost" style={{marginTop:12}} onClick={()=>askConfirmation("Покинуть семейный бюджет?","Вы потеряете доступ к общему бюджету и семейным операциям.",leave,"Покинуть")}><LogOut size={14}/>Покинуть семью</button>}
    </Panel>
   </div>
   <div style={{marginTop:16}}><Panel title="Семейные операции" subtitle="Доходы и расходы общего бюджета" action={<button className="pure-button compact-tool-button" onClick={()=>{setEditingTxId(null);setTitle("");setAmount("");setKind("expense");setCategory("Покупки");setDate(today());setShowForm(true);}}><Plus size={14}/>Добавить операцию</button>}>
    <div style={{display:"flex",flexWrap:"wrap",gap:8,marginBottom:14}}>
     <div style={{position:"relative"}}><button className={"filter-pill compact-tool-button "+(authorFilter?"active":"")} onClick={()=>setOpenFilter(openFilter==="member"?null:"member")}><Users size={13}/>Участник{authorFilter?": "+authorFilter:""}<ChevronDown size={12}/></button>{openFilter==="member"&&<div className="family-filter-menu">{["",...authors].map(a=><button key={a||"all"} className={authorFilter===a?"selected":""} onClick={()=>{setAuthorFilter(a);setOpenFilter(null);}}>{a||"Все участники"}</button>)}</div>}</div>
     <div style={{position:"relative"}}><button className={"filter-pill compact-tool-button "+((fromFilter||toFilter)?"active":"")} onClick={()=>{setDraftFrom(fromFilter);setDraftTo(toFilter);setOpenFilter(openFilter==="period"?null:"period");}}><CalendarDays size={13}/>{fromFilter||toFilter?formatDate(fromFilter)+"-"+formatDate(toFilter):"Период"}<ChevronDown size={12}/></button>{openFilter==="period"&&<div className="family-filter-menu family-period-menu"><label className="form-label">С какого<input className="pure-input" type="date" value={draftFrom} max={draftTo||undefined} onChange={e=>setDraftFrom(e.target.value)}/></label><label className="form-label">По какой<input className="pure-input" type="date" value={draftTo} min={draftFrom||undefined} onChange={e=>setDraftTo(e.target.value)}/></label><div className="form-row"><button className="pure-button compact-tool-button" onClick={()=>{setFromFilter(draftFrom);setToFilter(draftTo);setOpenFilter(null);}}>Применить</button><button className="pure-button ghost compact-tool-button" onClick={()=>{setFromFilter("");setToFilter("");setDraftFrom("");setDraftTo("");setOpenFilter(null);}}>Сбросить</button></div></div>}</div>
     <div style={{position:"relative"}}><button className={"filter-pill compact-tool-button "+(typeFilter?"active":"")} onClick={()=>setOpenFilter(openFilter==="type"?null:"type")}><Wallet size={13}/>Тип{typeFilter?": "+(familyTypeOptions.find(x=>x[1]===typeFilter)?.[0]||typeFilter):""}<ChevronDown size={12}/></button>{openFilter==="type"&&<div className="family-filter-menu">{[["Всё","",Wallet] as const,...familyTypeOptions].map(([label,value,Icon])=><button key={value||"all"} className={typeFilter===value?"selected":""} onClick={()=>{setTypeFilter(value);setOpenFilter(null);}}><Icon size={14}/>{label}</button>)}</div>}</div>
     <div style={{position:"relative"}}><button className={"filter-pill compact-tool-button "+(categoryFilter?"active":"")} onClick={()=>setOpenFilter(openFilter==="category"?null:"category")}><Tag size={13}/>Категория{categoryFilter?": "+categoryFilter:""}<ChevronDown size={12}/></button>{openFilter==="category"&&<div className="family-filter-menu family-category-menu"><button className={!categoryFilter?"selected":""} onClick={()=>{setCategoryFilter("");setOpenFilter(null);}}>Все категории</button>{familyCategoryOptions.map(([name,Icon])=><button key={name} className={categoryFilter.toLowerCase()===name.toLowerCase()?"selected":""} onClick={()=>{setCategoryFilter(name);setOpenFilter(null);}}><Icon size={14}/>{name}</button>)}</div>}</div>
    </div>
    <div className="pure-grid two family-filter-stats" style={{marginBottom:12}}><Stat label="Доход" value={moneyCents(filteredIncome)} tone="mint"/><Stat label="Расход" value={moneyCents(filteredExpense)} tone="violet"/></div>
    {filteredTransactions.length?<><div className="transaction-list operations-history-list">{pagedTransactions.map(t=>{const Icon=typeIcon(t.type);const CategoryIcon=categoryIcon(t.category);return <div className="transaction-row" key={t.id}>
     <span className={"transaction-icon "+(t.type==="income"?"mint":t.type==="expense"?"coral":"orange")}><Icon size={15}/></span><span className="transaction-copy"><strong>{t.title}</strong><small><CategoryIcon size={11} style={{verticalAlign:"middle"}}/> {t.category} · {t.author} · {formatDate(t.date)}</small></span><strong className={"transaction-amount "+(t.type==="income"?"positive":"negative")}>{t.type==="income"?"+":t.type==="expense"?"−":""}{money(t.amount)}</strong><button className="pure-button ghost operation-edit-button" aria-label="Изменить операцию" onClick={()=>editTransaction(t)}><Pencil size={14}/></button><button className="pure-button ghost operation-delete-button" aria-label="Удалить операцию" onClick={()=>askConfirmation("Удалить семейную операцию?","Операция будет удалена из истории.",()=>removeTransaction(t.id))}><Trash2 size={13}/></button>
    </div>;})}</div>{historyPages>1&&<div className="operations-pagination"><button className="pure-button ghost" disabled={historyPage<=1} onClick={()=>setHistoryPage(p=>Math.max(1,p-1))}><ChevronLeft size={14}/>Назад</button><span>Страница {historyPage} из {historyPages}</span><button className="pure-button ghost" disabled={historyPage>=historyPages} onClick={()=>setHistoryPage(p=>Math.min(historyPages,p+1))}>Вперёд<ChevronRight size={14}/></button></div>}</>:<div className="empty-note">Операций по выбранным фильтрам нет.</div>}
   </Panel></div>
   <div style={{marginTop:16}}><Panel title="Семейные цели" subtitle="Общие накопления семьи" action={<button className="pure-button" onClick={()=>{resetFamilyGoalForm();setGoalLimitExceeded(familyGoals.length>=3);setShowGoalForm(true);}}><Plus size={15}/>Создать цель</button>}>

    {familyGoals.length===0?<div className="empty-note">Семейных целей пока нет. Создайте первую и начните копить на важное.</div>:<div className="pure-grid three">{familyGoals.map(g=>{const current=cents(g.currentAmount);const targetCents=cents(g.targetAmount);const pct=targetCents>0n?Math.max(0,Math.min(100,Number(current*10000n/targetCents)/100)):0;return <Panel key={g.id} className="goal-card"><div className="goal-art">{g.imageUrl?<img src={g.imageUrl} alt={g.title} style={{width:"100%",height:"100%",objectFit:"cover"}}/>:<Target/>}</div><div className="goal-content"><div className="form-row" style={{justifyContent:"space-between"}}><h3>{g.title}</h3><span className="muted">{g.priority==="high"?"Высокий":g.priority==="low"?"Низкий":"Средний"}</span></div><p>{g.description||"Семейная цель"}</p><strong style={{fontSize:21}}>{pct.toFixed(1).replace(".",",")}%</strong><div className="goal-meta"><span>{money(g.currentAmount)}</span><span>{money(g.targetAmount)}</span></div><Progress value={pct}/><div className="goal-meta"><span>Срок: {g.dueDate||"не задан"}</span><span>{moneyCents(targetCents>current?targetCents-current:0n)}</span></div><span className="muted">{g.status==="paused"?"Приостановлена":g.status==="completed"?"Завершена":"Активна"}</span><div className="form-row" style={{marginTop:10,flexWrap:"wrap"}}><button className="pure-button secondary" disabled={g.status!=="active"} onClick={()=>{setContributionGoalId(g.id);setGoalEntryType("contribution");setContributionKey(crypto.randomUUID());}}>Пополнить</button><button className="pure-button ghost" disabled={g.status==="paused"} onClick={()=>{setContributionGoalId(g.id);setGoalEntryType("withdrawal");setContributionKey(crypto.randomUUID());}}>Снять</button><button className="pure-button ghost" onClick={()=>beginEditFamilyGoal(g)}><Pencil size={14}/></button><button className="pure-button ghost" onClick={()=>void showFamilyGoalHistory(g.id)}>История</button><label className="pure-button ghost" title="Изменить изображение"><Image size={14}/><input type="file" hidden accept="image/jpeg,image/png,image/webp,image/gif" onChange={e=>void uploadFamilyGoalImage(g.id,e)}/></label><button className="pure-button ghost" onClick={()=>askConfirmation("Удалить семейную цель?","Цель без истории операций будет удалена.",()=>removeFamilyGoal(g.id))}><Trash2 size={14}/></button></div>{contributionGoalId===g.id&&<div style={{marginTop:10}}><strong>{goalEntryType==="withdrawal"?"Снятие с семейной цели":"Пополнение семейной цели"}</strong><label className="form-label">Сумма<input className="pure-input" value={contributionAmount} onChange={e=>setContributionAmount(e.target.value)}/></label><label className="form-label" style={{marginTop:8}}>Личный счёт<select className="pure-select" value={familyGoalAccountId} onChange={e=>setFamilyGoalAccountId(e.target.value)}>{accounts.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</select></label><div className="form-row" style={{marginTop:8}}><button className="pure-button" disabled={!familyGoalAccountId||!/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(contributionAmount)||cents(contributionAmount.replace(",", "."))<=0n} onClick={()=>void contributeFamilyGoal()}>{goalEntryType==="withdrawal"?"Подтвердить снятие":"Подтвердить пополнение"}</button><button className="pure-button ghost" onClick={()=>setContributionGoalId(null)}>Отмена</button></div></div>}{goalHistoryId===g.id&&<div className="empty-note" style={{marginTop:10}}>{(goalHistory[g.id]||[]).length?(goalHistory[g.id]||[]).map(entry=><div className="legend-row" key={entry.id}><span>{entry.author} · {new Date(entry.createdAt).toLocaleDateString("ru-RU")}</span><strong>{entry.entryType==="withdrawal"?"−":"+"}{money(entry.amount)} <small>{entry.entryType==="withdrawal"?"Снятие":"Пополнение"}</small></strong></div>):"История пуста."}</div>}</div></Panel>;})}</div>}
   </Panel></div>
   <div className="empty-note" style={{marginTop:16}}>Начальный семейный баланс и общие операции учитываются отдельно от личных счетов участников.</div>
  </>}
  {showForm&&<div className="operation-modal-backdrop" onMouseDown={e=>{if(e.target===e.currentTarget){setShowForm(false);setEditingTxId(null);}}}><section className="debt-modal operation-modal" role="dialog" aria-modal="true" aria-labelledby="family-operation-title"><div className="debt-modal-heading"><div><strong id="family-operation-title">{editingTxId===null?"Добавить семейную операцию":"Изменить семейную операцию"}</strong><small>Заполните данные общей операции</small></div><button type="button" className="pure-button ghost debt-modal-close" aria-label="Закрыть" onClick={()=>{setShowForm(false);setEditingTxId(null);}}><X size={16}/></button></div><div className="pure-grid two"><label className="form-label">Тип операции<select className="pure-select" value={kind} onChange={e=>setKind(e.target.value as "income"|"expense")}><option value="expense">Расход</option><option value="income">Доход</option></select></label><label className="form-label">Название<input className="pure-input" value={title} onChange={e=>setTitle(e.target.value)} maxLength={160}/></label><label className="form-label">Сумма, ₽<input className="pure-input" inputMode="decimal" value={amount} onChange={e=>setAmount(e.target.value)} placeholder="1000,00"/></label><label className="form-label">Категория<select className="pure-select" value={category} onChange={e=>setCategory(e.target.value)}>{familyCategoryOptions.map(([name])=><option key={name} value={name}>{name}</option>)}</select></label><label className="form-label">Дата<input className="pure-input" type="date" value={date} onChange={e=>setDate(e.target.value)}/></label></div><div className="debt-modal-actions"><button className="pure-button" disabled={!title.trim()||!/^(?:0|[1-9][0-9]*)(?:[.,][0-9]{1,2})?$/.test(amount.trim())||cents(amount.replace(",", "."))<=0n} onClick={()=>void addTransaction()}>{editingTxId===null?"Сохранить операцию":"Сохранить изменения"}</button><button className="pure-button ghost" onClick={()=>{setShowForm(false);setEditingTxId(null);}}>Отмена</button></div></section></div>}
  {showInviteForm&&<div className="debt-modal-backdrop" onMouseDown={e=>{if(e.target===e.currentTarget)setShowInviteForm(false);}}><section className="debt-modal" role="dialog" aria-modal="true" aria-labelledby="family-invite-title"><div className="debt-modal-heading"><div><strong id="family-invite-title">Пригласить участника</strong><small>Приглашение действует 7 дней</small></div><button type="button" className="pure-button ghost debt-modal-close" aria-label="Закрыть" onClick={()=>setShowInviteForm(false)}><X size={16}/></button></div>{!inviteToken?<><label className="form-label">Email участника<input className="pure-input" type="email" value={inviteEmail} onChange={e=>setInviteEmail(e.target.value)} placeholder="name@example.com"/></label><div className="debt-modal-actions"><button className="pure-button" disabled={!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(inviteEmail.trim())} onClick={()=>void invite()}>Создать приглашение</button><button className="pure-button ghost" onClick={()=>setShowInviteForm(false)}>Отмена</button></div></>:<><div className="empty-note" style={{marginTop:12}}><strong>Приглашение создано</strong><p style={{margin:"8px 0"}}>Передайте этот тикет участнику. Он сможет принять приглашение в разделе «Семья».</p><small>Email: {inviteTicketEmail||"указан при создании"}</small><div style={{marginTop:8,wordBreak:"break-all"}}><strong>{inviteToken}</strong></div></div><div className="debt-modal-actions"><button className="pure-button" onClick={()=>setShowInviteForm(false)}>Готово</button></div></>}</section></div>}
  {showGoalForm&&<div className="debt-modal-backdrop" onMouseDown={e=>{if(e.target===e.currentTarget)resetFamilyGoalForm();}}><section className="debt-modal goal-modal" role="dialog" aria-modal="true" aria-labelledby="family-goal-modal-title"><div className="debt-modal-heading"><div><strong id="family-goal-modal-title">{editingFamilyGoalId===null?"Создать семейную цель":"Редактировать семейную цель"}</strong><small>Настройте параметры накоплений</small></div><button type="button" className="pure-button ghost debt-modal-close" aria-label="Закрыть" onClick={resetFamilyGoalForm}><X size={16}/></button></div>{goalLimitExceeded?<><p role="alert" style={{color:"#f87171",fontSize:14}}>Достигнут лимит: можно создать не более трёх семейных целей.</p><div className="debt-modal-actions"><button className="pure-button" onClick={resetFamilyGoalForm}>Понятно</button></div></>:<><div className="pure-grid two"><label className="form-label">Название<input className="pure-input" style={goalInvalidFields.title?{borderColor:"#ef4444"}:undefined} aria-invalid={!!goalInvalidFields.title} value={goalTitle} onChange={e=>{setGoalTitle(e.target.value);setGoalInvalidFields(old=>({...old,title:false}));}} maxLength={24} placeholder="Например, Путешествие"/></label><label className="form-label">Целевая сумма, ₽<input className="pure-input" style={goalInvalidFields.target?{borderColor:"#ef4444"}:undefined} aria-invalid={!!goalInvalidFields.target} value={goalTarget} onChange={e=>{setGoalTarget(e.target.value);setGoalInvalidFields(old=>({...old,target:false}));}} maxLength={12} placeholder="100000"/><small className="muted">Максимум: 999 999 999 ₽</small></label><label className="form-label">Уже накоплено, ₽<input className="pure-input" style={goalInvalidFields.starting?{borderColor:"#ef4444"}:undefined} aria-invalid={!!goalInvalidFields.starting} value={goalStarting} onChange={e=>{setGoalStarting(e.target.value);setGoalInvalidFields(old=>({...old,starting:false}));}} maxLength={12} placeholder="0"/><small className="muted">Максимум: 999 999 999 ₽</small></label><label className="form-label">Срок<input className="pure-input" type="date" value={goalDueDate} onChange={e=>setGoalDueDate(e.target.value)}/></label><label className="form-label">Приоритет<select className="pure-select" value={goalPriority} onChange={e=>setGoalPriority(e.target.value)}><option value="low">Низкий</option><option value="medium">Средний</option><option value="high">Высокий</option></select></label><label className="form-label">Описание<input className="pure-input" style={goalInvalidFields.description?{borderColor:"#ef4444"}:undefined} aria-invalid={!!goalInvalidFields.description} value={goalDescription} onChange={e=>{setGoalDescription(e.target.value);setGoalInvalidFields(old=>({...old,description:false}));}} maxLength={30}/></label></div><div className="debt-modal-actions"><button className="pure-button" onClick={()=>void saveFamilyGoal()}>{editingFamilyGoalId===null?"Создать цель":"Сохранить изменения"}</button><button className="pure-button ghost" onClick={resetFamilyGoalForm}>Отмена</button></div></>}</section></div>}
  <style>{`
   .family-filter-menu{position:absolute;top:calc(100% + 7px);left:0;z-index:50;display:grid;gap:3px;min-width:190px;max-height:300px;overflow:auto;padding:7px;border:1px solid #293953;border-radius:12px;background:var(--pure-surface,#fff);box-shadow:0 14px 34px rgba(20,43,85,.18)}
   .family-filter-menu button{display:flex;align-items:center;gap:8px;min-height:32px;padding:0 9px;border:0;border-radius:8px;color:var(--pure-ink,#31496e);background:transparent;text-align:left;font-size:11px;white-space:nowrap}
   .family-filter-menu button:hover,.family-filter-menu button.selected{color:#286be9;background:#eaf1ff}
   .family-period-menu{width:230px;padding:12px;gap:10px}
   .family-period-menu .form-row{flex-direction:row}
   .family-period-menu .form-label{gap:5px}
   .family-category-menu{min-width:220px}
   .family-filter-stats .stat-card{padding:10px 13px}
   .family-filter-stats .stat-card strong{margin:5px 0 0;font-size:18px}
   .family-filter-stats .stat-top{font-size:10px}
   .operations-history-list .transaction-row{min-width:0}
   @media(max-width:650px){.family-filter-stats{grid-template-columns:repeat(2,minmax(0,1fr))}.operations-history-list{grid-template-columns:1fr}.operations-history-list::after{display:none}.operations-history-list .transaction-row{grid-template-columns:30px minmax(0,1fr) auto 26px 26px}}
   :root[data-theme="dark"] .family-filter-menu{border-color:#344866;background:#17263d}
   :root[data-theme="dark"] .family-filter-menu button{color:#e7eefc}
   :root[data-theme="dark"] .family-filter-menu button:hover,:root[data-theme="dark"] .family-filter-menu button.selected{color:#8ab5ff;background:#263b5b}
  `}</style>
  {confirmAction&&<ConfirmPopover title={confirmAction.title} message={confirmAction.message} confirmLabel={confirmAction.confirmLabel} onCancel={()=>setConfirmAction(null)} onConfirm={()=>{const action=confirmAction;setConfirmAction(null);void action.onConfirm();}}/>}
  {error&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
 </div>;
}
export default Family;
