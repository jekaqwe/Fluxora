import { useEffect, useState } from "react";
import { Home, ArrowLeftRight, ChartNoAxesCombined, Target, Users, UserRound, Settings as SettingsIcon, LogOut, Menu, X, WalletCards, HandCoins, Bell } from "lucide-react";
import HomeScreen from "./screens/Home";
import Debts from "./screens/Debts";
import Notifications from "./screens/Notifications";
import Operations from "./screens/Operations";
import Accounts from "./screens/Accounts";
import Analytics from "./screens/Analytics";
import Goals from "./screens/Goals";
import Family from "./screens/Family";
import Profile from "./screens/Profile";
import Settings from "./screens/Settings";
import Auth from "./screens/Auth";
import logo from "./assets/fluxora-logo.png";
import "./App.css";

type PageKey = "home" | "operations" | "accounts" | "analytics" | "goals" | "debts" | "notifications" | "family" | "profile" | "settings";
const navigation = [
  { id: "home", label: "Главная", icon: Home },
  { id: "operations", label: "Операции", icon: ArrowLeftRight },
  { id: "accounts", label: "Счета", icon: WalletCards },
  { id: "analytics", label: "Аналитика", icon: ChartNoAxesCombined },
  { id: "goals", label: "Цели", icon: Target },
  { id: "debts", label: "Долги", icon: HandCoins },
  { id: "notifications", label: "Уведомления", icon: Bell },
  { id: "family", label: "Семья", icon: Users },
  { id: "profile", label: "Профиль", icon: UserRound },
  { id: "settings", label: "Настройки", icon: SettingsIcon },
] as const;

function App() {
  const [currentPage, setCurrentPage] = useState<PageKey>(() => {
    const saved = localStorage.getItem("fluxora_current_page");
    return navigation.some((item) => item.id === saved) ? saved as PageKey : "home";
  });
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  useEffect(() => { localStorage.setItem("fluxora_current_page", currentPage); }, [currentPage]);
  useEffect(() => {
    if (!isAuthenticated) return;
    fetch("/api/settings/preferences")
      .then((response) => response.ok ? response.json() : null)
      .then((preferences) => {
        if (!preferences?.theme) return;
        document.documentElement.dataset.theme = preferences.theme === "auto" ? (window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light") : preferences.theme;
        localStorage.setItem("fluxora_theme", preferences.theme);
      })
      .catch(() => {});
  }, [isAuthenticated]);
  useEffect(() => {
    fetch("/api/me")
      .then((response) => { if (!response.ok) throw new Error("Unauthorized"); return response.json(); })
      .then(() => setIsAuthenticated(true))
      .catch(() => setIsAuthenticated(false))
      .finally(() => setIsCheckingAuth(false));
  }, []);

  const handleLogout = async () => {
    try {
      const response = await fetch("/api/logout", { method: "POST" });
      if (!response.ok) throw new Error("Logout failed");
      localStorage.setItem("fluxora_current_page", "home");
      setCurrentPage("home");
      setIsAuthenticated(false);
      setIsMobileMenuOpen(false);
    } catch (error) {
      console.error("Ошибка выхода:", error);
      alert("Не удалось выйти из аккаунта. Попробуйте ещё раз.");
    }
  };
  const selectPage = (page: PageKey) => { setCurrentPage(page); setIsMobileMenuOpen(false); };

  if (isCheckingAuth) return <div className="auth-loading"><span className="loading-mark" /></div>;
  if (!isAuthenticated) return <Auth onLogin={() => setIsAuthenticated(true)} />;

  let page;
  switch (currentPage) {
    case "home": page = <HomeScreen />; break;
    case "operations": page = <Operations />; break;
    case "accounts": page = <Accounts />; break;
    case "analytics": page = <Analytics />; break;
    case "goals": page = <Goals />; break;
    case "debts": page = <Debts />; break;
    case "notifications": page = <Notifications />; break;
    case "family": page = <Family />; break;
    case "profile": page = <Profile />; break;
    case "settings": page = <Settings />; break;
  }

  return (
    <div className="app-shell">
      <button className="mobile-menu-toggle" type="button" aria-label={isMobileMenuOpen ? "Закрыть меню" : "Открыть меню"} aria-expanded={isMobileMenuOpen} onClick={() => setIsMobileMenuOpen((open) => !open)}>
        {isMobileMenuOpen ? <X size={21} /> : <Menu size={21} />}
      </button>
      {isMobileMenuOpen && <button className="sidebar-backdrop" type="button" aria-label="Закрыть меню" onClick={() => setIsMobileMenuOpen(false)} />}
      <aside className={`app-sidebar ${isMobileMenuOpen ? "app-sidebar--open" : ""}`}>
        <div className="sidebar-brand"><img src={logo} alt="Fluxora" /><span>Fluxora</span></div>
        <nav className="sidebar-nav" aria-label="Основная навигация">
          {navigation.map(({ id, label, icon: Icon }) => (
            <button key={id} type="button" title={label} aria-label={label} aria-current={currentPage === id ? "page" : undefined} className={`sidebar-link ${currentPage === id ? "sidebar-link--active" : ""}`} onClick={() => selectPage(id)}>
              <Icon size={19} strokeWidth={1.9} /><span>{label}</span>
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <button type="button" className="sidebar-link sidebar-logout" title="Выйти" aria-label="Выйти" onClick={handleLogout}><LogOut size={19} /><span>Выйти</span></button>
          <div className="sidebar-version">FLUXORA <span>•</span> PURE</div>
        </div>
      </aside>
      <main className="app-main"><div className="mobile-top-spacer" />{page}</main>
    </div>
  );
}
export default App;
