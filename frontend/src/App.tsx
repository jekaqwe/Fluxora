import { useEffect, useState } from "react";
import {
  Home,
  ArrowLeftRight,
  ChartNoAxesCombined,
  Target,
  Users,
  UserRound,
  Settings as SettingsIcon,
  LogOut,
  Menu,
  X,
} from "lucide-react";
import HomeScreen from "./screens/Home";
import Operations from "./screens/Operations";
import Analytics from "./screens/Analytics";
import Goals from "./screens/Goals";
import Family from "./screens/Family";
import Profile from "./screens/Profile";
import Settings from "./screens/Settings";
import Auth from "./screens/Auth";
import logo from "./assets/fluxora-logo.png";
import background from "./assets/fluxora-bg.png";
import "./App.css";

type PageKey =
  | "home"
  | "operations"
  | "analytics"
  | "goals"
  | "family"
  | "profile"
  | "settings";

const navigation = [
  { id: "home", label: "Главная", icon: Home },
  { id: "operations", label: "Операции", icon: ArrowLeftRight },
  { id: "analytics", label: "Аналитика", icon: ChartNoAxesCombined },
  { id: "goals", label: "Цели", icon: Target },
  { id: "family", label: "Семья", icon: Users },
  { id: "profile", label: "Профиль", icon: UserRound },
  { id: "settings", label: "Настройки", icon: SettingsIcon },
] as const;

function App() {
  const [currentPage, setCurrentPage] = useState<PageKey>(() => {
    const savedPage = localStorage.getItem("fluxora_current_page");
    return navigation.some((item) => item.id === savedPage)
      ? (savedPage as PageKey)
      : "home";
  });
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  useEffect(() => {
    localStorage.setItem("fluxora_current_page", currentPage);
  }, [currentPage]);

  useEffect(() => {
    fetch("/api/me")
      .then((response) => {
        if (!response.ok) throw new Error("Пользователь не авторизован");
        return response.json();
      })
      .then(() => setIsAuthenticated(true))
      .catch(() => setIsAuthenticated(false))
      .finally(() => setIsCheckingAuth(false));
  }, []);

  const handleLogout = async () => {
    try {
      const response = await fetch("/api/logout", { method: "POST" });
      if (!response.ok) throw new Error("Ошибка при выходе из аккаунта");
      localStorage.setItem("fluxora_current_page", "home");
      setCurrentPage("home");
      setIsAuthenticated(false);
      setIsMobileMenuOpen(false);
    } catch (error) {
      console.error("Ошибка выхода:", error);
      alert("Не удалось выйти из аккаунта. Попробуйте ещё раз.");
    }
  };

  const selectPage = (page: PageKey) => {
    setCurrentPage(page);
    setIsMobileMenuOpen(false);
  };

  if (isCheckingAuth) return null;
  if (!isAuthenticated) {
    return <Auth onLogin={() => setIsAuthenticated(true)} />;
  }

  let page;
  switch (currentPage) {
    case "home":
      page = <HomeScreen />;
      break;
    case "operations":
      page = <Operations />;
      break;
    case "analytics":
      page = <Analytics />;
      break;
    case "goals":
      page = <Goals />;
      break;
    case "family":
      page = <Family />;
      break;
    case "profile":
      page = <Profile />;
      break;
    case "settings":
      page = <Settings />;
      break;
  }

  return (
    <div
      className="app-shell"
      style={{ backgroundImage: `linear-gradient(115deg, rgba(13, 10, 27, .94), rgba(17, 12, 35, .88)), url(${background})` }}
    >
      <button
        className="mobile-menu-toggle"
        type="button"
        aria-label={isMobileMenuOpen ? "Закрыть меню" : "Открыть меню"}
        aria-expanded={isMobileMenuOpen}
        onClick={() => setIsMobileMenuOpen((open) => !open)}
      >
        {isMobileMenuOpen ? <X size={22} /> : <Menu size={22} />}
      </button>

      {isMobileMenuOpen && (
        <button
          className="sidebar-backdrop"
          type="button"
          aria-label="Закрыть меню"
          onClick={() => setIsMobileMenuOpen(false)}
        />
      )}

      <aside className={`app-sidebar ${isMobileMenuOpen ? "app-sidebar--open" : ""}`}>
        <div className="sidebar-brand">
          <img src={logo} alt="Fluxora" />
          <span>Fluxora</span>
        </div>

        <div className="sidebar-caption">МЕНЮ</div>
        <nav className="sidebar-nav" aria-label="Основная навигация">
          {navigation.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              type="button"
              className={`sidebar-link ${currentPage === id ? "sidebar-link--active" : ""}`}
              aria-current={currentPage === id ? "page" : undefined}
              onClick={() => selectPage(id)}
            >
              <Icon size={19} strokeWidth={1.8} />
              <span>{label}</span>
              {currentPage === id && <span className="sidebar-active-dot" />}
            </button>
          ))}
        </nav>

        <div className="sidebar-bottom">
          <div className="sidebar-divider" />
          <button type="button" className="sidebar-link sidebar-logout" onClick={handleLogout}>
            <LogOut size={19} strokeWidth={1.8} />
            <span>Выход</span>
          </button>
          <div className="sidebar-version">FLUXORA <span>•</span> PERSONAL SPACE</div>
        </div>
      </aside>

      <main className="app-main">
        <div className="mobile-top-spacer" />
        {page}
      </main>
    </div>
  );
}

export default App;
