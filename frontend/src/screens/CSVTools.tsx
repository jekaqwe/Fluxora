import { useState, type ChangeEvent } from "react";
import { Panel } from "./ui";
import { Download, FileUp, Check, AlertTriangle, Columns3 } from "lucide-react";

type Account = { id: number; name: string };
type PreviewRow = { date: string; title: string; type: string; amount: string; category: string; accountId: number; transferAccountId: number; comment: string; errors: string[] };
type Mapping = Record<string, string>;
const tinkoffTypeValue = "__tinkoff_type_from_amount__";
const tinkoffAccountPrefix = "__tinkoff_account_id__:";
const fields = [
  { key: "date", label: "Дата", required: true },
  { key: "title", label: "Название", required: true },
  { key: "type", label: "Тип операции", required: true },
  { key: "amount", label: "Сумма", required: true },
  { key: "category", label: "Категория", required: true },
  { key: "accountId", label: "Счёт Fluxora", required: true },
  { key: "transferAccountId", label: "ID счёта перевода", required: false },
  { key: "comment", label: "Комментарий", required: false },
];

export default function CSVTools({ accounts, onImported }: { accounts: Account[]; onImported: () => void }) {
  const [file, setFile] = useState<File | null>(null);
  const [headers, setHeaders] = useState<string[]>([]);
  const [mapping, setMapping] = useState<Mapping>({});
  const [format, setFormat] = useState("generic");
  const [rows, setRows] = useState<PreviewRow[]>([]);
  const [valid, setValid] = useState(0);
  const [invalid, setInvalid] = useState(0);
  const [skipped, setSkipped] = useState(0);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [accountId, setAccountId] = useState("");
  const [category, setCategory] = useState("");
  const [previewed, setPreviewed] = useState(false);

  const chooseFile = async (event: ChangeEvent<HTMLInputElement>) => {
    const selected = event.target.files?.[0] || null;
    event.target.value = "";
    if (!selected) return;
    setFile(selected);
    setError("");
    setRows([]);
    setPreviewed(false);
    setBusy(true);
    try {
      const form = new FormData();
      form.append("file", selected);
      const response = await fetch("/api/import/headers", { method: "POST", body: form });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось прочитать заголовки CSV");
      const detectedFormat = data.format || "generic";
      const suggested = data.suggestedMapping || {};
      setHeaders(data.headers || []);
      setFormat(detectedFormat);
      if (detectedFormat === "tinkoff" && accounts.length === 1) {
        suggested.accountId = tinkoffAccountPrefix + accounts[0].id;
      }
      setMapping(suggested);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка чтения файла");
      setFile(null);
      setHeaders([]);
    } finally {
      setBusy(false);
    }
  };

  const preview = async () => {
    if (!file) return;
    setBusy(true);
    setError("");
    setPreviewed(false);
    try {
      const form = new FormData();
      form.append("file", file);
      form.append("mapping", JSON.stringify(mapping));
      const response = await fetch("/api/import/preview", { method: "POST", body: form });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось проверить CSV");
      setRows(data.rows || []);
      setValid(data.valid || 0);
      setInvalid(data.invalid || 0);
      setSkipped(data.skipped || 0);
      setPreviewed(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка проверки");
      setRows([]);
    } finally {
      setBusy(false);
    }
  };

  const confirm = async () => {
    if (!file) return;
    setBusy(true);
    setError("");
    try {
      const form = new FormData();
      form.append("file", file);
      form.append("mapping", JSON.stringify(mapping));
      const response = await fetch("/api/import/confirm", { method: "POST", body: form });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Импорт не выполнен");
      setFile(null);
      setHeaders([]);
      setMapping({});
      setRows([]);
      setPreviewed(false);
      onImported();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка импорта");
    } finally {
      setBusy(false);
    }
  };

  const exportURL = "/api/export.csv?" + new URLSearchParams({ from, to, accountId, category }).toString();
  const mappingReady = fields.every(field => !field.required || Boolean(mapping[field.key]));

  return (
    <Panel title="CSV: импорт и экспорт" subtitle="Переносите операции между Fluxora и таблицами">
      <div className="pure-grid two">
        <div>
          <strong>Экспорт операций</strong>
          <div className="pure-grid two" style={{ marginTop: 10 }}>
            <label className="form-label">С даты<input className="pure-input" type="date" value={from} max={to || undefined} onChange={e => setFrom(e.target.value)} /></label>
            <label className="form-label">По дату<input className="pure-input" type="date" value={to} min={from || undefined} onChange={e => setTo(e.target.value)} /></label>
            <label className="form-label">Счёт<select className="pure-select" value={accountId} onChange={e => setAccountId(e.target.value)}><option value="">Все счета</option>{accounts.map(account => <option key={account.id} value={account.id}>{account.name}</option>)}</select></label>
            <label className="form-label">Категория<input className="pure-input" value={category} onChange={e => setCategory(e.target.value)} placeholder="Все категории" /></label>
          </div>
          <a className="pure-button" href={exportURL} style={{ marginTop: 12, display: "inline-flex" }}><Download size={15} />Скачать CSV</a>
        </div>
        <div>
          <strong>Импорт операций</strong>
          <label className="pure-button secondary" style={{ marginTop: 10, cursor: "pointer", display: "inline-flex" }}>
            <FileUp size={15} />{busy ? "Читаем…" : "Выбрать CSV"}
            <input type="file" accept=".csv,text/csv" hidden disabled={busy} onChange={chooseFile} />
          </label>
          <p className="muted" style={{ fontSize: 11, marginTop: 8 }}>До 5 МБ. Поддерживается автоматическое распознавание CSV Tinkoff и ручное сопоставление остальных файлов.</p>
        </div>
      </div>

      {file && headers.length > 0 && (
        <div className="empty-note" style={{ marginTop: 14 }}>
          <div className="form-row" style={{ marginBottom: 10 }}><Columns3 size={16} /><strong>Сопоставление колонок</strong><small className="muted">{file.name}</small></div>
          {format === "tinkoff" && <p className="muted" style={{ fontSize: 12 }}>Формат Tinkoff распознан. Для суммы используется «Сумма в валюте счёта», то есть сумма в валюте счёта Fluxora. Строки со статусом «Ошибка» и флагом «Учёт в аналитике: Нет» пропускаются, поскольку Fluxora пока не хранит этот флаг. Тип определяется по знаку суммы. Переводы между своими счетами пока попадут как расход или доход: выписка не содержит ID второго счёта Fluxora.</p>}
          <div className="pure-grid two">
            {fields.map(field => (
              <label className="form-label" key={field.key}>
                {field.label}{field.required ? " *" : " (необязательно)"}
                <select className="pure-select" value={mapping[field.key] || ""} onChange={event => { setMapping(old => ({ ...old, [field.key]: event.target.value })); setPreviewed(false); }}>
                  <option value="">{field.required ? "Выберите значение" : "Не использовать"}</option>
                  {field.key === "type" && format === "tinkoff" && <option value={tinkoffTypeValue}>Определять по знаку суммы</option>}
                  {field.key === "accountId" && format === "tinkoff" && accounts.map(account => <option key={account.id} value={tinkoffAccountPrefix + account.id}>{account.name}</option>)}
                  {headers.map(header => <option key={header} value={header}>{header}</option>)}
                </select>
              </label>
            ))}
          </div>
          <button className="pure-button" disabled={busy || !mappingReady} style={{ marginTop: 12 }} onClick={() => void preview()}>{busy ? "Проверка…" : "Проверить файл"}</button>
        </div>
      )}

      {previewed && <div className="empty-note" style={{ marginTop: 10, display: "flex", gap: 12, alignItems: "center" }}><Check size={16} />Проверено строк: {rows.length}. Корректных: {valid}. С ошибками: {invalid}.{skipped > 0 ? ` Пропущено по статусу: ${skipped}.` : ""}</div>}
      {previewed && invalid === 0 && rows.length > 0 && <button className="pure-button" disabled={busy} style={{ marginTop: 10 }} onClick={() => void confirm()}><Check size={15} />Подтвердить импорт</button>}
      {previewed && rows.length > 0 && (
        <div className="transaction-list" style={{ marginTop: 14 }}>
          {rows.slice(0, 12).map((row, index) => (
            <div className="transaction-row" key={index}>
              <span className={`transaction-icon ${row.errors.length ? "" : "mint"}`}>{row.errors.length ? <AlertTriangle size={15} /> : <Check size={15} />}</span>
              <span className="transaction-copy"><strong>{row.title || "Без названия"} · {row.amount} ₽</strong><small>{row.date} · {row.type} · счёт #{row.accountId}</small>{row.errors.length > 0 && <small style={{ color: "#f87171" }}>{row.errors.join("; ")}</small>}</span>
            </div>
          ))}
        </div>
      )}
      {rows.length > 12 && <p className="muted" style={{ fontSize: 11 }}>Показаны первые 12 строк предварительного просмотра.</p>}
      {error && <p role="alert" style={{ color: "#f87171", fontSize: 12 }}>{error}</p>}
    </Panel>
  );
}
