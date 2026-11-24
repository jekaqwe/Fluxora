import { useEffect, useState } from "react";
import { WalletCards, Plus } from "lucide-react";
import { PageHeading, Panel } from "./ui";
import "./Pure.css";

type Account = {
  id: number;
  name: string;
  type: "cash" | "bank" | "card" | "savings" | "other";
  initialBalance: string;
  currentBalance: string;
  currency: "RUB";
  createdAt: string;
};

const accountTypes: Record<Account["type"], string> = {
  cash: "Наличные",
  bank: "Банковский счёт",
  card: "Карта",
  savings: "Накопительный",
  other: "Другой",
};

function Accounts() {
  const [items, setItems] = useState<Account[]>([]);
  const [name, setName] = useState("");
  const [type, setType] = useState<Account["type"]>("bank");
  const [balance, setBalance] = useState("0");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editName, setEditName] = useState("");
  const [editType, setEditType] = useState<Account["type"]>("bank");
  const [editBalance, setEditBalance] = useState("0");

  const load = async () => {
    try {
      const response = await fetch("/api/accounts");
      if (!response.ok) throw new Error("Не удалось загрузить счета");
      setItems(await response.json());
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void load(); }, []);

  const create = async () => {
    setError("");
    setSaving(true);
    try {
      const response = await fetch("/api/accounts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, type, initialBalance: balance.replace(",", ".") }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось создать счёт");
      setItems(old => [...old, data]);
      setName("");
      setBalance("0");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка создания счёта");
    } finally {
      setSaving(false);
    }
  };

  const beginEdit = (item: Account) => {
    setEditingId(item.id);
    setEditName(item.name);
    setEditType(item.type);
    setEditBalance(item.initialBalance);
  };

  const saveEdit = async () => {
    if (editingId === null) return;
    setError("");
    try {
      const response = await fetch(`/api/accounts/${editingId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: editName, type: editType, initialBalance: editBalance.replace(",", ".") }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось обновить счёт");
      setItems(old => old.map(item => item.id === data.id ? data : item));
      setEditingId(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка обновления счёта");
    }
  };

  const removeAccount = async (id: number) => {
    if (!window.confirm("Удалить счёт? Его история операций также будет удалена.")) return;
    setError("");
    try {
      const response = await fetch(`/api/accounts/${id}`, { method: "DELETE" });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось удалить счёт");
      setItems(old => old.filter(item => item.id !== id));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка удаления счёта");
    }
  };

  const totalCents = items.reduce((sum, item) => {
    const negative = item.currentBalance.startsWith("-");
    const [wholePart, fractionPart = ""] = item.currentBalance.replace("-", "").split(".");
    const value = BigInt(wholePart) * 100n + BigInt((fractionPart + "00").slice(0, 2));
    return sum + (negative ? -value : value);
  }, 0n);
  const absoluteCents = totalCents < 0n ? -totalCents : totalCents;
  const totalText = (totalCents < 0n ? "−" : "") +
    new Intl.NumberFormat("ru-RU").format(absoluteCents / 100n) + "," +
    (absoluteCents % 100n).toString().padStart(2, "0") + " ₽";

  return <div className="pure-page">
    <PageHeading title="Счета" subtitle="Ваши деньги в одном месте"/>
    <div className="pure-grid two">
      <Panel title="Общий баланс" subtitle="Сумма текущих остатков на всех ваших счетах">
        <div style={{ fontSize: 30, fontWeight: 700 }}>{totalText}</div>
        <p className="muted" style={{ fontSize: 11 }}>{items.length} {items.length === 1 ? "счёт" : "счетов"}</p>
      </Panel>
      <Panel title="Добавить счёт" subtitle="Укажите баланс на момент подключения к Fluxora">
        <label className="form-label">Название<input className="pure-input" value={name} maxLength={100} onChange={e => setName(e.target.value)} placeholder="Например, Основная карта"/></label>
        <label className="form-label" style={{ marginTop: 12 }}>Тип<select className="pure-select" value={type} onChange={e => setType(e.target.value as Account["type"])}>{Object.entries(accountTypes).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select></label>
        <label className="form-label" style={{ marginTop: 12 }}>Начальный баланс, ₽<input className="pure-input" inputMode="decimal" value={balance} onChange={e => setBalance(e.target.value)} placeholder="0.00"/></label>
        <button type="button" className="pure-button" style={{ marginTop: 14 }} disabled={saving || !name.trim() || !/^-?(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(balance)} onClick={() => void create()}><Plus size={16}/>{saving ? "Сохранение…" : "Добавить счёт"}</button>
      </Panel>
    </div>
    <Panel title="Мои счета" subtitle="Текущие остатки и начальные балансы в рублях">
      {loading ? <div className="empty-note">Загружаем счета…</div> : items.length === 0 ? <div className="empty-note">Счетов пока нет. Создайте первый, чтобы начать вести баланс.</div> :
        <div className="transaction-list">{items.map(item => <div className="transaction-row" key={item.id}>
          <span className="transaction-icon mint"><WalletCards size={16}/></span>
          {editingId === item.id ? <div className="transaction-copy" style={{ display: "grid", gap: 6 }}>
            <input className="pure-input" value={editName} onChange={e => setEditName(e.target.value)} aria-label="Название счёта"/>
            <select className="pure-select" value={editType} onChange={e => setEditType(e.target.value as Account["type"])}>{Object.entries(accountTypes).map(([key, label]) => <option key={key} value={key}>{label}</option>)}</select>
            <input className="pure-input" value={editBalance} onChange={e => setEditBalance(e.target.value)} aria-label="Начальный баланс"/>
            <div className="form-row"><button type="button" className="pure-button" onClick={() => void saveEdit()}>Сохранить</button><button type="button" className="pure-button ghost" onClick={() => setEditingId(null)}>Отмена</button></div>
          </div> : <>
            <span className="transaction-copy"><strong>{item.name}</strong><small>{accountTypes[item.type]}</small></span>
            <strong className="transaction-amount">{item.currentBalance} ₽</strong>
            <span className="transaction-date">Начальный: {item.initialBalance} ₽</span>
            <div className="form-row"><button type="button" className="pure-button secondary" onClick={() => beginEdit(item)}>Изменить</button><button type="button" className="pure-button ghost" onClick={() => void removeAccount(item.id)}>Удалить</button></div>
          </>}
        </div>)}</div>}
    </Panel>
    {error && <p role="alert" style={{ color: "#f87171", fontSize: 12 }}>{error}</p>}
  </div>;
}

export default Accounts;
