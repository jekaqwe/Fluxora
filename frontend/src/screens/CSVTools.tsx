import { useState, type ChangeEvent } from "react";
import { Panel } from "./ui";
import { Download, FileUp, Check, AlertTriangle } from "lucide-react";

type Account={id:number;name:string};
type PreviewRow={date:string;title:string;type:string;amount:string;category:string;accountId:number;transferAccountId:number;comment:string;errors:string[]};
export default function CSVTools({accounts,onImported}:{accounts:Account[];onImported:()=>void}){
 const [file,setFile]=useState<File|null>(null);
 const [rows,setRows]=useState<PreviewRow[]>([]);
 const [valid,setValid]=useState(0);
 const [invalid,setInvalid]=useState(0);
 const [error,setError]=useState("");
 const [busy,setBusy]=useState(false);
 const [from,setFrom]=useState("");
 const [to,setTo]=useState("");
 const [accountId,setAccountId]=useState("");
 const [category,setCategory]=useState("");
 const [previewed,setPreviewed]=useState(false);
 const preview=async(event:ChangeEvent<HTMLInputElement>)=>{
  const selected=event.target.files?.[0]||null;event.target.value="";if(!selected)return;
  setFile(selected);setError("");setPreviewed(false);setBusy(true);
  try{const form=new FormData();form.append("file",selected);const r=await fetch("/api/import/preview",{method:"POST",body:form});const d=await r.json();if(!r.ok)throw new Error(d.error||"Не удалось проверить CSV");setRows(d.rows||[]);setValid(d.valid||0);setInvalid(d.invalid||0);setPreviewed(true);}
  catch(e){setError(e instanceof Error?e.message:"Ошибка проверки");setRows([]);}finally{setBusy(false);}
 };
 const confirm=async()=>{
  if(!file)return;setBusy(true);setError("");
  try{const form=new FormData();form.append("file",file);const r=await fetch("/api/import/confirm",{method:"POST",body:form});const d=await r.json();if(!r.ok)throw new Error(d.error||"Импорт не выполнен");setFile(null);setRows([]);setPreviewed(false);onImported();}
  catch(e){setError(e instanceof Error?e.message:"Ошибка импорта");}finally{setBusy(false);}
 };
 const exportURL="/api/export.csv?"+new URLSearchParams({from,to,accountId,category}).toString();
 return <Panel title="CSV: импорт и экспорт" subtitle="Переносите операции между Fluxora и таблицами">
  <div className="pure-grid two">
   <div>
    <strong>Экспорт операций</strong>
    <div className="pure-grid two" style={{marginTop:10}}>
     <label className="form-label">С даты<input className="pure-input" type="date" value={from} onChange={e=>setFrom(e.target.value)}/></label>
     <label className="form-label">По дату<input className="pure-input" type="date" value={to} onChange={e=>setTo(e.target.value)}/></label>
     <label className="form-label">Счёт<select className="pure-select" value={accountId} onChange={e=>setAccountId(e.target.value)}><option value="">Все счета</option>{accounts.map(a=><option key={a.id} value={a.id}>{a.name}</option>)}</select></label>
     <label className="form-label">Категория<input className="pure-input" value={category} onChange={e=>setCategory(e.target.value)} placeholder="Все категории"/></label>
    </div>
    <a className="pure-button" href={exportURL} style={{marginTop:12,display:"inline-flex"}}><Download size={15}/>Скачать CSV</a>
   </div>
   <div>
    <strong>Импорт операций</strong>
    <label className="pure-button secondary" style={{marginTop:10,cursor:"pointer",display:"inline-flex"}}><FileUp size={15}/>{busy?"Проверяем…":"Выбрать CSV"}<input type="file" accept=".csv,text/csv" hidden disabled={busy} onChange={preview}/></label>
    <p className="muted" style={{fontSize:11,marginTop:8}}>До 5 МБ. Используйте CSV, экспортированный из Fluxora, с колонками date, title, type, amount, category, accountId, transferAccountId, comment.</p>
    {previewed&&<div className="empty-note" style={{marginTop:10,display:"flex",gap:12,alignItems:"center"}}><Check size={16}/>Проверено строк: {rows.length}. Корректных: {valid}. С ошибками: {invalid}.</div>}
    {previewed&&invalid===0&&rows.length>0&&<button className="pure-button" disabled={busy} style={{marginTop:10}} onClick={()=>void confirm()}><Check size={15}/>Подтвердить импорт</button>}
   </div>
  </div>
  {previewed&&rows.length>0&&<div className="transaction-list" style={{marginTop:14}}>{rows.slice(0,12).map((row,index)=><div className="transaction-row" key={index}>
   <span className={`transaction-icon ${row.errors.length?"":"mint"}`}>{row.errors.length?<AlertTriangle size={15}/>:<Check size={15}/>}</span>
   <span className="transaction-copy"><strong>{row.title||"Без названия"} · {row.amount} ₽</strong><small>{row.date} · {row.type} · счёт #{row.accountId}</small>{row.errors.length>0&&<small style={{color:"#f87171"}}>{row.errors.join("; ")}</small>}</span>
  </div>)}</div>}
  {rows.length>12&&<p className="muted" style={{fontSize:11}}>Показаны первые 12 строк предварительного просмотра.</p>}
  {error&&<p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
 </Panel>;
}
