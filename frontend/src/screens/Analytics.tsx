import { useEffect, useMemo, useState } from "react";
import { ArrowDownLeft, ArrowUpRight, Wallet, TrendingUp } from "lucide-react";
import { PageHeading, Panel, Stat, Donut, LineChart } from "./ui";
import "./Pure.css";

type Period = "week" | "month" | "year" | "custom";
type Category = { name: string; amount: string; count: number };
type Point = { date: string; income: string; expense: string };
type Large = { title: string; amount: string; date: string };
type BalancePoint = { date:string; balance:string };
type CategoryBudget = {id:number;category:string;monthlyLimit:string;spent:string;remaining:string};
const toCents=(value:string)=>{const [whole,fraction=""]=value.split(".");return BigInt(whole||"0")*100n+BigInt((fraction+"00").slice(0,2));};
type AnalyticsData = {
  period: string; from: string; to: string; income: string; expense: string; net: string;
  averageExpense: string; categories: Category[]; trend: Point[]; largest: Large[];
  previousIncome: string; previousExpense: string; previousNet: string;
  balanceTrend: BalancePoint[];
  insights: string[];
};
const formatAmount = (value: string) => {
  const negative = value.startsWith("-");
  const [whole, fraction = ""] = value.replace("-", "").split(".");
  const cents = BigInt(whole || "0") * 100n + BigInt((fraction + "00").slice(0, 2));
  const abs = cents < 0n ? -cents : cents;
  return (negative || cents < 0n ? "−" : "") + new Intl.NumberFormat("ru-RU").format(abs / 100n) +
    "," + (abs % 100n).toString().padStart(2, "0") + " ₽";
};
const localDate = (date: Date) => [date.getFullYear(), String(date.getMonth()+1).padStart(2,"0"), String(date.getDate()).padStart(2,"0")].join("-");
const palette = ["#3478f6","#78c9ee","#8e7cf4","#ffb56e","#08a47f","#ef8b8b"];

