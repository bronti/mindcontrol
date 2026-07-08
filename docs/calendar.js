// Calendar view: shows which days have entries. Each day is split top/bottom —
// top = sleep (coloured by "how rested"), bottom = day (coloured by overall
// state). Grey halves mean that part isn't filled; a neutral colour means the
// part was filled but that rating wasn't set.

const tg = window.Telegram && window.Telegram.WebApp;
if (tg) {
  tg.ready();
  tg.expand();
}

const ru =
  tg && tg.initDataUnsafe && tg.initDataUnsafe.user && tg.initDataUnsafe.user.language_code === "ru";

// Tapping a day's half asks the bot to open that entry (it replies with a
// pre-filled edit button). Closes the calendar.
function sendEdit(date, part) {
  if (tg) tg.sendData(JSON.stringify({ t: "edit", date: date, part: part }));
}

const text = {
  weekdays: ru ? ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"] : ["Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"],
  months: ru
    ? ["Январь", "Февраль", "Март", "Апрель", "Май", "Июнь", "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"]
    : ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"],
  parts: ru ? "верх = сон · низ = день" : "top = sleep · bottom = day",
  scale: ru ? "хуже → лучше" : "low → high",
};

// ---- Parse the ?days= data into a map: "YYYY-MM-DD" -> { top, bottom } ----
const data = {};
(new URLSearchParams(location.search).get("days") || "")
  .split("_")
  .filter(Boolean)
  .forEach((entry) => {
    const [d, top, bottom] = entry.split(".");
    if (!d || d.length !== 8) return;
    const iso = `${d.slice(0, 4)}-${d.slice(4, 6)}-${d.slice(6, 8)}`;
    data[iso] = { top: top, bottom: bottom };
  });

// ---- Colour for one half: grey if not filled, neutral if filled-but-unrated,
// otherwise a red→green hue by how high the rating is. ----
function halfColor(token, maxValue) {
  if (!token || token === "-") return "var(--cal-missing)";
  if (token === "f") return "var(--cal-unrated)";
  const value = Number(token);
  const norm = Math.max(0, Math.min(1, value / maxValue));
  const hue = Math.round(norm * 130); // 0 = red (low), 130 = green (high)
  return `hsl(${hue}, 62%, 45%)`;
}

// ---- Render one month ----
const grid = document.getElementById("grid");
const monthLabel = document.getElementById("monthLabel");

const weekdaysEl = document.getElementById("weekdays");
text.weekdays.forEach((w) => {
  const s = document.createElement("span");
  s.textContent = w;
  weekdaysEl.appendChild(s);
});
document.getElementById("legendParts").textContent = text.parts;
document.getElementById("legendScale").textContent = text.scale;

const now = new Date();
let year = now.getFullYear();
let month = now.getMonth(); // 0-11

function iso(y, m, d) {
  return `${y}-${String(m + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

function render() {
  monthLabel.textContent = `${text.months[month]} ${year}`;
  grid.innerHTML = "";

  const firstDow = (new Date(year, month, 1).getDay() + 6) % 7; // Monday = 0
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const todayIso = iso(now.getFullYear(), now.getMonth(), now.getDate());

  for (let i = 0; i < firstDow; i++) {
    const blank = document.createElement("div");
    blank.className = "cal-blank";
    grid.appendChild(blank);
  }

  for (let d = 1; d <= daysInMonth; d++) {
    const dayIso = iso(year, month, d);
    const rec = data[dayIso];

    const cell = document.createElement("div");
    cell.className = "cal-day" + (rec ? " filled" : "") + (dayIso === todayIso ? " today" : "");

    const top = document.createElement("div");
    top.className = "cal-half cal-top";
    const bottom = document.createElement("div");
    bottom.className = "cal-half cal-bot";
    if (rec) {
      top.style.background = halfColor(rec.top, 4);
      bottom.style.background = halfColor(rec.bottom, 10);
    }
    // Tap a half to view/edit that part of the day (top = sleep, bottom = day).
    top.addEventListener("click", () => sendEdit(dayIso, "sleep"));
    bottom.addEventListener("click", () => sendEdit(dayIso, "day"));

    const num = document.createElement("span");
    num.className = "cal-num";
    num.textContent = d;

    cell.append(top, bottom, num);
    grid.appendChild(cell);
  }
}

document.getElementById("prev").addEventListener("click", () => {
  month--;
  if (month < 0) {
    month = 11;
    year--;
  }
  render();
});
document.getElementById("next").addEventListener("click", () => {
  month++;
  if (month > 11) {
    month = 0;
    year++;
  }
  render();
});

render();
