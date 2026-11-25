import { useEffect, useMemo, useState } from "react";
import { ArrowDownLeft, ArrowUpRight, ArrowLeftRight, Coffee, ShoppingBag, Car, Plus, X, Trash2, Pencil } from "lucide-react";
import { PageHeading, Panel, Stat } from "./ui";
import "./Pure.css";
import RecurringPanel from "./RecurringPanel";
import CSVTools from "./CSVTools";

type OperationType = "income" | "expense" | "transfer";
type Account = { id: number; name: string; type: string; initialBalance: string; currency: "RUB" };
type Tx = {
  id: number;
  title: string;
  type: OperationType;
  amount: string;
  category: string;
  date: string;
  accountId: number;
  transferAccountId?: number;
  comment: string;
};
const cents = (value: string) => {
  const negative = value.startsWith("-");
  const [whole, fraction = ""] = value.replace("-", "").split(".");
  const result = BigInt(whole || "0") * 100n + BigInt((fraction + "00").slice(0, 2));
  return negative ? -result : result;
};
const formatCents = (value: bigint) => {
  const negative = value < 0n;
  const absolute = negative ? -value : value;
  return (negative ? "−" : "") + new Intl.NumberFormat("ru-RU").format(absolute / 100n) +
    "," + (absolute % 100n).toString().padStart(2, "0") + " ₽";
};
const isPositiveAmount = (value: string) => {
  if (!/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(value)) return false;
  const [whole, fraction = ""] = value.replace(",", ".").split(".");
  return BigInt(whole) * 100n + BigInt((fraction + "00").slice(0, 2)) > 0n;
};
const localDate = () => {
  const now = new Date();
  return [now.getFullYear(), String(now.getMonth() + 1).padStart(2, "0"), String(now.getDate()).padStart(2, "0")].join("-");
};
const typeLabels: Record<OperationType, string> = { income: "Доход", expense: "Расход", transfer: "Перевод" };

