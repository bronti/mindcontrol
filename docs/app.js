// All user-facing strings, grouped by language.
// To add a language, add a new entry with the same keys.
const translations = {
  en: {
    title: "Daily questions",
    subtitle: "This is a form stub — we'll change the questions to fit you later.",
    q_mood: "How's your mood today?",
    mood_5: "😀 great", mood_4: "🙂 good", mood_3: "😐 okay", mood_2: "😕 meh", mood_1: "😞 bad",
    q_energy: "Energy level from 1 to 10?",
    q_note: "What was the main thing today?",
    note_placeholder: "a few words...",
    submit: "Submit",
    browser_alert: "Sending only works inside Telegram.\nCollected answers:\n\n",
  },
  ru: {
    title: "Ежедневные вопросы",
    subtitle: "Это заглушка формы — вопросы поменяем под тебя позже.",
    q_mood: "Как настроение сегодня?",
    mood_5: "😀 отлично", mood_4: "🙂 хорошо", mood_3: "😐 нормально", mood_2: "😕 так себе", mood_1: "😞 плохо",
    q_energy: "Сколько энергии по шкале 1–10?",
    q_note: "Что сегодня было главным?",
    note_placeholder: "пара слов...",
    submit: "Отправить",
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

form.addEventListener("submit", (event) => {
  event.preventDefault();

  // Collect the answers into a single object
  const answers = {
    mood: form.mood.value,
    energy: form.energy.value,
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
