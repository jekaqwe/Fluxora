import type { ReactNode } from "react";
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
export function Donut({ segments, center, caption }: { segments:number[]; center:string; caption?:string }) {
  const colors=["#3478f6","#78c9ee","#8e7cf4","#ffb56e","#a9c1e8"];
  let offset=0;
  const circles=segments.map((value,i)=>{const current=offset;offset+=value;return <circle key={i} cx="60" cy="60" r="46" fill="none" stroke={colors[i%colors.length]} strokeWidth="13" strokeDasharray={`${value*2.89} 289`} strokeDashoffset={-current*2.89} transform="rotate(-90 60 60)" strokeLinecap="round"/>;});
  return <div className="donut-wrap"><svg viewBox="0 0 120 120" role="img" aria-label="Круговая диаграмма">{circles}</svg><div className="donut-center"><strong>{center}</strong>{caption && <small>{caption}</small>}</div></div>;
}
export function LineChart({ values=[35,48,42,66,58,78,72], labels=["Янв","Фев","Мар","Апр","Май","Июн","Июл"] }: { values?:number[]; labels?:string[] }) {
  const points=values.map((v,i)=>`${values.length===1?150:12+i*(276/(values.length-1))},${104-v*.85}`).join(" ");
  return <div className="line-chart"><svg viewBox="0 0 300 130" preserveAspectRatio="none" role="img" aria-label="График динамики"><defs><linearGradient id="line-fill" x1="0" x2="0" y1="0" y2="1"><stop offset="0%" stopColor="#6da8ff" stopOpacity=".22"/><stop offset="100%" stopColor="#6da8ff" stopOpacity="0"/></linearGradient></defs>{[25,55,85,115].map(y=><line key={y} x1="8" x2="292" y1={y} y2={y} stroke="#edf1f8" strokeDasharray="3 5"/>)}<polygon points={`12,115 ${points} 288,115`} fill="url(#line-fill)"/><polyline points={points} fill="none" stroke="#438bfa" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"/><circle cx={values.length===1?150:288} cy={104-values[values.length-1]*.85} r="4" fill="#fff" stroke="#3478f6" strokeWidth="3"/></svg><div className="chart-labels">{labels.map(x=><span key={x}>{x}</span>)}</div></div>;
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
