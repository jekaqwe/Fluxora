import { useEffect, useState } from "react";
import { UserRound, ShieldCheck, Bell, Palette, Eye, Link2, LockKeyhole } from "lucide-react";
import { PageHeading, Panel } from "./ui";
import "./Pure.css";
type ProfileField = "firstName" | "lastName" | "username";
type UserProfile = {
  firstName: string;
  lastName: string;
  username: string;
  email: string;
  firstNameAvailableAt: string | null;
  lastNameAvailableAt: string | null;
  usernameAvailableAt: string | null;
};
type ProfileDraft = Pick<UserProfile, ProfileField>;
const sections=[{id:"account",label:"Аккаунт",Icon:UserRound},{id:"security",label:"Безопасность",Icon:ShieldCheck},{id:"notifications",label:"Уведомления",Icon:Bell},{id:"appearance",label:"Оформление",Icon:Palette},{id:"privacy",label:"Приватность",Icon:Eye},{id:"integrations",label:"Интеграции",Icon:Link2}];
function Settings(){
 const [section,setSection]=useState("account");const [user,setUser]=useState<UserProfile|null>(null);
 const [draft,setDraft]=useState<ProfileDraft>({firstName:"",lastName:"",username:""});
 const [savingField,setSavingField]=useState<ProfileField|null>(null);
 const [profileMessage,setProfileMessage]=useState("");
 const [profileError,setProfileError]=useState("");
 const [prefs,setPrefs]=useState(()=>{try{return JSON.parse(localStorage.getItem("fluxora_preferences")||"{}")}catch{return {}}});
 const [theme,setTheme]=useState(localStorage.getItem("fluxora_theme")||"light");
 useEffect(()=>{fetch("/api/profile").then(r=>{if(!r.ok)throw new Error("Не удалось загрузить аккаунт");return r.json();}).then((data:UserProfile)=>{setUser(data);setDraft({firstName:data.firstName,lastName:data.lastName,username:data.username});}).catch(e=>console.error(e));},[]);
 const saveProfileField=async(field:ProfileField)=>{
   setProfileError("");setProfileMessage("");setSavingField(field);
   try{
     const response=await fetch("/api/profile",{method:"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({[field]:draft[field]})});
     const data=await response.json();
     if(!response.ok)throw new Error(data.error||"Не удалось сохранить изменения");
     setUser(data);setDraft({firstName:data.firstName,lastName:data.lastName,username:data.username});
     setProfileMessage("Изменения сохранены");
   }catch(error){setProfileError(error instanceof Error?error.message:"Ошибка сохранения");}
   finally{setSavingField(null);}
 };
 const cooldownText=(date:string|null)=>{
   if(!date)return "Изменение доступно";
   const remaining=new Date(date).getTime()-Date.now();
   if(remaining<=0)return "Изменение доступно";
   return "Можно изменить: "+new Date(date).toLocaleString("ru-RU",{dateStyle:"short",timeStyle:"short"});
 };
 const isCooling=(date:string|null)=>!!date&&new Date(date).getTime()>Date.now();
 const toggle=(key:string)=>setPrefs((old:Record<string,boolean>)=>{const next={...old,[key]:!(old[key]??true)};localStorage.setItem("fluxora_preferences",JSON.stringify(next));return next;});
 const isOn=(key:string)=>prefs[key]??true;
 return <div className="pure-page"><PageHeading title="Настройки" subtitle="Управляйте аккаунтом и параметрами Fluxora"/>
 <div className="pure-grid two" style={{gridTemplateColumns:"minmax(170px,.65fr) minmax(0,2fr)"}}>
 <Panel title="Разделы настроек"><nav className="setting-nav">{sections.map(({id,label,Icon})=><button key={id} className={section===id?"active":""} onClick={()=>setSection(id)}><Icon size={15}/>{label}</button>)}</nav></Panel>
 <Panel title={sections.find(s=>s.id===section)?.label||"Настройки"} subtitle="Параметры вашего аккаунта и приложения">
 {section==="account"&&<><div className="pure-grid two">
   {(["firstName","lastName","username"] as ProfileField[]).map((field)=>{
     const labels:Record<ProfileField,string>={firstName:"Имя",lastName:"Фамилия",username:"Логин"};
     const availableAt=field==="firstName"?user?.firstNameAvailableAt:field==="lastName"?user?.lastNameAvailableAt:user?.usernameAvailableAt;
     return <div className="setting-field" key={field}>
       {labels[field]}
       <input value={draft[field]} placeholder="Загрузка…" onChange={e=>setDraft(old=>({...old,[field]:e.target.value}))} disabled={!user||savingField!==null}/>
       <small className="muted">{cooldownText(availableAt||null)}</small>
       <button type="button" className="pure-button secondary" style={{marginTop:8,alignSelf:"flex-start"}} disabled={!user||savingField!==null||isCooling(availableAt||null)||draft[field].trim()===""||draft[field]===user[field]} onClick={()=>saveProfileField(field)}>{savingField===field?"Сохранение…":"Сохранить "+labels[field].toLowerCase()}</button>
     </div>;
   })}
   <label className="setting-field">Email<input value={user?.email||""} readOnly placeholder="Загрузка…"/><small className="muted">Изменение email будет добавлено отдельным механизмом подтверждения адреса.</small></label>
 </div>
 {profileError&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{profileError}</p>}
 {profileMessage&&<p role="status" style={{color:"#86efac",fontSize:12}}>{profileMessage}</p>}
 <div className="empty-note">Имя, фамилия и логин изменяются независимо друг от друга. Для каждого поля действует отдельное ограничение: один раз в 14 дней.</div></>}
 {section==="security"&&<><div className="toggle-row"><span><strong>Пароль</strong><br/><small className="muted">Защитите аккаунт надёжным паролем</small></span><LockKeyhole size={18}/></div><label className="setting-field">Текущий пароль<input type="password" placeholder="Введите текущий пароль" disabled/></label><label className="setting-field">Новый пароль<input type="password" placeholder="Не менее 8 символов" disabled/></label><button className="pure-button" disabled title="Нужен серверный endpoint смены пароля">Изменить пароль</button><p className="muted" style={{fontSize:10}}>Смена пароля и управление активными сессиями пока не поддерживаются серверным API.</p></>}
 {section==="notifications"&&<>{[["operations","Уведомления об операциях","Сообщать о новых доходах и расходах"],["goals","Прогресс целей","Напоминать о важных этапах накоплений"],["family","Семейные события","Получать обновления общего бюджета"]].map(([k,t,d])=><div className="toggle-row" key={k}><span><strong>{t}</strong><br/><small className="muted">{d}</small></span><button type="button" aria-label={t} aria-pressed={isOn(k)} className={`toggle ${isOn(k)?"on":""}`} onClick={()=>toggle(k)}/></div>)}</>}
 {section==="appearance"&&<><div className="toggle-row"><span><strong>Тема оформления</strong><br/><small className="muted">Выберите внешний вид интерфейса</small></span></div><div className="filter-pills">{["light","dark","auto"].map(t=><button key={t} className={`filter-pill ${theme===t?"active":""}`} onClick={()=>{setTheme(t);localStorage.setItem("fluxora_theme",t);}}>{t==="light"?"Светлая":t==="dark"?"Тёмная":"Авто"}</button>)}</div><p className="muted" style={{fontSize:10,marginTop:12}}>Выбор темы сохраняется в этом браузере. Полная тёмная тема будет добавлена отдельно.</p></>}
 {section==="privacy"&&<>{[["profile","Публичный профиль","Разрешить другим видеть имя и достижения"],["achievements","Достижения","Показывать полученные награды"],["activity","Активность","Показывать события в ленте профиля"]].map(([k,t,d])=><div className="toggle-row" key={k}><span><strong>{t}</strong><br/><small className="muted">{d}</small></span><button type="button" aria-label={t} aria-pressed={isOn(k)} className={`toggle ${isOn(k)?"on":""}`} onClick={()=>toggle(k)}/></div>)}<div className="empty-note" style={{marginTop:14}}>Финансовые суммы и история операций не публикуются в профиле.</div></>}
 {section==="integrations"&&<div className="empty-note">Подключённых интеграций пока нет. Раздел подготовлен для будущего подключения банковских и внешних сервисов.</div>}
 </Panel></div></div>;
}
export default Settings;