function Analytics() {
  const [period, setPeriod] = useState<Period>("month");
  const [from, setFrom] = useState(localDate(new Date(new Date().getFullYear(), new Date().getMonth(), 1)));
  const [to, setTo] = useState(localDate(new Date()));
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [budgets,setBudgets]=useState<CategoryBudget[]>([]);
  const [budgetCategory,setBudgetCategory]=useState("");
  const [budgetAmount,setBudgetAmount]=useState("");
  const [budgetError,setBudgetError]=useState("");
  const [budgetBusy,setBudgetBusy]=useState(false);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      setLoading(true); setError("");
      try {
        const params = new URLSearchParams({ period });
        if (period === "custom") { params.set("from", from); params.set("to", to); }
        const response = await fetch(`/api/analytics?${params.toString()}`);
        const result = await response.json();
        if (!response.ok) throw new Error(result.error || "Не удалось загрузить аналитику");
        if (!cancelled) setData(result);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : "Ошибка загрузки аналитики");
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    void load();
    return () => { cancelled = true; };
  }, [period, from, to]);

  const loadBudgets=async()=>{try{const r=await fetch("/api/budgets");const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось загрузить бюджеты");setBudgets(d);}catch(e){setBudgetError(e instanceof Error?e.message:"Ошибка бюджетов");}};
  useEffect(()=>{void loadBudgets();},[]);
  const saveBudget=async()=>{
    setBudgetBusy(true);setBudgetError("");
    try{const r=await fetch("/api/budgets",{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify({category:budgetCategory,amount:budgetAmount.replace(",","." )})});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось сохранить бюджет");setBudgetCategory("");setBudgetAmount("");await loadBudgets();}
    catch(e){setBudgetError(e instanceof Error?e.message:"Ошибка сохранения бюджета");}
    finally{setBudgetBusy(false);}
  };
  const removeBudget=async(id:number)=>{try{const r=await fetch(`/api/budgets/${id}`,{method:"DELETE"});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось удалить бюджет");setBudgets(old=>old.filter(x=>x.id!==id));}catch(e){setBudgetError(e instanceof Error?e.message:"Ошибка удаления бюджета");}};
  const totalExpenses = useMemo(() => data?.categories.reduce((sum, item) => sum + BigInt(item.amount.split(".")[0] || "0") * 100n + BigInt((item.amount.split(".")[1] || "").padEnd(2,"0").slice(0,2)), 0n) ?? 0n, [data]);
  const categoryShares = useMemo(() => {
    if (!data?.categories.length || totalExpenses === 0n) return [];
    return data.categories.map(item => Number((BigInt(item.amount.split(".")[0] || "0") * 100n + BigInt((item.amount.split(".")[1] || "").padEnd(2,"0").slice(0,2))) * 10000n / totalExpenses) / 100);
  }, [data, totalExpenses]);
  const trendValues = useMemo(() => {
    const values = data?.trend.map(item => Number(item.expense)) ?? [];
    const max = Math.max(1, ...values);
    return values.map(value => Math.max(4, Math.round(value / max * 100)));
  }, [data]);
  const trendLabels = useMemo(() => data?.trend.map(item => {
    const date = new Date(item.date + "T00:00:00");
    return period === "year" ? date.toLocaleDateString("ru-RU",{month:"short"}) : date.toLocaleDateString("ru-RU",{day:"2-digit",month:"2-digit"});
  }) ?? [], [data, period]);
  const balanceValues = useMemo(() => {
    const values=data?.balanceTrend.map(item=>Number(item.balance))??[];
    if(!values.length)return [];
    const min=Math.min(...values),max=Math.max(...values);
    return values.map(value=>max===min?50:Math.max(4,Math.min(100,4+(value-min)/(max-min)*92)));
  },[data]);
  const balanceLabels = useMemo(() => data?.balanceTrend.map(item => {
    const date=new Date(item.date+"T00:00:00");
    return period==="year"?date.toLocaleDateString("ru-RU",{month:"short"}):date.toLocaleDateString("ru-RU",{day:"2-digit",month:"2-digit"});
  })??[],[data,period]);

  return <div className="pure-page">
    <PageHeading title="Аналитика" subtitle="Финансовые показатели из ваших операций"/>
    <div className="filter-pills" style={{marginBottom:16}}>
      {([["week","Неделя"],["month","Месяц"],["year","Год"],["custom","Период"]] as [Period,string][]).map(([key,label]) =>
        <button key={key} className={`filter-pill ${period===key?"active":""}`} onClick={() => setPeriod(key)}>{label}</button>)}
    </div>
    {period === "custom" && <div className="form-row" style={{marginBottom:16}}>
      <label className="form-label">С даты<input className="pure-input" type="date" value={from} max={to} onChange={e=>setFrom(e.target.value)}/></label>
      <label className="form-label">По дату<input className="pure-input" type="date" value={to} min={from} onChange={e=>setTo(e.target.value)}/></label>
    </div>}
    {loading && <div className="empty-note">Загружаем аналитику…</div>}
    {error && <p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
    <Panel title="Бюджеты категорий" subtitle="Месячные лимиты расходов по выбранным категориям">
      <div className="pure-grid three" style={{marginBottom:14}}>
        <label className="form-label">Категория<input className="pure-input" value={budgetCategory} onChange={e=>setBudgetCategory(e.target.value)} maxLength={80} placeholder="Например, Продукты"/></label>
        <label className="form-label">Лимит на месяц, ₽<input className="pure-input" value={budgetAmount} onChange={e=>setBudgetAmount(e.target.value)} placeholder="0.00"/></label>
        <div className="form-row" style={{alignItems:"end"}}><button className="pure-button" disabled={budgetBusy||!budgetCategory.trim()||!/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(budgetAmount)||toCents(budgetAmount.replace(",","."))<=0n} onClick={()=>void saveBudget()}>{budgetBusy?"Сохранение…":"Сохранить лимит"}</button></div>
      </div>
      {budgets.length===0?<div className="empty-note">Бюджеты не заданы. Добавьте лимит для категории, чтобы следить за расходами месяца.</div>:<div className="transaction-list">{budgets.map(b=>{const spent=toCents(b.spent),limit=toCents(b.monthlyLimit);const percent=limit>0n?Number(spent*10000n/limit)/100:0;const width=Math.max(0,Math.min(100,percent));return <div className="transaction-row" key={b.id} style={{gridTemplateColumns:"minmax(0,1fr) auto"}}>
        <span className="transaction-copy"><strong>{b.category}</strong><small>{formatAmount(b.spent)} из {formatAmount(b.monthlyLimit)} · {percent.toFixed(1).replace(".",",")}%</small><span className="progress-track" style={{display:"block",marginTop:7}}><span className="progress-fill" style={{width:width+"%",background:percent>100?"#ef6b78":undefined}}/></span><small className={spent>limit?"negative":"muted"}>{spent>limit?"Превышение: "+formatAmount((spent-limit).toString()):"Осталось: "+formatAmount(b.remaining)}</small></span>
        <button className="pure-button ghost" aria-label="Удалить бюджет" onClick={()=>void removeBudget(b.id)}>Удалить</button>
      </div>})}</div>}
      {budgetError&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{budgetError}</p>}
    </Panel>
    {data && <>
      <div className="pure-grid three">
        <Stat label="Расходы" value={formatAmount(data.expense)} tone="violet" icon={<ArrowUpRight size={17}/>}/>
        <Stat label="Доходы" value={formatAmount(data.income)} tone="mint" icon={<ArrowDownLeft size={17}/>}/>
        <Stat label="Разница" value={formatAmount(data.net)} icon={<Wallet size={17}/>}/>
      </div>
      <div className="pure-grid two" style={{marginTop:16}}>
        <Panel title="Динамика расходов" subtitle={`Период: ${data.from} – ${data.to}`}>
          {data.trend.length ? <LineChart values={trendValues} labels={trendLabels}/> : <div className="empty-note">За этот период операций пока нет.</div>}
        </Panel>
        <Panel title="Структура расходов" subtitle="Распределение по категориям">
          {data.categories.length ? <div style={{display:"flex",alignItems:"center",justifyContent:"space-around",gap:14,flexWrap:"wrap"}}>
            <Donut segments={categoryShares} center="100%" caption="расходов"/>
            <div className="chart-legend">{data.categories.map((item,index)=><div className="legend-row" key={item.name}>
              <i className="legend-dot" style={{background:palette[index%palette.length]}}/>
              <span>{item.name}</span><strong>{categoryShares[index].toFixed(1).replace(".",",")}%</strong>
            </div>)}</div>
          </div> : <div className="empty-note">Нет расходов по категориям за выбранный период.</div>}
        </Panel>
      </div>
      <Panel title="Динамика общего баланса" subtitle="Баланс личных счетов с учётом доходов и расходов, без внутренних переводов">
        {balanceValues.length?<LineChart values={balanceValues} labels={balanceLabels}/>:<div className="empty-note">Нет данных для динамики баланса.</div>}
      </Panel>
      {data.insights.length>0&&<Panel title="Финансовые подсказки" subtitle="Короткие выводы по выбранному периоду">
        <ul className="insight-list">{data.insights.map((insight,index)=><li key={index}>{insight}</li>)}</ul>
      </Panel>}
      <div className="pure-grid two" style={{marginTop:16}}>
        <Panel title="Крупнейшие расходы" subtitle="Пять самых больших операций">
          {data.largest.length ? <div className="transaction-list">{data.largest.map((item,index)=><div className="transaction-row" key={index}>
            <span className="transaction-copy"><strong>{item.title}</strong><small>{item.date}</small></span>
            <strong className="transaction-amount negative">−{formatAmount(item.amount)}</strong>
          </div>)}</div> : <div className="empty-note">Пока нет расходов за выбранный период.</div>}
        </Panel>
        <Panel title="Сводка периода" subtitle="Показатели по операциям">
          <div className="legend-row"><span>Средний расход</span><strong>{formatAmount(data.averageExpense)}</strong></div>
          <div className="legend-row" style={{marginTop:12}}><span>Категорий с расходами</span><strong>{data.categories.length}</strong></div>
          <div className="legend-row" style={{marginTop:12}}><span>Операций в динамике</span><strong>{data.trend.length}</strong></div>
          <div className="empty-note" style={{display:"flex",gap:12,alignItems:"center",marginTop:16}}><TrendingUp color="#08a47f" size={23}/><span>Переводы между собственными счетами не включаются в доходы и расходы.</span></div>
          <div className="legend-row"><span>Доходы в прошлом периоде</span><strong>{formatAmount(data.previousIncome)}</strong></div>
          <div className="legend-row" style={{marginTop:12}}><span>Расходы в прошлом периоде</span><strong>{formatAmount(data.previousExpense)}</strong></div>
          <div className="legend-row" style={{marginTop:12}}><span>Разница в прошлом периоде</span><strong>{formatAmount(data.previousNet)}</strong></div>
        </Panel>
      </div>
    </>}
  </div>;
}
export default Analytics;
