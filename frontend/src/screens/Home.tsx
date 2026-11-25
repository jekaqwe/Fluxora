import { useEffect, useMemo, useState } from "react";
import { ArrowDownLeft, ArrowUpRight, ArrowLeftRight, Coffee, ShoppingBag, Car, Wallet } from "lucide-react";
import { PageHeading, Panel, Stat, Donut, LineChart } from "./ui";
import "./Pure.css";
import QuickEntry from "./QuickEntry";

type Tx = { id:number; title:string; type:"income"|"expense"|"transfer"; amount:string; category:string; date:string; accountId:number; transferAccountId?:number };
type Dashboard = { totalBalance:string; monthIncome:string; monthExpense:string; monthNet:string; recent:Tx[] };
type Category = { name:string; amount:string; count:number };
type Point = { date:string; income:string; expense:string };
type Analytics = { categories:Category[]; trend:Point[] };
type Goal = { id:number; title:string; currentAmount:string; targetAmount:string; status:"active"|"paused"|"completed" };
const money = (value:string) => {
  const negative=value.startsWith("-");
  const [whole,fraction=""]=value.replace("-","").split(".");
  const cents=BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));
  return (negative?"−":"")+new Intl.NumberFormat("ru-RU").format(cents/100n)+","+(cents%100n).toString().padStart(2,"0")+" ₽";
};
const colors=["#3478f6","#78c9ee","#8e7cf4","#ffb56e","#08a47f","#ef8b8b"];
const toCents=(value:string)=>{const [whole,fraction=""]=value.split(".");return BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));};

