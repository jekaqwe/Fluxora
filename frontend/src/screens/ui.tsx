import { useState, type ReactNode } from "react";
import { ArrowUpRight, TrendingUp } from "lucide-react";

export function PageHeading({ title, subtitle, action }: { title:string; subtitle?:string; action?:ReactNode }) {
  return <div className="page-heading"><div><h1>{title}</h1>{subtitle && <p>{subtitle}</p>}</div>{action && <div className="page-heading-action">{action}</div>}</div>;
}
export function Panel({ title, subtitle, action, children, className="" }: { title?:string; subtitle?:string; action?:ReactNode; children:ReactNode; className?:string }) {
  return <section className={`pure-panel ${className}`}>{(title || subtitle || action) && <div className="panel-heading"><div>{title && <h2>{title}</h2>}{subtitle && <p>{subtitle}</p>}</div>{action}</div>}{children}</section>;
}
export function Stat({ label, value, change, tone="blue", icon }: { label:string; value:string; change?:string; tone?:string; icon?:ReactNode }) {
  return <div className="stat-card"><div className="stat-top"><span>{label}</span><span className={`stat-icon tone-${tone}`}>{icon || <TrendingUp size={17}/>}</span></div><strong>{value}</strong>{change && <small className="stat-change"><TrendingUp size={13}/>{change}</small>}</div>;
}
export function Progress({ value, color="blue" }: { value:number; color?:string }) {
  return <div className="progress-track" role="progressbar" aria-valuenow={value} aria-valuemin={0} aria-valuemax={100}><span className={`progress-fill fill-${color}`} style={{width:`${Math.max(0,Math.min(100,value))}%`}} /></div>;
}
export function Donut({ segments, center, caption, labels=[], values=[] }: { segments:number[]; center:string; caption?:string; labels?:string[]; values?:string[] }) {
  const colors=["#3478f6","#78c9ee","#8e7cf4","#ffb56e","#08a47f","#ef8b8b"];
  const [active,setActive]=useState<number|null>(null);
  let offset=0;
  const circles=segments.map((value,i)=>{const current=offset;offset+=value;return <circle key={i} cx="60" cy="60" r="46" fill="none" stroke={colors[i%colors.length]} strokeWidth={active===i?17:13} strokeDasharray={`${value*2.89} 289`} strokeDashoffset={-current*2.89} transform="rotate(-90 60 60)" strokeLinecap="round" className="donut-segment" tabIndex={0} role="button" aria-label={`${labels[i]||"Категория"}: ${values[i]||value+"%"}`} onMouseEnter={()=>setActive(i)} onMouseLeave={()=>setActive(null)} onFocus={()=>setActive(i)} onBlur={()=>setActive(null)}><title>{labels[i]||"Категория"}: {values[i]||value.toFixed(1)+"%"}</title></circle>;});
  return <div className="donut-wrap"><svg viewBox="0 0 120 120" role="group" aria-label="Расходы по категориям">{circles}</svg><div className="donut-center"><strong>{active===null?center:(segments[active]||0).toFixed(1).replace(".",",")+"%"}</strong><small>{active===null?(caption||""):(labels[active]||"Категория")}</small>{active!==null&&values[active]&&<small className="donut-value">{values[active]}</small>}</div></div>;
}
export function LineChart({ values=[35,48,42,66,58,78,72], labels=["Янв","Фев","Мар","Апр","Май","Июн","Июл"], valueLabels=[] }: { values?:number[]; labels?:string[]; valueLabels?:string[] }) {
  const [active,setActive]=useState<number|null>(null);
  const max=Math.max(1,...values);
  const min=Math.min(0,...values);
  const span=Math.max(1,max-min);
  const coords=values.map((v,i)=>({x:values.length===1?150:12+i*(276/(values.length-1)),y:105-((v-min)/span)*82}));
  const points=coords.map(p=>`${p.x},${p.y}`).join(" ");
  const ticks=labels;
  const total=values.reduce((sum,value)=>sum+value,0);
  const average=values.length?total/values.length:0;
  const peak=values.length?Math.max(...values):0;
  const fmt=(value:number)=>new Intl.NumberFormat("ru-RU",{maximumFractionDigits:0}).format(value);
  return <div className="line-chart-layout"><div className="line-chart">
    {active!==null&&<div className="line-chart-tooltip" style={{left:`${Math.max(15,Math.min(285,coords[active].x))/3}%`,top:`${(coords[active].y/130)*190+8}px`,transform:"translate(-50%, calc(-100% - 9px))"}}><strong>{valueLabels[active]||values[active].toLocaleString("ru-RU")}</strong><span>{labels[active]}</span></div>}
    <svg viewBox="0 0 300 130" preserveAspectRatio="none" role="group" aria-label="Интерактивный график расходов">
      <defs><linearGradient id="line-fill" x1="0" x2="0" y1="0" y2="1"><stop offset="0%" stopColor="#6da8ff" stopOpacity=".28"/><stop offset="100%" stopColor="#6da8ff" stopOpacity="0"/></linearGradient></defs>
      {[23,49,75,101].map((y,i)=><g key={y}><line x1="8" x2="292" y1={y} y2={y} stroke="var(--chart-grid,#e9eef7)" strokeDasharray="3 5"/><text x="7" y={y-3} className="chart-axis-label">{Math.round(max-(i/3)*span).toLocaleString("ru-RU")}</text></g>)}
      {values.length>1&&<polygon points={`12,112 ${points} 288,112`} fill="url(#line-fill)"/>}
      <polyline points={points} fill="none" stroke="#438bfa" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
      {coords.map((p,i)=><circle key={i} cx={p.x} cy={p.y} r={1.75} fill={active===i?"#fff":"#438bfa"} stroke="#3478f6" strokeWidth="1" className="line-chart-point" tabIndex={0} role="button" aria-label={`${labels[i]}: ${valueLabels[i]||values[i]}`} onMouseEnter={()=>setActive(i)} onMouseLeave={()=>setActive(null)} onFocus={()=>setActive(i)} onBlur={()=>setActive(null)}><title>{labels[i]}: {valueLabels[i]||values[i].toLocaleString("ru-RU")}</title></circle>)}
    </svg>
    <div className="chart-labels">{ticks.map((x,i)=><span key={i}>{x}</span>)}</div></div><aside className="line-chart-legend" aria-label="Сводка расходов"><span className="line-chart-legend-title"><i/> Расходы</span><div><small>Всего</small><strong>{fmt(total)} ₽</strong></div><div><small>В среднем за день</small><strong>{fmt(average)} ₽</strong></div><div><small>Максимум за день</small><strong>{fmt(peak)} ₽</strong></div></aside>
  </div>;
}
export function BarChart({ values=[42,58,47,75,64,88,70], labels=["Пн","Вт","Ср","Чт","Пт","Сб","Вс"] }: { values?:number[]; labels?:string[] }) {
  return <div className="bar-chart">{values.map((v,i)=><div className="bar-column" key={i}><div className="bar-track"><span style={{height:`${v}%`}} className={i===3?"bar-highlight":""}/></div><small>{labels[i]}</small></div>)}</div>;
}
export function LinkAction({ children, onClick }: { children:string; onClick?:()=>void }) {
  return <button className="text-action" type="button" onClick={onClick}>{children}<ArrowUpRight size={14}/></button>;
}
export const money=(value:number)=>new Intl.NumberFormat("ru-RU",{style:"currency",currency:"RUB",maximumFractionDigits:0}).format(value);


export function ConfirmPopover({ title, message, confirmLabel="Удалить", onConfirm, onCancel }: { title:string; message:string; confirmLabel?:string; onConfirm:()=>void; onCancel:()=>void }) {
  return <div className="confirm-popover-backdrop" onMouseDown={e=>{if(e.target===e.currentTarget)onCancel();}}>
    <div className="confirm-popover" role="dialog" aria-modal="true" aria-labelledby="confirm-popover-title">
      <strong id="confirm-popover-title">{title}</strong>
      <p>{message}</p>
      <div className="confirm-popover-actions"><button type="button" className="pure-button ghost" onClick={onCancel}>Отмена</button><button type="button" className="pure-button danger" onClick={onConfirm}>{confirmLabel}</button></div>
    </div>
  </div>;
}
