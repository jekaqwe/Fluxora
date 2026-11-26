import { useEffect, useState } from "react";
import { PageHeading, Panel } from "./ui";
import { Bell, Check, CheckCheck } from "lucide-react";
import "./Pure.css";

type Notification = {
  id: number;
  type: string;
  title: string;
  message: string;
  readAt?: string;
  createdAt: string;
};
export default function Notifications() {
  const [items, setItems] = useState<Notification[]>([]);
  const [unread, setUnread] = useState(0);
  const [error, setError] = useState("");
  const load = async () => {
    try {
      const r = await fetch("/api/notifications");
      const d = await r.json();
      if (!r.ok) throw new Error(d.error || "Не удалось загрузить уведомления");
      setItems(d.items || []);
      setUnread(d.unread || 0);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка уведомлений");
    }
  };
  useEffect(() => {
    void load();
  }, []);
  const read = async (id: number) => {
    try {
      const r = await fetch(`/api/notifications/${id}`, { method: "PATCH" });
      const d = await r.json();
      if (!r.ok) throw new Error(d.error || "Не удалось отметить уведомление");
      setItems((old) =>
        old.map((x) =>
          x.id === id ? { ...x, readAt: new Date().toISOString() } : x,
        ),
      );
      setUnread((n) => Math.max(0, n - 1));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка");
    }
  };
  const readAll = async () => {
    try {
      const r = await fetch("/api/notifications/read-all", { method: "POST" });
      const d = await r.json();
      if (!r.ok) throw new Error(d.error || "Не удалось отметить уведомления");
      setItems((old) =>
        old.map((x) => ({
          ...x,
          readAt: x.readAt || new Date().toISOString(),
        })),
      );
      setUnread(0);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Ошибка");
    }
  };
  return (
    <div className="pure-page">
      <PageHeading
        title="Уведомления"
        subtitle={`Непрочитанных: ${unread}`}
        action={
          <button
            className="pure-button"
            disabled={!unread}
            onClick={() => void readAll()}
          >
            <CheckCheck size={15} />
            Прочитать все
          </button>
        }
      />
      <Panel title="Центр уведомлений">
        {items.length === 0 ? (
          <div className="empty-note">Пока уведомлений нет.</div>
        ) : (
          <div className="transaction-list">
            {items.map((item) => (
              <div
                className="transaction-row"
                key={item.id}
                style={{ opacity: item.readAt ? 0.6 : 1 }}
              >
                <span
                  className={`transaction-icon ${item.readAt ? "" : "mint"}`}
                >
                  <Bell size={16} />
                </span>
                <span className="transaction-copy">
                  <strong>{item.title}</strong>
                  <small>
                    {item.message} ·{" "}
                    {new Date(item.createdAt).toLocaleString("ru-RU")}
                  </small>
                </span>
                {!item.readAt && (
                  <button
                    className="pure-button ghost"
                    aria-label="Прочитать"
                    onClick={() => void read(item.id)}
                  >
                    <Check size={14} />
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </Panel>
      {error && (
        <p role="alert" style={{ color: "#f87171", fontSize: 12 }}>
          {error}
        </p>
      )}
    </div>
  );
}
