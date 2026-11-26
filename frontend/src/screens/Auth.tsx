import { useState, type ChangeEvent } from "react";

function Auth({ onLogin }: { onLogin: () => void }) {
  const [isRegister, setIsRegister] = useState(false);
  // Данные пользователя
  const [formData, setFormData] = useState({
    firstName: "",
    lastName: "",
    username: "",
    email: "",
    password: "",
    confirmPassword: "",
  });

  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  // Обработка изменений полей
  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { name, value } = event.target;

    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  // Проверка паролей
  const validatePasswords = () => {
    if (formData.password.length < 8) {
      return "Пароль должен содержать минимум 8 символов";
    }
    if (formData.password !== formData.confirmPassword) {
      return "Пароли не совпадают";
    }
    return "";
  };

  // Обработка нажатия кнопки
  const handleSubmit = async () => {
    setError("");
    setSuccess("");

    if (!formData.username.trim() || !formData.password) {
      setError("Заполните все поля");
      return;
    }

    if (isRegister) {
      if (
        !formData.firstName.trim() ||
        !formData.lastName.trim() ||
        !formData.email.trim()
      ) {
        setError("Заполните все поля");
        return;
      }

      const passwordError = validatePasswords();
      if (passwordError) {
        setError(passwordError);
        return;
      }
    }

    try {
      const url = isRegister ? "/api/register" : "/api/login";

      // Отправляем запрос
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(
          isRegister
            ? {
                firstName: formData.firstName,
                lastName: formData.lastName,
                username: formData.username,
                email: formData.email,
                password: formData.password,
              }
            : {
                identifier: formData.username,
                password: formData.password,
              },
        ),
      });

      const data = await response.json();

      if (!response.ok) {
        setError(data.error || "Произошла ошибка");
        return;
      }

      setSuccess(data.message);

      if (!isRegister) {
        onLogin();
      }
    } catch {
      setError("Не удалось подключиться к серверу");
    }
  };
  return (
    <div className="flex min-h-screen items-center justify-center bg-[#100B1D]">
      {/* Карточка авторизации */}
      <div className="w-full max-w-md rounded-3xl border border-white/10 bg-[#1B1429] p-10">
        {/* Логотип и название */}
        <div className="mb-8 text-center">
          <h1 className="text-4xl font-bold text-white">Fluxora</h1>

          <p className="mt-3 text-gray-400">Добро пожаловать!</p>
        </div>

        {/* Форма входа */}
        <div className="flex flex-col gap-5">
          {isRegister && (
            <div>
              <label className="mb-2 block text-sm text-gray-300">Имя</label>
              <input
                type="text"
                name="firstName"
                value={formData.firstName}
                onChange={handleChange}
                placeholder="Введите ваше имя"
                className="w-full rounded-xl border border-white/10 bg-[#100B1D] px-4 py-3 text-white outline-none transition focus:border-[#853CF5]"
              />
            </div>
          )}

          {isRegister && (
            <div>
              <label className="mb-2 block text-sm text-gray-300">
                Фамилия
              </label>
              <input
                type="text"
                name="lastName"
                value={formData.lastName}
                onChange={handleChange}
                placeholder="Введите вашу фамилию"
                className="w-full rounded-xl border border-white/10 bg-[#100B1D] px-4 py-3 text-white outline-none transition focus:border-[#853CF5]"
              />
            </div>
          )}
          <div>
            <label className="mb-2 block text-sm text-gray-300">
              {isRegister ? "Логин" : "Логин или Email"}
            </label>
            <input
              type="text"
              name="username"
              value={formData.username}
              onChange={handleChange}
              placeholder={
                isRegister ? "Придумайте логин" : "Введите логин или email"
              }
              className="w-full rounded-xl border border-white/10 bg-[#100B1D] px-4 py-3 text-white outline-none transition focus:border-[#853CF5]"
            />
          </div>

          {isRegister && (
            <div>
              <label className="mb-2 block text-sm text-gray-300">Email</label>
              <input
                type="email"
                name="email"
                value={formData.email}
                onChange={handleChange}
                placeholder="Введите email"
                className="w-full rounded-xl border border-white/10 bg-[#100B1D] px-4 py-3 text-white outline-none transition focus:border-[#853CF5]"
              />
            </div>
          )}

          <div>
            <label className="mb-2 block text-sm text-gray-300">Пароль</label>
            <input
              type="password"
              name="password"
              value={formData.password}
              onChange={handleChange}
              placeholder="Введите пароль"
              className="w-full rounded-xl border border-white/10 bg-[#100B1D] px-4 py-3 text-white outline-none transition focus:border-[#853CF5]"
            />
          </div>

          {isRegister && (
            <div>
              <label className="mb-2 block text-sm text-gray-300">
                Повторите пароль
              </label>
              <input
                type="password"
                name="confirmPassword"
                value={formData.confirmPassword}
                onChange={handleChange}
                placeholder="Повторите пароль"
                className="w-full rounded-xl border border-white/10 bg-[#100B1D] px-4 py-3 text-white outline-none transition focus:border-[#853CF5]"
              />
            </div>
          )}

          <button
            onClick={handleSubmit}
            className="mt-2 rounded-xl bg-[#853CF5] py-3 font-medium text-white transition hover:bg-[#7130D6]"
          >
            {isRegister ? "Зарегистрироваться" : "Войти"}
          </button>
        </div>

        {error && (
          <p className="mt-4 text-center text-sm text-red-400">{error}</p>
        )}
        {success && (
          <p className="mt-4 text-center text-sm text-green-400">{success}</p>
        )}

        {/* Переключение на регистрацию */}
        <p className="mt-7 text-center text-sm text-gray-400">
          {isRegister ? "Уже есть аккаунт?" : "Нет аккаунта?"}{" "}
          <button
            onClick={() => {
              setIsRegister(!isRegister);
              setError("");
              setSuccess("");
            }}
            className="font-medium text-[#A875FF] hover:text-white"
          >
            {isRegister ? "Войти" : "Зарегистрироваться"}
          </button>
        </p>
      </div>
    </div>
  );
}
export default Auth;