function Home() {
  const [dashboard,setDashboard]=useState<Dashboard|null>(null);
  const [analytics,setAnalytics]=useState<Analytics|null>(null);
  const [activeGoals,setActiveGoals]=useState<Goal[]>([]);
  const [error,setError]=useState("");
  const [refreshKey,setRefreshKey]=useState(0);
  useEffect(()=>{
    let cancelled=false;
    Promise.all([fetch("/api/dashboard"),fetch("/api/analytics?period=month"),fetch("/api/goals")])
      .then(async([a,b,c])=>{
        const [summary,stats,goals]=await Promise.all([a.json(),b.json(),c.json()]);
        if(!a.ok||!b.ok||!c.ok)throw new Error(summary.error||stats.error||goals.error||"Не удалось загрузить главную страницу");
        if(!cancelled){setDashboard(summary);setAnalytics(stats);setActiveGoals((goals as Goal[]).filter(goal=>goal.status==="active").slice(0,3));}
      })
      .catch(e=>{if(!cancelled)setError(e instanceof Error?e.message:"Ошибка загрузки");});
    return()=>{cancelled=true;};
  },[refreshKey]);
  const categoryTotal=useMemo(()=>analytics?.categories.reduce((sum,item)=>sum+BigInt(item.amount.split(".")[0]||"0")*100n+BigInt((item.amount.split(".")[1]||"").padEnd(2,"0").slice(0,2)),0n)??0n,[analytics]);
  const shares=useMemo(()=>analytics?.categories.map(item=>{
    const amount=BigInt(item.amount.split(".")[0]||"0")*100n+BigInt((item.amount.split(".")[1]||"").padEnd(2,"0").slice(0,2));
    return categoryTotal===0n?0:Number(amount*10000n/categoryTotal)/100;
  })??[],[analytics,categoryTotal]);
  const trend=analytics?.trend??[];
  const trendValues=trend.map(item=>Number(item.expense));
  
  const labels=trend.map(item=>new Date(item.date+"T00:00:00").toLocaleDateString("ru-RU",{day:"2-digit",month:"2-digit"}));
  return <div className="pure-page" style={{padding:16}}>
    <PageHeading title="Главная" subtitle="Ваши финансы. Ваши цели. Вместе с Fluxora."/>
    {error&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
    {!dashboard?<div className="empty-note">Загружаем финансовую сводку…</div>:<>
      <QuickEntry onSaved={()=>setRefreshKey(value=>value+1)}/>
      <div className="pure-grid four" style={{marginTop: 16}}>
        <Stat label="Общий баланс" value={money(dashboard.totalBalance)} icon={<Wallet size={17}/>}/>
        <Stat label="Доходы за месяц" value={money(dashboard.monthIncome)} tone="mint" icon={<ArrowDownLeft size={17}/>}/>
        <Stat label="Расходы за месяц" value={money(dashboard.monthExpense)} tone="violet" icon={<ArrowUpRight size={17}/>}/>
        <Stat label="Разница за месяц" value={money(dashboard.monthNet)} tone="blue" icon={<Wallet size={17}/>}/>
      </div>
      {activeGoals.length>0&&<Panel title="Активная цель" subtitle="Ваши текущие планы накоплений">
        <div className="pure-grid three">
          {activeGoals.map(goal=><div key={goal.id} style={{minWidth:0,padding:12,borderRadius:12,background:"var(--surface-soft, rgba(148,163,184,.08))"}}>
            <strong style={{display:"block",overflow:"hidden",textOverflow:"ellipsis",whiteSpace:"nowrap"}}>{goal.title}</strong>
            <div className="legend-row" style={{marginTop:10,gap:6,flexWrap:"wrap"}}><span>{money(goal.currentAmount)}</span><strong>{money(goal.targetAmount)}</strong></div>
            <div className="progress-track" style={{marginTop:10}}><div className="progress-fill" style={{width:Math.min(100,Number(toCents(goal.currentAmount)*10000n/toCents(goal.targetAmount))/100)+"%"}}/></div>
          </div>)}
        </div>
      </Panel>}
      <div className="pure-grid two" style={{marginTop:16}}>
        <Panel title="Динамика расходов" subtitle="По дням текущего месяца">
          {trend.length?<LineChart values={trendValues} labels={labels} valueLabels={trend.map(item=>money(item.expense))}/>:<div className="empty-note">За этот месяц операций пока нет.</div>}
        </Panel>
        <Panel title="Расходы по категориям" subtitle="Распределение за текущий месяц">
          {analytics?.categories.length?<div style={{display:"flex",alignItems:"center",justifyContent:"space-around",gap:16,flexWrap:"wrap"}}>
            <Donut segments={shares} center="100%" caption="расходов" labels={analytics.categories.map(item=>item.name)} values={analytics.categories.map(item=>money(item.amount))}/>
            <div className="chart-legend">{analytics.categories.map((item,index)=><div className="category-legend-row" key={item.name}><i className="legend-dot" style={{background:colors[index%colors.length]}}/><span className="category-legend-name">{item.name}</span><strong className="category-legend-value">{money(item.amount)} · {shares[index].toFixed(1).replace(".",",")}%</strong></div>)}</div>
          </div>:<div className="empty-note">Нет расходов по категориям за этот месяц.</div>}
        </Panel>
      </div>
      <div className="pure-grid two" style={{marginTop:16}}>
        <Panel title="Последние операции" subtitle="Пять последних записей">
          {dashboard.recent.length?<div className="transaction-list">{dashboard.recent.map(item=>{
            const Icon=item.type==="income"?ArrowDownLeft:item.type==="transfer"?ArrowLeftRight:item.category==="Транспорт"?Car:item.category==="Кафе"?Coffee:ShoppingBag;
            const tone=item.type==="income"?"mint":item.type==="transfer"?"violet":"";
            return <div className="transaction-row" key={item.id}>
              <span className={`transaction-icon ${tone}`}><Icon size={16}/></span>
              <span className="transaction-copy"><strong>{item.title}</strong><small>{item.category} · {item.date}</small></span>
              <strong className={`transaction-amount ${item.type==="income"?"positive":item.type==="expense"?"negative":""}`}>{item.type==="income"?"+":item.type==="expense"?"−":"↔ "}{money(item.amount)}</strong>
            </div>;
          })}</div>:<div className="empty-note">Добавьте первую операцию, и она появится здесь.</div>}
        </Panel>
        <Panel title="Финансовый итог" subtitle="Текущий месяц">
          <div className="legend-row"><span>Доходы</span><strong className="positive">{money(dashboard.monthIncome)}</strong></div>
          <div className="legend-row" style={{marginTop:12}}><span>Расходы</span><strong className="negative">{money(dashboard.monthExpense)}</strong></div>
          <div className="legend-row" style={{marginTop:12}}><span>Чистый результат</span><strong>{money(dashboard.monthNet)}</strong></div>
          <div className="empty-note" style={{marginTop:16}}>Переводы между вашими счетами не увеличивают доходы и расходы.</div>
        </Panel>
      </div>
    </>}
  </div>;
}
export default Home;
