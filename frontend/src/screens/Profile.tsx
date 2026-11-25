import { useEffect, useState, type ChangeEvent } from "react";
import { UserRound, Trophy, Star, ShieldCheck, TrendingUp, CalendarDays, Camera, Trash2 } from "lucide-react";
import { PageHeading, Panel, Progress } from "./ui";
import "./Pure.css";

type UserProfile={id:number;firstName:string;lastName:string;username:string;email:string;avatarUrl:string|null};
type Achievement={key:string;title:string;description:string;xpAwarded:number;awardedAt:string};
type Gamification={xp:number;level:number;operations:number;currentStreak:number;achievements:Achievement[]};

function Profile(){
 const [user,setUser]=useState<UserProfile|null>(null);
 const [game,setGame]=useState<Gamification|null>(null);
 const [avatarBusy,setAvatarBusy]=useState(false);
 const [avatarError,setAvatarError]=useState("");
 const loadProfile=async()=>{const r=await fetch("/api/profile");if(!r.ok)throw new Error("Не удалось загрузить профиль");setUser(await r.json());};
 const loadGame=async()=>{const r=await fetch("/api/gamification");if(!r.ok)throw new Error("Не удалось загрузить достижения");setGame(await r.json());};
 useEffect(()=>{void loadProfile().catch(e=>setAvatarError(e.message));void loadGame().catch(e=>setAvatarError(e.message));},[]);
 const handleAvatarChange=async(event:ChangeEvent<HTMLInputElement>)=>{
  const file=event.target.files?.[0];event.target.value="";if(!file)return;setAvatarError("");setAvatarBusy(true);
  try{const form=new FormData();form.append("avatar",file);const r=await fetch("/api/profile/avatar",{method:"PUT",body:form});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось загрузить аватар");setUser(x=>x?{...x,avatarUrl:d.avatarUrl}:x);}
  catch(e){setAvatarError(e instanceof Error?e.message:"Ошибка загрузки аватара");}finally{setAvatarBusy(false);}
 };
 const deleteAvatar=async()=>{setAvatarBusy(true);setAvatarError("");try{const r=await fetch("/api/profile/avatar",{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить аватар");setUser(x=>x?{...x,avatarUrl:null}:x);}catch(e){setAvatarError(e instanceof Error?e.message:"Ошибка удаления");}finally{setAvatarBusy(false);}};
 const name=user?user.firstName+" "+user.lastName:"Загрузка профиля";
 const progress=game?game.xp%100:0;
 return <div className="pure-page" style={{padding:16}}>
  <PageHeading title="Профиль" subtitle="Ваш путь, достижения и активность"/>
  <Panel className="profile-hero">
   <div className="profile-avatar">{user?.avatarUrl?<img src={user.avatarUrl} alt="Аватар" style={{width:"100%",height:"100%",objectFit:"cover",borderRadius:"50%"}}/>:<UserRound/>}</div>
   <div className="profile-name" style={{flex:1}}>
    <h2>{name}</h2><p>@{user?.username??"username"} · {game?"Уровень "+game.level:"Уровень…"}</p>
    <div style={{maxWidth:280,marginTop:13}}><Progress value={progress}/><small className="muted">{progress} / 100 XP до следующего уровня · Всего {game?.xp??0} XP</small></div>
   </div>
   <div style={{display:"flex",flexWrap:"wrap",gap:8}}>
    <label className="pure-button secondary" style={{cursor:avatarBusy?"wait":"pointer",opacity:avatarBusy?0.6:1}}><Camera size={15}/>{avatarBusy?"Загрузка…":"Изменить фото"}<input type="file" accept="image/jpeg,image/png,image/webp,image/gif" hidden disabled={avatarBusy} onChange={handleAvatarChange}/></label>
    {user?.avatarUrl&&<button type="button" className="pure-button secondary" disabled={avatarBusy} onClick={()=>void deleteAvatar()}><Trash2 size={15}/>Удалить фото</button>}
   </div>
  </Panel>
  {avatarError&&<p role="alert" style={{color:"#f87171",fontSize:12,marginTop:8}}>{avatarError}</p>}
  <div className="pure-grid three" style={{marginTop:16}}>
   <div className="stat-card"><span className="muted" style={{fontSize:11}}>Операции</span><strong>{game?.operations??"…"}</strong></div>
   <div className="stat-card"><span className="muted" style={{fontSize:11}}>Достижения</span><strong>{game?.achievements.length??"…"}</strong></div>
   <div className="stat-card"><span className="muted" style={{fontSize:11}}>Дней подряд</span><strong>{game?.currentStreak??"…"}</strong></div>
  </div>
  <div className="pure-grid two" style={{marginTop:16}}>
   <Panel title="Достижения" action={<span className="muted" style={{fontSize:10}}>{game?.achievements.length??0} наград</span>}>
    {game?.achievements.length?<div className="badge-grid">{game.achievements.map((a,index)=>{const Icon=index===0?Trophy:index%2?Star:ShieldCheck;return <div className="badge-item" key={a.key}><Icon/><strong>{a.title}</strong><small>{a.description}</small><small>+{a.xpAwarded} XP</small></div>;})}</div>:<div className="empty-note">Достижения появятся по мере вашей финансовой активности.</div>}
   </Panel>
   <Panel title="Уровень и опыт" subtitle="Опыт начисляется за реальные действия">
    <div style={{display:"flex",alignItems:"center",gap:14}}><div className="transaction-icon mint"><TrendingUp size={20}/></div><div><strong style={{fontSize:26}}>Уровень {game?.level??1}</strong><div className="muted">{game?.xp??0} XP всего</div></div></div>
    <div style={{marginTop:18}}><div className="legend-row"><span>Прогресс уровня</span><strong>{progress}%</strong></div><Progress value={progress}/></div>
    <div className="empty-note" style={{marginTop:16}}>Следующий уровень открывается каждые 100 XP.</div>
   </Panel>
  </div>
  <Panel title="Активность" subtitle="Последовательные дни финансовых действий">
   <div className="member-row"><div className="transaction-icon"><CalendarDays size={16}/></div><div className="member-copy"><strong>{game?.currentStreak??0} дней подряд</strong><small>Активность учитывает операции, семейные записи и движения по целям.</small></div></div>
  </Panel>
 </div>;
}
export default Profile;
