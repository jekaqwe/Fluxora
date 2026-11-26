import { useEffect, useState } from "react";
import { ConfirmPopover, PageHeading, Panel, Stat } from "./ui";
import {
  Plus,
  ArrowDownLeft,
  ArrowUpRight,
  Trash2,
  History,
  X,
} from "lucide-react";
import "./Pure.css";

type Account = { id: number; name: string };
type Debt = {
  id: number;
  name: string;
  direction: "i_owe" | "owed_to_me";
  initialAmount: string;
  remainingAmount: string;
  dueDate?: string;
  comment: string;
  status: "open" | "closed";
  createdAt: string;
};
type Payment = {
  id: number;
  amount: string;
  accountId: number;
  transactionId: number;
  date: string;
  comment: string;
  createdAt: string;
};
const toCents = (v: string) => {
  const [whole, fraction = ""] = v.replace(",", ".").split(".");
  return BigInt(whole || "0") * 100n + BigInt((fraction + "00").slice(0, 2));
};
const money = (v: string) => {
  const neg = v.startsWith("-");
  const abs = v.replace("-", "");
  const cents = toCents(abs);
  return (
    (neg ? "−" : "") +
    new Intl.NumberFormat("ru-RU").format(cents / 100n) +
    "," +
    (cents % 100n).toString().padStart(2, "0") +
    " ₽"
  );
};
const today = () => {
  const d = new Date();
  return [
    d.getFullYear(),
    String(d.getMonth() + 1).padStart(2, "0"),
    String(d.getDate()).padStart(2, "0"),
  ].join("-");
};

