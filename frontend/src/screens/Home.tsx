import { ArrowDownLeft, ArrowUpRight, Coffee, ShoppingBag, Plane, Wallet } from "lucide-react";
import { PageHeading, Panel, Stat, Donut, LineChart, Progress, money } from "./ui";
import "./Pure.css";
const transactions=[{name:"Пятёрочка",category:"Продукты",amount:-1250,date:"Сегодня",icon:ShoppingBag,tone:"coral"},{name:"Зарплата",category:"Доход",amount:85000,date:"Вчера",icon:ArrowDownLeft,tone:"mint"},{name:"Кафе",category:"Рестораны",amount:-720,date:"14 мая",icon:Coffee,tone:"orange"}];
function Home(){
 return <div className="pure-page"><PageHeading title="Главная" subtitle="Ваши финансы. Ваши цели. Вместе с Fluxora."/>
  <div className="pure-grid four">
   <Stat label="Общий баланс" value="125 750 ₽" change="+12,5%" icon={<Wallet size={17}/>}/>
   <Stat label="Доходы за месяц" value="120 000 ₽" change="+8,2%" tone="mint" icon={<ArrowDownLeft size={17}/>}/>
   <Stat label="Расходы за месяц" value="75 320 ₽" change="−3,4%" tone="violet" icon={<ArrowUpRight size={17}/>}/>
   <Stat label="Остаток бюджета" value="44 680 ₽" change="+6,1%" tone="blue" icon={<Wallet size={17}/>}/>
  </div>
  <div className="pure-grid two" style={{marginTop:16}}>
   <Panel title="Динамика доходов и расходов" subtitle="Сравнение по месяцам"><LineChart values={[35,48,42,66,58,78,72]}/></Panel>
   <Panel title="Расходы по категориям" subtitle="Распределение за текущий месяц"><div style={{display:"flex",alignItems:"center",justifyContent:"space-around",gap:16,flexWrap:"wrap"}}><Donut segments={[31,25,20,24]} center="31%" caption="Продукты"/><div className="chart-legend">{[["Продукты","31%","#3478f6"],["Транспорт","25%","#78c9ee"],["Развлечения","20%","#8e7cf4"],["Другое","24%","#ffb56e"]].map(([a,b,c])=><div className="legend-row" key={a}><i className="legend-dot" style={{background:c}}/><span>{a}</span><strong>{b}</strong></div>)}</div></div></Panel>
  </div>
  <div className="pure-grid two" style={{marginTop:16}}>
   <Panel title="Последние операции" action={<span className="muted" style={{fontSize:10}}>3 операции</span>}><div className="transaction-list">{transactions.map((t)=>{const Icon=t.icon;return <div className="transaction-row" key={t.name}><span className={`transaction-icon ${t.tone}`}><Icon size={16}/></span><span className="transaction-copy"><strong>{t.name}</strong><small>{t.category}</small></span><strong className={`transaction-amount ${t.amount>0?"positive":"negative"}`}>{t.amount>0?"+":""}{money(t.amount)}</strong><span className="transaction-date">{t.date}</span></div>})}</div></Panel>
   <Panel title="Цель на отпуск" subtitle="Продолжайте двигаться к мечте"><div style={{display:"flex",alignItems:"center",gap:18}}><div className="goal-art" style={{width:105,height:90,borderRadius:12,flexShrink:0}}><Plane size={35}/></div><div style={{flex:1}}><strong style={{fontSize:24}}>62%</strong><p className="muted" style={{fontSize:10,margin:"4px 0 12px"}}>62 000 ₽ из 100 000 ₽</p><Progress value={62}/></div></div></Panel>
  </div>
 </div>;
}
export default Home;
