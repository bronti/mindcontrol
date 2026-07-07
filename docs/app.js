// All user-facing strings, grouped by language.
// To add a language, add a new entry with the same keys.
const translations = {
  en: {
    title: "Daily check-in",
    subtitle: "How was your day?",

    sec_sleep: "Sleep",
    q_bedtime: "Fell asleep at",
    q_wake: "Woke up at",
    sleep_duration_label: "Sleep duration",

    q_sleep_quality: "Sleep quality",
    q_dreams: "Dreams?",
    dream_none: "None", dream_dreams: "Dreams", dream_nightmares: "Nightmares",

    sec_scales: "How you felt (1–10)",
    q_state: "Overall state",
    q_anxiety: "Anxiety",
    q_irritability: "Irritability",
    q_libido: "Libido",
    q_drowsiness: "Drowsiness",
    q_appetite: "Appetite",
    q_energy: "Energy",
    q_ate_well: "Ate well",

    sec_today: "Today",
    q_menstruation: "Menstruation",
    q_sex: "Sex",
    q_masturbation: "Masturbation",
    q_headache: "Headache",
    yes: "Yes", no: "No",

    sec_meds: "Medications",
    meds_hint: "Tick what you took and enter the dose in mg.",
    med_lamotrigine: "Lamotrigine",
    med_olanzapine: "Olanzapine",
    med_fluoxetine: "Fluoxetine",
    med_trittico: "Trittico",
    med_grandaxin: "Grandaxin",
    mg: "mg",

    q_note: "Diary entry",
    note_placeholder: "anything you want to remember about today...",
    submit: "Save",
    browser_alert: "Sending only works inside Telegram.\nCollected answers:\n\n",
  },
  ru: {
    title: "Ежедневный чек-ин",
    subtitle: "Как прошёл день?",

    sec_sleep: "Сон",
    q_bedtime: "Уснул(а) в",
    q_wake: "Проснулся(лась) в",
    sleep_duration_label: "Продолжительность сна",

    q_sleep_quality: "Качество сна",
    q_dreams: "Сны?",
    dream_none: "Нет", dream_dreams: "Сны", dream_nightmares: "Кошмары",

    sec_scales: "Как ты себя чувствовал(а) (1–10)",
    q_state: "Общее состояние",
    q_anxiety: "Тревожность",
    q_irritability: "Раздражительность",
    q_libido: "Либидо",
    q_drowsiness: "Сонливость",
    q_appetite: "Аппетит",
    q_energy: "Энергия",
    q_ate_well: "Хорошо кушал(а)",

    sec_today: "Сегодня",
    q_menstruation: "Менструация",
    q_sex: "Секс",
    q_masturbation: "Мастурбация",
    q_headache: "Болела голова",
    yes: "Да", no: "Нет",

    sec_meds: "Лекарства",
    meds_hint: "Отметь принятое и укажи дозу в мг.",
    med_lamotrigine: "Ламотриджин",
    med_olanzapine: "Оланзапин",
    med_fluoxetine: "Флуоксетин",
    med_trittico: "Тритико",
    med_grandaxin: "Грандаксин",
    mg: "мг",

    q_note: "Дневниковая запись",
    note_placeholder: "что угодно, что хочешь запомнить о дне...",
    submit: "Сохранить",
    browser_alert: "Отправка работает только внутри Телеграма.\nСобранные ответы:\n\n",
  },
};

// Connect to Telegram (if the form is opened inside it)
const tg = window.Telegram && window.Telegram.WebApp;
if (tg) {
  tg.ready();   // tell Telegram the form has loaded
  tg.expand();  // expand to full height
}

// Choose the language: the Telegram user's language if we support it, otherwise English.
const userLanguage = tg && tg.initDataUnsafe && tg.initDataUnsafe.user && tg.initDataUnsafe.user.language_code;
const language = translations[userLanguage] ? userLanguage : "en";
const dict = translations[language];

// Fill in every element marked with data-translate / data-translate-placeholder
function applyTranslations() {
  document.querySelectorAll("[data-translate]").forEach((el) => {
    const key = el.getAttribute("data-translate");
    if (dict[key]) el.textContent = dict[key];
  });
  document.querySelectorAll("[data-translate-placeholder]").forEach((el) => {
    const key = el.getAttribute("data-translate-placeholder");
    if (dict[key]) el.placeholder = dict[key];
  });
}
applyTranslations();

