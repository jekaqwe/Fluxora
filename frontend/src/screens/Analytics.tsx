import { useState } from "react";
import { ArrowDownLeft, ArrowUpRight, Wallet, TrendingUp } from "lucide-react";
import { PageHeading, Panel, Stat, Donut, LineChart, BarChart } from "./ui";
import "./Pure.css";
function Analytics(){
 const [period,setPeriod]=useState("Месяц");
 const values=period==="Неделя"?[35,42,38,60,55,74,68]:period==="Год"?[30,40,37,55,49,65,82]:[35,48,42,66,58,78,72];
 return <div className="pure-page"><PageHeading title="Аналитика" subtitle="Понимайте свои финансовые привычки"/>
 <div className="pure-grid three"><Stat label="Расходы" value="75 320 ₽" change="+8,2%" tone="violet" icon={<ArrowUpRight size={17}/>}/><Stat label="Доходы" value="120 000 ₽" change="+12,5%" tone="mint" icon={<ArrowDownLeft size={17}/>}/><Stat label="Баланс" value="44 680 ₽" change="+6,1%" icon={<Wallet size={17}/>}/></div>
 <div className="pure-grid two" style={{marginTop:16}}><Panel title="Динамика расходов" subtitle={`Период: ${period}`} action={<div className="filter-pills">{["Неделя","Месяц","Год"].map(p=><button className={`filter-pill ${period===p?"active":""}`} onClick={()=>setPeriod(p)} key={p}>{p}</button>)}</div>}><LineChart values={values} labels={period==="Неделя"?["Пн","Вт","Ср","Чт","Пт","Сб","Вс"]:period==="Год"?["Янв","Мар","Май","Июл","Сен","Ноя","Дек"]:["Янв","Фев","Мар","Апр","Май","Июн","Июл"]}/></Panel>
 <Panel title="Структура расходов" subtitle="Доли категорий"><div style={{display:"flex",alignItems:"center",justifyContent:"space-around",gap:14,flexWrap:"wrap"}}><Donut segments={[31,25,20,24]} center="100%" caption="расходов"/><div className="chart-legend">{[["Продукты","31%","#3478f6"],["Транспорт","25%","#78c9ee"],["Развлечения","20%","#8e7cf4"],["Другое","24%","#ffb56e"]].map(([a,b,c])=><div className="legend-row" key={a}><i className="legend-dot" style={{background:c}}/><span>{a}</span><strong>{b}</strong></div>)}</div></div></Panel></div>
 <div className="pure-grid two" style={{marginTop:16}}><Panel title="Сравнение периодов" subtitle="Расходы по неделям"><BarChart values={[42,58,47,75,64,88,70]}/></Panel><Panel title="Финансовые наблюдения"><div className="empty-note" style={{display:"flex",gap:12,alignItems:"center"}}><TrendingUp color="#08a47f" size={23}/><span>В демонстрационном примере расходы на 8% ниже предыдущего месяца. Продолжайте следить за регулярными категориями.</span></div><div style={{marginTop:15}} className="legend-row"><span>Самая крупная категория</span><strong>Продукты · 31%</strong></div><div className="legend-row" style={{marginTop:12}}><span>Изменение накоплений</span><strong className="positive">+12,5%</strong></div></Panel></div>
 </div>;
}
export default Analytics;
