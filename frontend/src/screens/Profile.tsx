import { useEffect, useState } from "react";
import {
  CalendarDays,
  Camera,
  ChevronUp,
  Crown,
  Gem,
  LockKeyhole,
  Mail,
  Phone,
  Star,
  Trophy,
  UserRound,
  ArrowUpRight,
} from "lucide-react";
import "./Profile.css";

type UserProfile = {
  id: number;
  firstName: string;
  lastName: string;
  username: string;
  email: string;
};

function Profile() {
  const [user, setUser] = useState<UserProfile | null>(null);
  const [avatar, setAvatar] = useState<string | null>(null);

  useEffect(() => {
    fetch("/api/me")
      .then((response) => {
        if (!response.ok) throw new Error("Не удалось загрузить профиль");
        return response.json();
      })
      .then((data: UserProfile) => setUser(data))
      .catch((error) => console.error("Ошибка загрузки профиля:", error));
  }, []);

  useEffect(() => {
    return () => {
      if (avatar) URL.revokeObjectURL(avatar);
    };
  }, [avatar]);

  const handleAvatarChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) setAvatar(URL.createObjectURL(file));
  };

  const fullName = user ? `${user.firstName} ${user.lastName}` : "Загрузка профиля";
  const initials = user ? `${user.firstName[0] ?? ""}${user.lastName[0] ?? ""}` : "F";

  const accountRows = [
    { label: "Логин", value: user?.username ?? "Загрузка…", icon: UserRound },
    { label: "Пароль", value: "••••••••••••", icon: LockKeyhole },
    { label: "Телефон", value: "Не указан", icon: Phone },
    { label: "Почта", value: user?.email ?? "Загрузка…", icon: Mail },
  ];

  return (
    <div className="profile-page">
      <header className="profile-page-header">
        <div>
          <div className="profile-eyebrow">ЛИЧНЫЙ КАБИНЕТ</div>
          <h1>Мой профиль</h1>
          <p>Управляйте своей учётной записью и следите за прогрессом</p>
        </div>
        <div className="profile-status"><span /> Аккаунт активен</div>
      </header>

      <div className="profile-layout">
        <section className="profile-card glass-card" aria-labelledby="profile-name">
          <div className="profile-card-glow" />
          <div className="profile-avatar-wrap">
            <div className="profile-avatar-ring">
              {avatar ? (
                <img src={avatar} alt="Аватар пользователя" className="profile-avatar-image" />
              ) : (
                <div className="profile-avatar-placeholder">
                  <UserRound size={58} strokeWidth={1.2} />
                  <span>{initials}</span>
                </div>
              )}
            </div>
            <label className="profile-avatar-edit" title="Изменить аватар">
              <Camera size={16} />
              <input type="file" accept="image/*" onChange={handleAvatarChange} />
            </label>
          </div>

          <div className="profile-identity">
            <h2 id="profile-name">{fullName}</h2>
            <span className="profile-username">@{user?.username ?? "username"}</span>
            <div className="profile-member-since">
              <CalendarDays size={15} />
              <span>Участник Fluxora</span>
            </div>
          </div>

          <div className="profile-card-separator" />

          <div className="progress-card">
            <div className="progress-heading">
              <div>
                <span className="progress-kicker">ВАШ ПРОГРЕСС</span>
                <h3>Отличный темп!</h3>
              </div>
              <div className="progress-trend"><ChevronUp size={17} /> +12%</div>
            </div>
            <div className="progress-number">75<span>%</span></div>
            <div
              className="progress-track"
              role="progressbar"
              aria-label="Прогресс профиля"
              aria-valuenow={75}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <div className="progress-fill" />
            </div>
            <div className="progress-foot">
              <span><Star size={15} fill="currentColor" /> Отличный результат!</span>
              <span>75 из 100</span>
            </div>
          </div>

          <div className="profile-card-footnote">
            <span className="profile-footnote-icon"><Gem size={16} /></span>
            <span>Каждый шаг приближает к вашим целям</span>
          </div>
        </section>

        <div className="profile-side-column">
          <section className="account-card glass-card" aria-labelledby="account-title">
            <div className="section-heading">
              <div>
                <span className="section-eyebrow">БЕЗОПАСНОСТЬ И ДОСТУП</span>
                <h2 id="account-title">Настройки аккаунта</h2>
              </div>
              <div className="section-heading-icon"><LockKeyhole size={19} /></div>
            </div>

            <div className="account-list">
              {accountRows.map(({ label, value, icon: Icon }) => (
                <div className="account-row" key={label}>
                  <div className="account-row-icon"><Icon size={18} /></div>
                  <div className="account-row-copy">
                    <span>{label}</span>
                    <strong>{value}</strong>
                  </div>
                  <span className="account-edit-label">Изменить <ArrowUpRight size={14} /></span>
                </div>
              ))}
            </div>
            <p className="account-note">Изменение личных данных станет доступно в следующих обновлениях Fluxora.</p>
          </section>

          <section className="achievements-card glass-card" aria-labelledby="achievements-title">
            <div className="section-heading achievements-heading">
              <div>
                <span className="section-eyebrow">ВАШИ УСПЕХИ</span>
                <h2 id="achievements-title">Достижения</h2>
              </div>
              <span className="achievements-count">4 награды</span>
            </div>
            <div className="achievement-list">
              <div className="achievement-item achievement-purple">
                <div className="achievement-hex"><Trophy size={23} /></div>
                <span>Первый шаг</span>
              </div>
              <div className="achievement-item achievement-blue">
                <div className="achievement-hex"><Star size={23} /></div>
                <span>Активность</span>
              </div>
              <div className="achievement-item achievement-gold">
                <div className="achievement-hex"><Crown size={23} /></div>
                <span>Лидер</span>
              </div>
              <div className="achievement-item achievement-cyan">
                <div className="achievement-hex"><Gem size={23} /></div>
                <span>Эксперт</span>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

export default Profile;
