import { useEffect, useState } from "react";
import { Plus, X } from "lucide-react";
import { Panel } from "./ui";

type Account = { id: number; name: string };
type OperationType = "income" | "expense";

const localDate = () => {
  const now = new Date();
  return [now.getFullYear(), String(now.getMonth() + 1).padStart(2, "0"), String(now.getDate()).padStart(2, "0")].join("-");
};
const validAmount = (value: string) => {
  if (!/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(value)) return false;
  const [whole, fraction = ""] = value.replace(",", ".").split(".");
  return BigInt(whole) * 100n + BigInt((fraction + "00").slice(0, 2)) > 0n;
};

export default function QuickEntry({ onSaved }: { onSaved: () => void }) {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [open, setOpen] = useState(false);
  const [kind, setKind] = useState<OperationType>("expense");
  const [title, setTitle] = useState("");
  const [amount, setAmount] = useState("");
  const [category, setCategory] = useState("Покупки");
  const [date, setDate] = useState(localDate());
  const [accountId, setAccountId] = useState("");
  const [comment, setComment] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetch("/api/accounts")
      .then(async response => {
        const data = await response.json();
        if (!response.ok) throw new Error(data.error || "Не удалось загрузить счета");
        if (!cancelled) {
          setAccounts(data);
          if (data.length) setAccountId(String(data[0].id));
        }
      })
      .catch(reason => { if (!cancelled) setError(reason instanceof Error ? reason.message : "Ошибка загрузки счетов"); });
    return () => { cancelled = true; };
  }, []);

  const save = async () => {
    setError("");
    setSaving(true);
    try {
      const response = await fetch("/api/transactions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          title: title.trim(), type: kind, amount: amount.replace(",", "."),
          category: category.trim(), date, accountId: Number(accountId),
          transferAccountId: 0, comment: comment.trim(), idempotencyKey: crypto.randomUUID(),
        }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось сохранить операцию");
      setTitle(""); setAmount(""); setComment(""); setCategory("Покупки"); setDate(localDate());
      setKind("expense"); setOpen(false);
      onSaved();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Ошибка сохранения");
    } finally {
      setSaving(false);
    }
  };

  return <Panel title="Быстрая операция" subtitle="Добавьте доход или расход, не переходя в историю"
    action={<button type="button" className="pure-button" onClick={() => setOpen(value => !value)}>{open ? <X size={15}/> : <Plus size={15}/>} {open ? "Свернуть" : "Добавить"}</button>}>
    {open && (accounts.length === 0 ? <div className="empty-note">Сначала создайте счёт в разделе «Счета».</div> : <>
      <div className="pure-grid three">
        <label className="form-label">Тип
          <select className="pure-select" value={kind} onChange={event => setKind(event.target.value as OperationType)}>
            <option value="expense">Расход</option><option value="income">Доход</option>
          </select>
        </label>
        <label className="form-label">Название
          <input className="pure-input" value={title} onChange={event => setTitle(event.target.value)} maxLength={160} placeholder="Например, Продукты"/>
        </label>
        <label className="form-label">Сумма, ₽
          <input className="pure-input" inputMode="decimal" value={amount} onChange={event => setAmount(event.target.value)} placeholder="0,00"/>
        </label>
        <label className="form-label">Категория
          <input className="pure-input" value={category} onChange={event => setCategory(event.target.value)} maxLength={80}/>
        </label>
        <label className="form-label">Дата
          <input className="pure-input" type="date" value={date} onChange={event => setDate(event.target.value)}/>
        </label>
        <label className="form-label">Счёт
          <select className="pure-select" value={accountId} onChange={event => setAccountId(event.target.value)}>
            {accounts.map(account => <option key={account.id} value={account.id}>{account.name}</option>)}
          </select>
        </label>
        <label className="form-label">Комментарий
          <input className="pure-input" value={comment} onChange={event => setComment(event.target.value)} maxLength={1000}/>
        </label>
      </div>
      {error && <p role="alert" style={{color:"#dc5555",fontSize:12}}>{error}</p>}
      <div className="form-row" style={{marginTop:14}}>
        <button type="button" className="pure-button" disabled={saving || !title.trim() || !category.trim() || !accountId || !validAmount(amount)} onClick={() => void save()}>
          {saving ? "Сохранение…" : "Сохранить операцию"}
        </button>
        <button type="button" className="pure-button ghost" onClick={() => setOpen(false)}>Отмена</button>
      </div>
    </>)}
  </Panel>;
}
