import { useEffect, useState, type ChangeEvent } from "react";
import { UserRound, Trophy, Star, Crown, ShieldCheck, TrendingUp, CalendarDays, Camera, Trash2 } from "lucide-react";
import { PageHeading, Panel, Progress } from "./ui";
import "./Pure.css";

type UserProfile = {
  id: number;
  firstName: string;
  lastName: string;
  username: string;
  email: string;
  avatarUrl: string | null;
};

function Profile() {
  const [user, setUser] = useState<UserProfile | null>(null);
  const [avatarBusy, setAvatarBusy] = useState(false);
  const [avatarError, setAvatarError] = useState("");

  const loadProfile = async () => {
    const response = await fetch("/api/profile");
    if (!response.ok) throw new Error("Не удалось загрузить профиль");
    setUser(await response.json());
  };

  useEffect(() => {
    loadProfile().catch((error) => console.error("Ошибка загрузки профиля:", error));
  }, []);

  const handleAvatarChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    setAvatarError("");
    setAvatarBusy(true);
    try {
      const formData = new FormData();
      formData.append("avatar", file);
      const response = await fetch("/api/profile/avatar", { method: "PUT", body: formData });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось загрузить аватар");
      setUser((current) => current ? { ...current, avatarUrl: data.avatarUrl } : current);
    } catch (error) {
      setAvatarError(error instanceof Error ? error.message : "Ошибка загрузки аватара");
    } finally {
      setAvatarBusy(false);
    }
  };

  const handleAvatarDelete = async () => {
    setAvatarError("");
    setAvatarBusy(true);
    try {
      const response = await fetch("/api/profile/avatar", { method: "DELETE" });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Не удалось удалить аватар");
      setUser((current) => current ? { ...current, avatarUrl: null } : current);
    } catch (error) {
      setAvatarError(error instanceof Error ? error.message : "Ошибка удаления аватара");
    } finally {
      setAvatarBusy(false);
    }
  };

  const fullName = user ? `${user.firstName} ${user.lastName}` : "Загрузка профиля";

  return (
    <div className="pure-page">
      <PageHeading title="Профиль" subtitle="Ваш путь, достижения и активность" />
      <Panel className="profile-hero">
        <div className="profile-avatar">
          {user?.avatarUrl
            ? <img src={user.avatarUrl} alt="Аватар" style={{ width: "100%", height: "100%", objectFit: "cover", borderRadius: "50%" }} />
            : <UserRound />}
        </div>
        <div className="profile-name" style={{ flex: 1 }}>
          <h2>{fullName}</h2>
          <p>@{user?.username ?? "username"} · Уровень 8 · Финансовый стратег</p>
          <div style={{ maxWidth: 280, marginTop: 13 }}>
            <Progress value={75} />
            <small className="muted">750 / 1 000 XP до следующего уровня</small>
          </div>
        </div>
        <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
          <label className="pure-button secondary" style={{ cursor: avatarBusy ? "wait" : "pointer", opacity: avatarBusy ? 0.6 : 1 }}>
            <Camera size={15} />{avatarBusy ? "Загрузка…" : "Изменить фото"}
            <input type="file" accept="image/jpeg,image/png,image/webp,image/gif" hidden disabled={avatarBusy} onChange={handleAvatarChange} />
          </label>
          {user?.avatarUrl && <button type="button" className="pure-button secondary" disabled={avatarBusy} onClick={handleAvatarDelete}><Trash2 size={15} />Удалить фото</button>}
        </div>
      </Panel>
      {avatarError && <p role="alert" style={{ color: "#f87171", fontSize: 12, marginTop: 8 }}>{avatarError}</p>}

      <div className="pure-grid three" style={{ marginTop: 16 }}>
        <div className="stat-card"><span className="muted" style={{ fontSize: 11 }}>Операции</span><strong>247</strong></div>
        <div className="stat-card"><span className="muted" style={{ fontSize: 11 }}>Достижения</span><strong>12</strong></div>
        <div className="stat-card"><span className="muted" style={{ fontSize: 11 }}>Друзья</span><strong>3</strong></div>
      </div>

      <div className="pure-grid two" style={{ marginTop: 16 }}>
        <Panel title="Достижения" action={<span className="muted" style={{ fontSize: 10 }}>12 наград</span>}>
          <div className="badge-grid">
            {[{ name: "Первый шаг", Icon: Trophy }, { name: "Активность", Icon: Star }, { name: "Лидер", Icon: Crown }, { name: "Надёжность", Icon: ShieldCheck }].map(({ name, Icon }) => (
              <div className="badge-item" key={name}><Icon /><strong>{name}</strong></div>
            ))}
          </div>
        </Panel>
        <Panel title="Активность за месяц" subtitle="Динамика активности">
          <div style={{ display: "flex", alignItems: "center", gap: 18 }}>
            <strong className="positive" style={{ fontSize: 30 }}>+18%</strong>
            <div style={{ flex: 1 }}>
              <div className="bar-chart" style={{ height: 115 }}>
                {[30, 42, 36, 56, 50, 75, 88].map((value, index) => (
                  <div className="bar-column" key={index}>
                    <div className="bar-track" style={{ height: 90 }}><span style={{ height: `${value}%` }} /></div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </Panel>
      </div>

      <Panel title="Последняя активность">
        <div className="member-row"><div className="transaction-icon"><CalendarDays size={16} /></div><div className="member-copy"><strong>Вы добавили новую операцию</strong><small>Сегодня · Финансовая активность</small></div></div>
        <div className="member-row"><div className="transaction-icon mint"><Trophy size={16} /></div><div className="member-copy"><strong>Получено достижение «Активность»</strong><small>Вчера · Достижения</small></div></div>
        <div className="member-row"><div className="transaction-icon"><TrendingUp size={16} /></div><div className="member-copy"><strong>Прогресс профиля вырос</strong><small>На этой неделе · Прогресс</small></div></div>
      </Panel>
      <p className="muted" style={{ fontSize: 10, marginTop: 12 }}>Финансовая статистика, уровни и достижения будут подключены к реальным данным на следующих этапах.</p>
    </div>
  );
}

export default Profile;
