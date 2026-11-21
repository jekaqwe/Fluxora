import { useState } from "react";
// Экраны приложения
import Home from "./screens/Home";
import Operations from "./screens/Operations";
import Analytics from "./screens/Analytics";
import Goals from "./screens/Goals";
import Family from "./screens/Family";
import Profile from "./screens/Profile";
import Settings from "./screens/Settings";
// Изображения
import logo from "./assets/fluxora-logo.png";
import background from "./assets/fluxora-bg.png";

function App() {
  // Текущая открытая страница
  const [currentPage, setCurrentPage] = useState("home");
  // Переключение экранов
  let page;

  switch (currentPage) {
    case "home":
      page = <Home />;
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

    default:
      page = null;
  }

  return (
    // Основной контейнер приложения
    <div className="flex min-h-screen bg-cover bg-center bg-no-repeat" style={{ backgroundImage: `url(${background})` }}>
      {/* Боковое меню */}
      <aside className="flex min-h-screen w-64 flex-col">

        <img
          src={logo}
          alt="Fluxora"
          className="h-19 w-auto object-contain"
        />

        <button
          onClick={() => setCurrentPage("home")}
          className="mx-3 mt-8 flex items-center rounded-full px-4 py-3 text-left text-gray-400 transition-all duration-300 hover:translate-x-1 hover:bg-[#853CF5]/15 hover:text-gray-400">
          Главная
        </button>

        <button
          onClick={() => setCurrentPage("operations")}
          className="mx-3 mt-8 flex items-center rounded-full px-4 py-3 text-left text-gray-400 transition-all duration-300 hover:translate-x-1 hover:bg-[#853CF5]/15 hover:text-gray-400">
          Операции
        </button>

        <button
          onClick={() => setCurrentPage("analytics")}
          className="mx-3 mt-8 flex items-center rounded-full px-4 py-3 text-left text-gray-400 transition-all duration-300 hover:translate-x-1 hover:bg-[#853CF5]/15 hover:text-gray-400">
          Аналитика
        </button>

        <button
          onClick={() => setCurrentPage("goals")}
          className="mx-3 mt-8 flex items-center rounded-full px-4 py-3 text-left text-gray-400 transition-all duration-300 hover:translate-x-1 hover:bg-[#853CF5]/15 hover:text-gray-400">
          Цели
        </button>

        <button
          onClick={() => setCurrentPage("family")}
          className="mx-3 mt-8 flex items-center rounded-full px-4 py-3 text-left text-gray-400 transition-all duration-300 hover:translate-x-1 hover:bg-[#853CF5]/15 hover:text-gray-400">
          Семья
        </button>

        <button
          onClick={() => setCurrentPage("profile")}
          className="mx-3 mt-8 flex items-center rounded-full px-4 py-3 text-left text-gray-400 transition-all duration-300 hover:translate-x-1 hover:bg-[#853CF5]/15 hover:text-gray-400">
          Профиль
        </button>

        <button
          onClick={() => setCurrentPage("settings")}
          className="mx-3 mt-8 flex items-center rounded-full px-4 py-3 text-left text-gray-400 transition-all duration-300 hover:translate-x-1 hover:bg-[#853CF5]/15 hover:text-gray-400">
          Настройки
        </button>

        <button
          onClick={() => {}}
          className="mx-3 mt-8 flex items-center rounded-full px-4 py-3 text-left text-gray-400 transition-all duration-300 hover:translate-x-1 hover:bg-[#853CF5]/15 hover:text-gray-400">
          Выход
        </button>
      </aside>
      <main className="flex-1">
        {page}
      </main>
    </div>
  );
}
export default App;