const form = document.getElementById("form");

// ---- Sliders: show the current value next to each label ----
form.querySelectorAll('input[type="range"]').forEach((range) => {
  const out = range.closest(".slider").querySelector("output");
  const sync = () => { out.textContent = range.value; };
  range.addEventListener("input", sync);
  sync();
});

// ---- Medications: enable the dose field only when the box is ticked ----
form.querySelectorAll(".med").forEach((row) => {
  const check = row.querySelector('input[type="checkbox"]');
  const dose = row.querySelector('input[type="number"]');
  const sync = () => {
    dose.disabled = !check.checked;
    if (!check.checked) dose.value = "";
  };
  check.addEventListener("change", sync);
  sync();
});

// ---- Sleep clock: draw the arc between bedtime and wake, show the duration ----
const bedtime = form.bedtime;
const wake = form.wake;
const sleepArc = document.getElementById("sleepArc");
const sleepDuration = document.getElementById("sleepDuration");

// Turn "HH:MM" into minutes since midnight (null if empty/invalid).
function toMinutes(value) {
  if (!value) return null;
  const [h, m] = value.split(":").map(Number);
  if (Number.isNaN(h) || Number.isNaN(m)) return null;
  return h * 60 + m;
}

// A point on the clock: 0 min = top (midnight), time runs clockwise.
function pointOnClock(minutes, radius) {
  const angle = (minutes / 1440) * 2 * Math.PI - Math.PI / 2;
  return {
    x: 60 + radius * Math.cos(angle),
    y: 60 + radius * Math.sin(angle),
  };
}

// Sleep length in minutes (wraps past midnight); null if either time is missing.
function sleepMinutes() {
  const b = toMinutes(bedtime.value);
  const w = toMinutes(wake.value);
  if (b === null || w === null) return null;
  let d = w - b;
  if (d <= 0) d += 1440; // crossed midnight
  return d;
}

function updateSleep() {
  const minutes = sleepMinutes();
  if (minutes === null) {
    sleepArc.setAttribute("d", "");
    sleepDuration.textContent = "—";
    return;
  }
  const b = toMinutes(bedtime.value);
  const start = pointOnClock(b, 52);
  const end = pointOnClock(b + minutes, 52);
  const largeArc = minutes > 720 ? 1 : 0; // more than 12h → long way around
  sleepArc.setAttribute("d", `M ${start.x} ${start.y} A 52 52 0 ${largeArc} 1 ${end.x} ${end.y}`);

  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  sleepDuration.textContent = `${h}h ${String(m).padStart(2, "0")}m`;
}
bedtime.addEventListener("input", updateSleep);
wake.addEventListener("input", updateSleep);
updateSleep();

// ---- Collect every medication that is ticked, with its dose ----
function collectMedications() {
  const meds = [];
  form.querySelectorAll(".med").forEach((row) => {
    const check = row.querySelector('input[type="checkbox"]');
    if (check.checked) {
      const dose = row.querySelector('input[type="number"]').value;
      meds.push({ name: check.value, dose: dose });
    }
  });
  return meds;
}

// ---- Submit: gather everything and hand it to the bot ----
form.addEventListener("submit", (event) => {
  event.preventDefault();

  const minutes = sleepMinutes();
  const answers = {
    bedtime: bedtime.value,
    wake: wake.value,
    sleep_hours: minutes === null ? null : Math.round((minutes / 60) * 100) / 100,
    sleep_quality: Number(form.sleep_quality.value),
    dreams: form.dreams.value,
    state: Number(form.state.value),
    anxiety: Number(form.anxiety.value),
    irritability: Number(form.irritability.value),
    libido: Number(form.libido.value),
    drowsiness: Number(form.drowsiness.value),
    appetite: Number(form.appetite.value),
    energy: Number(form.energy.value),
    ate_well: Number(form.ate_well.value),
    menstruation: form.menstruation.value === "yes",
    sex: form.sex.value === "yes",
    masturbation: form.masturbation.value === "yes",
    headache: form.headache.value === "yes",
    medications: collectMedications(),
    note: form.note.value,
  };

  if (tg) {
    // Inside Telegram: send the data to the bot and close the form
    tg.sendData(JSON.stringify(answers));
  } else {
    // In a regular browser: just show what would be sent (to check the layout)
    alert(dict.browser_alert + JSON.stringify(answers, null, 2));
  }
});