function Operations() {
  const [items, setItems] = useState<Tx[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [filter, setFilter] = useState("all");
  const [query, setQuery] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [title, setTitle] = useState("");
  const [amount, setAmount] = useState("");
  const [category, setCategory] = useState("Покупки");
  const [date, setDate] = useState(localDate());
  const [accountId, setAccountId] = useState("");
  const [transferAccountId, setTransferAccountId] = useState("");
  const [kind, setKind] = useState<OperationType>("expense");
  const [comment, setComment] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);

  const load = async () => {
    try {
      const [txResponse, accountResponse] = await Promise.all([fetch("/api/transactions"), fetch("/api/accounts")]);
      if (!txResponse.ok || !accountResponse.ok) throw new Error("Не удалось загрузить финансовые данные");
      const [txData, accountData] = await Promise.all([txResponse.json(), accountResponse.json()]);
      setItems(txData);
      setAccounts(accountData);
      if (accountData.length && !accountId) setAccountId(String(accountData[0].id));
      if (accountData.length > 1 && !transferAccountId) setTransferAccountId(String(accountData[1].id));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    }
  };
  useEffect(() => { void load(); }, []);

  const visible = useMemo(() => items.filter(item =>
    (filter === "all" || item.type === filter) &&
    (item.title + " " + item.category + " " + item.comment).toLowerCase().includes(query.toLowerCase())
  ), [items, filter, query]);

  const income = items.filter(x => x.type === "income").reduce((sum, x) => sum + cents(x.amount), 0n);
  const expense = items.filter(x => x.type === "expense").reduce((sum, x) => sum + cents(x.amount), 0n);

  const add = async () => {
    setError("");
    setSaving(true);
    try {
      const response = await fetch(editingId === null ? "/api/transactions" : `/api/transactions/${editingId}`, {
        method: editingId === null ? "POST" : "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title, type: kind, amount: amount.replace(",", "."), category, date,
          accountId: Number(accountId),
          transferAccountId: kind === "transfer" ? Number(transferAccountId) : 0,
          comment, ...(editingId === null ? { idempotencyKey: crypto.randomUUID() } : {}),
        }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось сохранить операцию");
      setItems(old => editingId === null ? [data, ...old.filter(item => item.id !== data.id)] : old.map(item => item.id === data.id ? data : item));
      setTitle(""); setAmount(""); setComment(""); setShowForm(false); setEditingId(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка сохранения");
    } finally {
      setSaving(false);
    }
  };

  const beginEdit = (item: Tx) => {
    setEditingId(item.id);
    setTitle(item.title);
    setKind(item.type);
    setAmount(item.amount);
    setCategory(item.category);
    setDate(item.date);
    setAccountId(String(item.accountId));
    setTransferAccountId(item.transferAccountId ? String(item.transferAccountId) : "");
    setComment(item.comment);
    setShowForm(true);
  };

  const remove = async (id: number) => {
    if (!window.confirm("Удалить эту операцию?")) return;
    try {
      const response = await fetch(`/api/transactions/${id}`, { method: "DELETE" });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось удалить операцию");
      setItems(old => old.filter(item => item.id !== id));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка удаления");
    }
  };

  return <div className="pure-page" style={{padding:16, display:"flex", flexDirection:"column", gap:16}}>
    <PageHeading title="Операции" subtitle="Доходы, расходы и переводы между счетами" action={<button className="pure-button" onClick={() => setShowForm(v => !v)}><Plus size={16}/>Добавить операцию</button>}/>
    <div className="pure-grid three">
      <Stat label="Доходы" value={formatCents(income)} tone="mint" icon={<ArrowDownLeft size={17}/>}/>
      <Stat label="Расходы" value={formatCents(expense)} tone="violet" icon={<ArrowUpRight size={17}/>}/>
      <Stat label="Разница" value={formatCents(income - expense)} change="Без учёта переводов"/>
    </div>
    {showForm && <Panel title="Новая операция" className="form-panel">
      {accounts.length === 0 ? <div className="empty-note">Сначала создайте хотя бы один счёт в разделе «Счета».</div> : <>
        <div className="pure-grid three">
          <label className="form-label">Тип<select className="pure-select" value={kind} onChange={e => setKind(e.target.value as OperationType)}>{Object.entries(typeLabels).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></label>
          <label className="form-label">Название<input className="pure-input" value={title} onChange={e => setTitle(e.target.value)} maxLength={160} placeholder="Например, Продукты"/></label>
          <label className="form-label">Сумма, ₽<input className="pure-input" inputMode="decimal" value={amount} onChange={e => setAmount(e.target.value)} placeholder="0.00"/></label>
          <label className="form-label">Категория<input className="pure-input" value={category} onChange={e => setCategory(e.target.value)} maxLength={80}/></label>
          <label className="form-label">Дата<input className="pure-input" type="date" value={date} onChange={e => setDate(e.target.value)}/></label>
          <label className="form-label">{kind === "transfer" ? "С какого счёта" : "Счёт"}<select className="pure-select" value={accountId} onChange={e => setAccountId(e.target.value)}>{accounts.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}</select></label>
          {kind === "transfer" && <label className="form-label">На какой счёт<select className="pure-select" value={transferAccountId} onChange={e => setTransferAccountId(e.target.value)}>{accounts.filter(a => String(a.id) !== accountId).map(a => <option key={a.id} value={a.id}>{a.name}</option>)}</select></label>}
          <label className="form-label">Комментарий<input className="pure-input" value={comment} onChange={e => setComment(e.target.value)} maxLength={1000}/></label>
        </div>
        <div className="form-row" style={{marginTop:14}}><button className="pure-button" disabled={saving || !title.trim() || !isPositiveAmount(amount) || (kind === "transfer" && (!transferAccountId || transferAccountId === accountId))} onClick={() => void add()}>{saving ? "Сохранение…" : editingId === null ? "Сохранить операцию" : "Сохранить изменения"}</button><button className="pure-button ghost" onClick={() => {setShowForm(false);setEditingId(null);}}><X size={15}/>Отмена</button></div>
      </>}
    </Panel>}
    <RecurringPanel accounts={accounts}/>
    <CSVTools accounts={accounts} onImported={()=>void load()}/>
    <Panel title="История операций" subtitle={`${visible.length} записей`}>
      <div className="form-row" style={{justifyContent:"space-between",marginBottom:16,flexWrap:"wrap"}}>
        <div className="filter-pills">{[["all","Все"],["income","Доходы"],["expense","Расходы"],["transfer","Переводы"]].map(([key,label])=><button key={key} className={`filter-pill ${filter===key?"active":""}`} onClick={()=>setFilter(key)}>{label}</button>)}</div>
        <input className="pure-input" style={{maxWidth:240}} value={query} onChange={e=>setQuery(e.target.value)} placeholder="Поиск по операциям"/>
      </div>
      <div className="transaction-list">{visible.map(item => {
        const isIncome = item.type === "income";
        const isTransfer = item.type === "transfer";
        const account = accounts.find(a => a.id === item.accountId)?.name || "Счёт";
        const destination = accounts.find(a => a.id === item.transferAccountId)?.name;
        const Icon = isIncome ? ArrowDownLeft : isTransfer ? ArrowLeftRight : item.category === "Транспорт" ? Car : item.category === "Кафе" ? Coffee : ShoppingBag;
        return <div className="transaction-row" key={item.id}>
          <span className={`transaction-icon ${isIncome ? "mint" : isTransfer ? "violet" : ""}`}><Icon size={16}/></span>
          <span className="transaction-copy"><strong>{item.title}</strong><small>{item.category} · {typeLabels[item.type]} · {isTransfer ? account + " → " + (destination || "счёт") : account}</small></span>
          <strong className={`transaction-amount ${isIncome ? "positive" : isTransfer ? "" : "negative"}`}>{isIncome ? "+" : isTransfer ? "↔ " : "−"}{item.amount} ₽</strong>
          <span className="transaction-date">{item.date}</span>
          <button type="button" className="pure-button ghost" aria-label="Изменить операцию" onClick={() => beginEdit(item)}><Pencil size={14}/></button><button type="button" className="pure-button ghost" aria-label="Удалить операцию" onClick={() => void remove(item.id)}><Trash2 size={14}/></button>
        </div>;
      })}{visible.length===0&&<div className="empty-note">Операций по выбранным условиям не найдено.</div>}</div>
    </Panel>
    {error && <p role="alert" style={{color:"#f87171",fontSize:12}}>{error}</p>}
  </div>;
}
export default Operations;
