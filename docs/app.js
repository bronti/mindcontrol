// The daily form: one page serving two modes (?form=sleep|day) and an edit mode
// (?mode=…). Wrapped in an IIFE so nothing leaks to the global scope.
(function () {
  "use strict";

// All user-facing strings, grouped by language (same keys per language).
const translations = {
  en: {
    title: "Daily check-in",
    subtitle: "How was your day?",
    title_sleep: "Sleep 🌙",
    subtitle_sleep: "How did you sleep?",
    title_day: "Your day ☀️",
    subtitle_day: "How was your day?",

    q_date: "Date",
    date_taken: "This date is already filled — pick another.",

    sec_sleep: "Sleep",
    q_bedtime: "Fell asleep at",
    q_wake: "Woke up at",
    sleep_duration_label: "Sleep duration",

    q_rested: "How rested",
    q_dreams: "Dreams?",
    dream_none: "None", dream_dreams: "Dreams", dream_nightmares: "Nightmares", dream_anxious: "Anxious",
    q_dream_note: "What was the dream about?",
    dream_note_placeholder: "what happened in the dream...",

    sec_scales: "How you felt (0–4)",
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
    q_smoking: "Smoking",
    yes: "Yes", no: "No",

    sec_meds: "Medications",
    sec_meds_sleep: "Medications (sleep)",
    meds_hint: "Pick what you took; the dose in mg is pre-filled and editable.",
    med_add: "+ Add a medication…",
    med_other: "Other…",
    med_name_placeholder: "Medication name",
    mg: "mg",

    q_note: "Diary entry",
    note_placeholder: "anything you want to remember about today...",
    submit: "Save",
    submit_update: "Update entry",
    submit_create: "Create entry",
    browser_alert: "Sending only works inside Telegram.\nCollected answers:\n\n",
  },
  ru: {
    title: "Ежедневный чек-ин",
    subtitle: "Как прошёл день?",
    title_sleep: "Сон 🌙",
    subtitle_sleep: "Как ты спал(а)?",
    title_day: "Твой день ☀️",
    subtitle_day: "Как прошёл день?",

    q_date: "Дата",
    date_taken: "Эта дата уже заполнена — выбери другую.",

    sec_sleep: "Сон",
    q_bedtime: "Уснул(а) в",
    q_wake: "Проснулся(лась) в",
    sleep_duration_label: "Продолжительность сна",

    q_rested: "Насколько выспался(ась)",
    q_dreams: "Сны?",
    dream_none: "Нет", dream_dreams: "Сны", dream_nightmares: "Кошмары", dream_anxious: "Тревожные",
    q_dream_note: "О чём был сон?",
    dream_note_placeholder: "что происходило во сне...",

    sec_scales: "Как ты себя чувствовал(а) (0–4)",
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
    q_smoking: "Курение",
    yes: "Да", no: "Нет",

    sec_meds: "Лекарства",
    sec_meds_sleep: "Лекарства (сон)",
    meds_hint: "Выбери принятое; доза в мг подставится сама, её можно поправить.",
    med_add: "+ Добавить лекарство…",
    med_other: "Другое…",
    med_name_placeholder: "Название лекарства",
    mg: "мг",

    q_note: "Дневниковая запись",
    note_placeholder: "что угодно, что хочешь запомнить о дне...",
    submit: "Сохранить",
    submit_update: "Обновить запись",
    submit_create: "Создать запись",
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

// Everything the bot passed in the page URL — the form mode, filled dates and
// pre-fill values. Parsed once; the whole script reads from here.
const pageParams = new URLSearchParams(location.search);

// This page is one of two forms, chosen by ?form=sleep|day (default: day).
// We hide the other form's fields and only send this part on submit.
const formMode = pageParams.get("form") === "sleep" ? "sleep" : "day";
const editMode = pageParams.has("mode"); // opened from the calendar to view/edit a day
const editUpdate = pageParams.get("mode") === "update"; // an existing entry (vs a new one)
document.querySelector("h1").textContent = dict[formMode === "sleep" ? "title_sleep" : "title_day"];
document.querySelector(".subtitle").textContent = dict[formMode === "sleep" ? "subtitle_sleep" : "subtitle_day"];
document.querySelectorAll(formMode === "sleep" ? ".part-day" : ".part-sleep").forEach((el) => {
  el.hidden = true;
});

const form = document.getElementById("form");
const submitButton = form.querySelector('button[type="submit"]');

// ---- Entry date: default today, and block days that are already filled ----
const dateInput = form.date;
const dateWarning = document.getElementById("dateWarning");

// Local date as YYYY-MM-DD (not UTC — avoids the day being off near midnight).
function isoDate(d) {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}
const today = isoDate(new Date());
// A reminder may open the form pre-set to a specific day via ?date=YYYY-MM-DD.
const requestedDate = pageParams.get("date");
const startDate =
  requestedDate && /^\d{4}-\d{2}-\d{2}$/.test(requestedDate) && requestedDate <= today
    ? requestedDate
    : today;
dateInput.value = startDate;
dateInput.max = today; // no filling in the future

// Dates already saved — the bot passes them in the form URL as ?filled=a,b,c
const filledDates = new Set(
  (pageParams.get("filled") || "")
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean)
);

function checkDate() {
  const taken = filledDates.has(dateInput.value);
  dateInput.classList.toggle("taken", taken);
  dateWarning.hidden = !taken;
  submitButton.disabled = taken;
}
dateInput.addEventListener("input", checkDate);
checkDate();

// ---- Sliders: start empty & grey; a value is recorded only once you move one ----
form.querySelectorAll('input[type="range"]').forEach((range) => {
  const slider = range.closest(".slider");
  const out = slider.querySelector("output");
  slider.classList.add("untouched"); // grey, no value yet
  out.textContent = "—";
  range.addEventListener("input", () => {
    slider.classList.remove("untouched");
    out.textContent = range.value;
  });
});

// A slider's answer: null until it has been touched, then its number.
function sliderValue(name) {
  const range = form[name];
  return range.closest(".slider").classList.contains("untouched") ? null : Number(range.value);
}

// ---- Dream note: show the text field only when there were some kind of dreams ----
// The textarea keeps its text when hidden, so switching between the options never
// loses what was typed. Whether it's actually saved is decided at submit.
const dreamsWithContent = ["dreams", "nightmares", "anxious"];
const dreamNoteField = document.getElementById("dreamNoteField");
function updateDreamNote() {
  dreamNoteField.hidden = !dreamsWithContent.includes(form.dreams.value);
}
form.querySelectorAll('input[name="dreams"]').forEach((radio) => {
  radio.addEventListener("change", updateDreamNote);
});
updateDreamNote();

// ---- Medications: each section (sleep + day) is an independent add-from-dropdown ----
// The drugs the pickers offer come from the bot's URL (?meds=Name 200mg; Other):
// the list is personal, so it lives in the bot's .env (MEDICATIONS), never in this
// public page. A drug's dose is the usual one, pre-filled when it's added ("" = blank).
// With no ?meds= the pickers offer only "Other…" (free-text entry).
const medicationCatalog = parseMedications(pageParams.get("meds"));

// "Name 200mg; Other" -> [{ name, dose }] (dose "" when absent). The same format
// the sheet cells and every medication URL param use.
function parseMedications(str) {
  if (!str) return [];
  return str
    .split(";")
    .map((s) => s.trim())
    .filter(Boolean)
    .map((item) => {
      const m = item.match(/^(.*?)\s+([\d.]+)mg$/);
      return m ? { name: m[1], dose: m[2] } : { name: item, dose: "" };
    });
}

// Wire up every medications section on the page (the sleep one and the day one).
document.querySelectorAll(".medications").forEach(setupMedications);

// Pre-fill medications from the most recent previous entry: the bot passes that
// day's drugs and doses in ?def_meds= (as "Name 200mg; Other 3mg"). Skipped when
// editing an existing entry — there we load that entry's own meds instead. Every
// pre-filled drug can still be removed or have its dose changed.
if (!editUpdate) {
  const section = document.querySelector(
    formMode === "sleep" ? ".medications.part-sleep" : ".medications.part-day"
  );
  prefillMedsFromString(section, pageParams.get("def_meds"));
}

function setupMedications(section) {
  const picker = section.querySelector(".med-picker");
  const list = section.querySelector(".med-list");
  populatePicker(picker);
  picker.addEventListener("change", () => {
    const value = picker.value;
    if (!value) return;
    if (value === "__other__") {
      addMedicationRow(picker, list, { custom: true }); // free-text medication
    } else {
      const med = medicationCatalog.find((m) => m.name === value);
      const option = picker.querySelector(`option[value="${value}"]`);
      addMedicationRow(picker, list, { name: value, label: option.textContent, dose: med.dose });
      setOptionTaken(picker, value, true); // can't add the same drug twice
    }
    picker.value = ""; // back to the "+ Add…" placeholder
  });
}

// Fill a picker with the catalog drugs (shown exactly as named in .env) plus "Other…".
function populatePicker(picker) {
  medicationCatalog.forEach(({ name }) => {
    const option = document.createElement("option");
    option.value = name;
    option.textContent = name;
    picker.append(option);
  });
  const other = document.createElement("option");
  other.value = "__other__";
  other.textContent = dict.med_other;
  picker.append(other);
}

// An added drug leaves its dropdown; removing its row puts it back.
function setOptionTaken(picker, name, taken) {
  const option = picker.querySelector(`option[value="${name}"]`);
  if (!option) return; // custom drugs have no dropdown entry
  option.hidden = taken;
  option.disabled = taken;
}

// Build one medication row inside `list`; `picker` is its own dropdown (to restore on remove).
function addMedicationRow(picker, list, { name = "", label = "", dose = "", custom = false }) {
  const row = document.createElement("div");
  row.className = "med";
  row.dataset.name = name;

  let nameEl;
  if (custom) {
    nameEl = document.createElement("input");
    nameEl.type = "text";
    nameEl.className = "med-name med-name-input";
    nameEl.placeholder = dict.med_name_placeholder;
    if (name) nameEl.value = name; // pre-filled custom medication when editing
  } else {
    nameEl = document.createElement("span");
    nameEl.className = "med-name";
    nameEl.textContent = label;
  }

  const doseWrap = document.createElement("span");
  doseWrap.className = "med-dose";
  const doseInput = document.createElement("input");
  doseInput.type = "number";
  doseInput.min = "0";
  doseInput.step = "0.5";
  doseInput.inputMode = "decimal";
  doseInput.placeholder = "0";
  doseInput.value = dose;
  const unit = document.createElement("span");
  unit.className = "unit";
  unit.textContent = dict.mg;
  doseWrap.append(doseInput, unit);

  const remove = document.createElement("button");
  remove.type = "button";
  remove.className = "med-remove";
  remove.textContent = "×";
  remove.setAttribute("aria-label", "remove");
  remove.addEventListener("click", () => {
    row.remove();
    // Put a fixed drug back into its dropdown (custom rows have nothing to restore).
    if (!custom) setOptionTaken(picker, name, false);
  });

  row.append(nameEl, doseWrap, remove);
  list.append(row);
  if (custom) nameEl.focus(); // let the user type the name right away
}

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

// ---- Collect every medication row the user added, with its dose ----
function collectMedications(list) {
  const meds = [];
  list.querySelectorAll(".med").forEach((row) => {
    const nameInput = row.querySelector(".med-name-input");
    const name = nameInput ? nameInput.value.trim() : row.dataset.name;
    if (!name) return; // skip a custom row left without a name
    const dose = row.querySelector('input[type="number"]').value;
    meds.push({ name, dose });
  });
  return meds;
}

// The two medication lists (sleep form and day form).
const sleepMedList = document.querySelector(".medications.part-sleep .med-list");
const dayMedList = document.querySelector(".medications.part-day .med-list");

// ---- Submit: gather everything and hand it to the bot ----
form.addEventListener("submit", (event) => {
  event.preventDefault();

  // Safety net: never send an already-filled date (except when editing on purpose).
  if (!editMode && filledDates.has(dateInput.value)) return;

  const minutes = sleepMinutes();
  const answers = {
    form_type: formMode,
    edit: editMode,
    date: dateInput.value,
    bedtime: bedtime.value,
    wake: wake.value,
    sleep_hours: minutes === null ? null : Math.round((minutes / 60) * 100) / 100,
    rested: sliderValue("rested"),
    dreams: form.dreams.value,
    // Only send the dream text when there were some kind of dreams — if the user
    // typed something then switched back to "none", it must not be saved.
    dream_note: dreamsWithContent.includes(form.dreams.value) ? form.dream_note.value : "",
    sleep_medications: collectMedications(sleepMedList),
    state: sliderValue("state"),
    anxiety: sliderValue("anxiety"),
    irritability: sliderValue("irritability"),
    libido: sliderValue("libido"),
    drowsiness: sliderValue("drowsiness"),
    appetite: sliderValue("appetite"),
    energy: sliderValue("energy"),
    ate_well: sliderValue("ate_well"),
    menstruation: form.menstruation.value === "yes",
    sex: form.sex.value === "yes",
    masturbation: form.masturbation.value === "yes",
    headache: form.headache.value === "yes",
    smoking: form.smoking.value === "yes",
    medications: collectMedications(dayMedList),
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

// ---- Edit mode: locked date + pre-filled values from the ?p_* params ----
if (editMode) applyEditMode();

function applyEditMode() {
  const p = pageParams;

  // The date is fixed to the day being edited.
  dateInput.value = p.get("date") || dateInput.value;
  dateInput.disabled = true;
  submitButton.textContent = editUpdate ? dict.submit_update : dict.submit_create;

  if (formMode === "sleep") {
    setTime("bedtime", p.get("p_bedtime"));
    setTime("wake", p.get("p_wake"));
    setSlider("rested", p.get("p_rested"));
    setRadio("dreams", p.get("p_dreams"));
    setText("dream_note", p.get("p_dream_note"));
    prefillMedsFromString(document.querySelector(".medications.part-sleep"), p.get("p_sleep_meds"));
  } else {
    setSlider("state", p.get("p_state"));
    setSlider("anxiety", p.get("p_anxiety"));
    setSlider("irritability", p.get("p_irritability"));
    setSlider("libido", p.get("p_libido"));
    setSlider("drowsiness", p.get("p_drowsiness"));
    setSlider("appetite", p.get("p_appetite"));
    setSlider("energy", p.get("p_energy"));
    setSlider("ate_well", p.get("p_ate_well"));
    setRadio("menstruation", p.get("p_menstruation"));
    setRadio("sex", p.get("p_sex"));
    setRadio("masturbation", p.get("p_masturbation"));
    setRadio("headache", p.get("p_headache"));
    setRadio("smoking", p.get("p_smoking"));
    setText("note", p.get("p_note"));
    prefillMedsFromString(document.querySelector(".medications.part-day"), p.get("p_meds"));
  }
}

function setTime(name, value) {
  if (!value) return;
  form[name].value = value;
  form[name].dispatchEvent(new Event("input")); // redraw the sleep arc
}
function setSlider(name, value) {
  if (value === null || value === "" || value === undefined) return;
  const range = form[name];
  const slider = range.closest(".slider");
  range.value = value;
  slider.classList.remove("untouched"); // a saved value counts as touched
  slider.querySelector("output").textContent = range.value;
}
function setRadio(name, value) {
  if (!value) return;
  form.querySelectorAll(`input[name="${name}"]`).forEach((r) => {
    r.checked = r.value === value;
  });
  if (name === "dreams") updateDreamNote();
}
function setText(name, value) {
  if (value) form[name].value = value;
}
function prefillMedsFromString(section, str) {
  const picker = section.querySelector(".med-picker");
  const list = section.querySelector(".med-list");
  parseMedications(str).forEach(({ name, dose }) => {
    const option = picker.querySelector(`option[value="${name}"]`);
    if (option) {
      // A catalog drug: a fixed row, and it leaves the dropdown.
      addMedicationRow(picker, list, { name, label: option.textContent, dose });
      setOptionTaken(picker, name, true);
    } else {
      // Not in the catalog (any more): an editable free-text row.
      addMedicationRow(picker, list, { custom: true, name, dose });
    }
  });
}
})();
