import { useState } from "react";
import { UserRound } from "lucide-react";
// Изображения
import calendar from "../assets/calendar.png";

function Profile() {
  const [avatar, setAvatar] = useState<string | null>(null);
  const handleAvatarChange = (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const file = event.target.files?.[0];

    if (file) {
      setAvatar(URL.createObjectURL(file));
    }
  };
  return (
    <div className="grid grid-cols-[1fr_2fr] gap-6 p-8">

      {/* Левая карточка */}
      <div className="flex min-h-[820px] flex-col items-center rounded-3xl border border-white/10 bg-white/5 p-6">

        {/* Аватарка и информация пользователя */}
        <div className="flex flex-col items-center gap-4">

          <label className="relative flex h-72 w-72 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-white/20 bg-white/10 transition hover:border-[#853CF5]">

            {avatar ? (
              <img
                src={avatar}
                alt="Аватар"
                className="h-full w-full object-cover"
              />
            ) : (
              <UserRound
                size={40}
                className="text-white/70"
              />
            )}

            <input
              type="file"
              accept="image/*"
              className="hidden"
              onChange={handleAvatarChange}
            />

          </label>

          <div>
            <h2 className="text-xl font-semibold text-white">
              ФИО
            </h2>

            <p className="mt-1 text-sm text-white/50">
              логин
            </p>
          </div>

        </div>

        {/* Разделитель */}
        <div className="mt-8 h-px w-full bg-white/15" />

        {/* Дата регистрации */}
        <div className="mt-12 flex items-center gap-3.5">
          <img
            src={calendar}
            alt="Календарь"
            className="h-5 w-5 object-contain"
          />

          <p className="text-sm text-white/50">
            Дата регистрации:
          </p>
        </div>

        <p className="mt-6 flex items-center text-sm text-white/50">
          20.11.2026
        </p>

        {/* Карточка прогресса */}
        <div className="mt-18 h-60 w-full rounded-3xl border border-white/10 bg-white/5 p-5">

          <h3 className="text-xl font-semibold text-white">
            Ваш прогресс
          </h3>

          <p className="mt-5 text-5xl font-bold text-white">
            75%
          </p>

          {/* Полоса прогресса */}
          <div className="mt-4 h-8 w-full overflow-hidden rounded-full bg-white/10">
            <div className="h-full w-[75%] rounded-full bg-gradient-to-r from-[#853CF5] to-[#B56AFF]" />
          </div>

          {/* Оценка результата */}
          <div className="mt-8 flex items-center gap-5">
            <span className="text-3xl">⭐</span>

            <span className="text-lg font-medium text-white/70">
              Отличный результат!
            </span>
          </div>
        </div>
      </div>

      {/* Правая часть */}
      <div className="flex flex-col gap-4">

        {/* Верхняя карточка */}
        <div className="rounded-3xl border border-white/10 bg-white/5 p-6">

          <div className="flex flex-col gap-12">

            <div className="flex justify-between">
              <span>Логин</span>
              <button className="text-[#C4A1FF] underline">
                изменить
              </button>
            </div>

            <div className="flex justify-between">
              <span>Пароль</span>
              <button className="text-[#C4A1FF] underline">
                изменить
              </button>
            </div>

            <div className="flex justify-between">
              <span>Телефон</span>
              <button className="text-[#C4A1FF] underline">
                изменить
              </button>
            </div>

            <div className="flex justify-between">
              <span>Почта</span>
              <button className="text-[#C4A1FF] underline">
                изменить
              </button>
            </div>
          </div>
        </div>
        {/* Нижняя карточка */}
        <div className="min-h-[560px] rounded-3xl border border-white/10 bg-white/5">
        </div>
      </div>
    </div>
  );
}
export default Profile;