function Debts() {
  const [debts, setDebts] = useState<Debt[]>([]);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [direction, setDirection] = useState<"i_owe" | "owed_to_me">("i_owe");
  const [initialAmount, setInitialAmount] = useState("");
  const [dueDate, setDueDate] = useState("");
  const [comment, setComment] = useState("");
  const [paymentDebtId, setPaymentDebtId] = useState<number | null>(null);
  const [paymentAmount, setPaymentAmount] = useState("");
  const [paymentAccountId, setPaymentAccountId] = useState("");
  const [paymentDate, setPaymentDate] = useState(today());
  const [paymentKey, setPaymentKey] = useState("");
  const [history, setHistory] = useState<Record<number, Payment[]>>({});
  const [historyDebtId, setHistoryDebtId] = useState<number | null>(null);
  const [confirmDeleteId, setConfirmDeleteId] = useState<number | null>(null);

  const load = async () => {
    try {
      const [d, a] = await Promise.all([
        fetch("/api/debts"),
        fetch("/api/accounts"),
      ]);
      const [debtData, accountData] = await Promise.all([d.json(), a.json()]);
      if (!d.ok || !a.ok) throw new Error("Не удалось загрузить долги и счета");
      setDebts(debtData);
      setAccounts(accountData);
      if (accountData.length && !paymentAccountId)
        setPaymentAccountId(String(accountData[0].id));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка загрузки");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  const create = async () => {
    setError("");
    try {
      const r = await fetch("/api/debts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name,
          direction,
          initialAmount: initialAmount.replace(",", "."),
          dueDate,
          comment,
        }),
      });
      const d = await r.json();
      if (!r.ok) throw new Error(d.error || "Не удалось создать долг");
      setDebts((old) => [d, ...old]);
      setShowForm(false);
      setName("");
      setInitialAmount("");
      setComment("");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка создания");
    }
  };
  const pay = async () => {
    if (paymentDebtId === null) return;
    setError("");
    try {
      const r = await fetch(`/api/debts/${paymentDebtId}/payments`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          amount: paymentAmount.replace(",", "."),
          accountId: Number(paymentAccountId),
          date: paymentDate,
          comment: "",
          idempotencyKey: paymentKey,
        }),
      });
      const d = await r.json();
      if (!r.ok) throw new Error(d.error || "Не удалось записать платёж");
      setPaymentDebtId(null);
      setPaymentAmount("");
      setPaymentKey("");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка платежа");
    }
  };
  const remove = async (id: number) => {
    try {
      const r = await fetch(`/api/debts/${id}`, { method: "DELETE" });
      const d = await r.json();
      if (!r.ok) throw new Error(d.error || "Не удалось удалить долг");
      setDebts((old) => old.filter((x) => x.id !== id));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка удаления");
    }
  };
  const toggleStatus = async (debt: Debt) => {
    try {
      const r = await fetch(`/api/debts/${debt.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: debt.name,
          dueDate: debt.dueDate || "",
          comment: debt.comment,
          status: debt.status === "open" ? "closed" : "open",
        }),
      });
      const d = await r.json();
      if (!r.ok) throw new Error(d.error || "Не удалось изменить статус");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка статуса");
    }
  };
  const showHistory = async (id: number) => {
    if (historyDebtId === id) {
      setHistoryDebtId(null);
      return;
    }
    try {
      const r = await fetch(`/api/debts/${id}/payments`);
      const d = await r.json();
      if (!r.ok) throw new Error(d.error || "Не удалось загрузить историю");
      setHistory((old) => ({ ...old, [id]: d }));
      setHistoryDebtId(id);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка истории");
    }
  };
  const iOwe = debts
    .filter((d) => d.direction === "i_owe" && d.status === "open")
    .reduce((s, d) => s + toCents(d.remainingAmount), 0n);
  const owed = debts
    .filter((d) => d.direction === "owed_to_me" && d.status === "open")
    .reduce((s, d) => s + toCents(d.remainingAmount), 0n);

  return (
    <div className="pure-page">
      <PageHeading
        title="Долги и кредиты"
        subtitle="Упрощённый учёт обязательств и возвратов"
        action={
          <button
            className="pure-button"
            onClick={() => setShowForm((v) => !v)}
          >
            <Plus size={15} />
            Добавить долг
          </button>
        }
      />
      <div className="pure-grid two">
        <Stat
          label="Я должен"
          value={moneyCents(iOwe)}
          tone="violet"
          icon={<ArrowUpRight size={17} />}
        />
        <Stat
          label="Мне должны"
          value={moneyCents(owed)}
          tone="mint"
          icon={<ArrowDownLeft size={17} />}
        />
      </div>
      {showForm && (
        <div
          className="debt-modal-backdrop"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setShowForm(false);
          }}
        >
          <section
            className="debt-modal"
            role="dialog"
            aria-modal="true"
            aria-labelledby="debt-modal-title"
          >
            <div className="debt-modal-heading">
              <div>
                <strong id="debt-modal-title">Новый долг</strong>
                <small>Заполните информацию об обязательстве</small>
              </div>
              <button
                type="button"
                className="pure-button ghost debt-modal-close"
                aria-label="Закрыть"
                onClick={() => setShowForm(false)}
              >
                <X size={17} />
              </button>
            </div>
            <div className="pure-grid two">
              <label className="form-label">
                Название
                <input
                  className="pure-input"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Например, Кредит"
                />
              </label>
              <label className="form-label">
                Направление
                <select
                  className="pure-select"
                  value={direction}
                  onChange={(e) =>
                    setDirection(e.target.value as "i_owe" | "owed_to_me")
                  }
                >
                  <option value="i_owe">Я должен</option>
                  <option value="owed_to_me">Мне должны</option>
                </select>
              </label>
              <label className="form-label">
                Сумма, ₽
                <input
                  className="pure-input"
                  value={initialAmount}
                  onChange={(e) => setInitialAmount(e.target.value)}
                  placeholder="0.00"
                />
              </label>
              <label className="form-label">
                Срок возврата
                <input
                  className="pure-input"
                  type="date"
                  value={dueDate}
                  onChange={(e) => setDueDate(e.target.value)}
                />
              </label>
              <label className="form-label debt-modal-comment">
                Комментарий
                <input
                  className="pure-input"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                  maxLength={1000}
                />
              </label>
            </div>
            {error && (
              <p role="alert" className="debt-modal-error">
                {error}
              </p>
            )}
            <div className="debt-modal-actions">
              <button
                type="button"
                className="pure-button ghost"
                onClick={() => setShowForm(false)}
              >
                Отмена
              </button>
              <button
                type="button"
                className="pure-button"
                disabled={
                  !name.trim() ||
                  !/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(
                    initialAmount,
                  ) ||
                  toCents(initialAmount) <= 0n
                }
                onClick={() => void create()}
              >
                Сохранить долг
              </button>
            </div>
          </section>
        </div>
      )}
      {loading ? (
        <div className="empty-note">Загружаем долги…</div>
      ) : debts.length === 0 ? (
        <div className="empty-note">Долгов пока нет.</div>
      ) : (
        <div className="pure-grid two" style={{ marginTop: 16 }}>
          {debts.map((debt) => (
            <Panel
              key={debt.id}
              title={debt.name}
              subtitle={
                debt.direction === "i_owe" ? "Моё обязательство" : "Мне должны"
              }
            >
              <div className="legend-row">
                <span>Остаток</span>
                <strong>{money(debt.remainingAmount)}</strong>
              </div>
              <div className="legend-row" style={{ marginTop: 8 }}>
                <span>Изначальная сумма</span>
                <strong>{money(debt.initialAmount)}</strong>
              </div>
              <div className="legend-row" style={{ marginTop: 8 }}>
                <span>Срок</span>
                <strong>{debt.dueDate || "Не задан"}</strong>
              </div>
              <div className="legend-row" style={{ marginTop: 8 }}>
                <span>Статус</span>
                <strong>
                  {debt.status === "open" ? "Активен" : "В архиве"}
                </strong>
              </div>
              <div
                className="form-row"
                style={{ marginTop: 12, flexWrap: "wrap" }}
              >
                <button
                  className="pure-button"
                  disabled={
                    debt.status !== "open" ||
                    toCents(debt.remainingAmount) <= 0n ||
                    accounts.length === 0
                  }
                  onClick={() => {
                    setPaymentDebtId(debt.id);
                    setPaymentKey(crypto.randomUUID());
                  }}
                >
                  Платёж
                </button>
                <button
                  className="pure-button ghost"
                  onClick={() => void showHistory(debt.id)}
                >
                  <History size={14} />
                  История
                </button>
                <button
                  className="pure-button ghost"
                  onClick={() => void toggleStatus(debt)}
                >
                  {debt.status === "open" ? "В архив" : "Вернуть в активные"}
                </button>
                <button
                  className="pure-button ghost"
                  onClick={() => setConfirmDeleteId(debt.id)}
                >
                  <Trash2 size={14} />
                </button>
              </div>
              {paymentDebtId === debt.id && (
                <div style={{ marginTop: 12 }}>
                  <label className="form-label">
                    Сумма платежа
                    <input
                      className="pure-input"
                      value={paymentAmount}
                      onChange={(e) => setPaymentAmount(e.target.value)}
                    />
                  </label>
                  <label className="form-label" style={{ marginTop: 8 }}>
                    Счёт
                    <select
                      className="pure-select"
                      value={paymentAccountId}
                      onChange={(e) => setPaymentAccountId(e.target.value)}
                    >
                      {accounts.map((a) => (
                        <option key={a.id} value={a.id}>
                          {a.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="form-label" style={{ marginTop: 8 }}>
                    Дата
                    <input
                      className="pure-input"
                      type="date"
                      value={paymentDate}
                      onChange={(e) => setPaymentDate(e.target.value)}
                    />
                  </label>
                  <div className="form-row" style={{ marginTop: 8 }}>
                    <button
                      className="pure-button"
                      disabled={
                        !paymentAccountId ||
                        !/^(?:0|[1-9][0-9]{0,17})(?:[.,][0-9]{1,2})?$/.test(
                          paymentAmount,
                        ) ||
                        toCents(paymentAmount) <= 0n ||
                        toCents(paymentAmount) > toCents(debt.remainingAmount)
                      }
                      onClick={() => void pay()}
                    >
                      Подтвердить платёж
                    </button>
                    <button
                      className="pure-button ghost"
                      onClick={() => setPaymentDebtId(null)}
                    >
                      Отмена
                    </button>
                  </div>
                </div>
              )}
              {historyDebtId === debt.id && (
                <div className="empty-note" style={{ marginTop: 12 }}>
                  {(history[debt.id] || []).length
                    ? (history[debt.id] || []).map((p) => (
                        <div className="legend-row" key={p.id}>
                          <span>{p.date}</span>
                          <strong>{money(p.amount)}</strong>
                        </div>
                      ))
                    : "Платежей пока нет."}
                </div>
              )}
            </Panel>
          ))}
        </div>
      )}
      {confirmDeleteId !== null && (
        <ConfirmPopover
          title="Удалить долг?"
          message="Если по нему уже есть платежи, удаление будет запрещено, чтобы сохранить финансовую историю."
          onCancel={() => setConfirmDeleteId(null)}
          onConfirm={() => {
            const id = confirmDeleteId;
            setConfirmDeleteId(null);
            void remove(id);
          }}
        />
      )}
      {error && (
        <p role="alert" style={{ color: "#f87171", fontSize: 12 }}>
          {error}
        </p>
      )}
    </div>
  );
}
function moneyCents(v: bigint) {
  const neg = v < 0n;
  const abs = neg ? -v : v;
  return (
    (neg ? "−" : "") +
    new Intl.NumberFormat("ru-RU").format(abs / 100n) +
    "," +
    (abs % 100n).toString().padStart(2, "0") +
    " ₽"
  );
}
export default Debts;
