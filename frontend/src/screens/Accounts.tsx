import { useEffect, useState } from "react";
import { WalletCards, Plus } from "lucide-react";
import { PageHeading, Panel } from "./ui";
import "./Pure.css";

type Account = {
  id: number;
  name: string;
  type: "cash" | "bank" | "card" | "savings" | "other";
  initialBalance: string;
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
        body: JSON.stringify({ name, type, initialBalance: balance }),
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

  const total = items.reduce((sum, item) => sum + Number(item.initialBalance), 0);
  const totalText = new Intl.NumberFormat("ru-RU", { style: "currency", currency: "RUB" }).format(total);

  return <div className="pure-page">
    <PageHeading title="Счета" subtitle="Ваши деньги в одном месте"/>
    <div className="pure-grid two">
      <Panel title="Общий начальный баланс" subtitle="Сумма начальных балансов всех ваших счетов">
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
    <Panel title="Мои счета" subtitle="Начальные балансы в рублях">
      {loading ? <div className="empty-note">Загружаем счета…</div> : items.length === 0 ? <div className="empty-note">Счетов пока нет. Создайте первый, чтобы начать вести баланс.</div> :
        <div className="transaction-list">{items.map(item => <div className="transaction-row" key={item.id}>
          <span className="transaction-icon mint"><WalletCards size={16}/></span>
          <span className="transaction-copy"><strong>{item.name}</strong><small>{accountTypes[item.type]}</small></span>
          <strong className="transaction-amount">{item.initialBalance} ₽</strong>
          <span className="transaction-date">Начальный</span>
        </div>)}</div>}
    </Panel>
    {error && <p role="alert" style={{ color: "#f87171", fontSize: 12 }}>{error}</p>}
  </div>;
}

export default Accounts;
