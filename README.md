# Makhi-Bot

A Telegram bot that asks me a few questions once a day and saves the answers to a
Google Sheet (the `Makhi-Bot` tab). Written in Go as a learning project.

> ℹ️ **This project is built mostly with [Claude](https://claude.ai) (Claude Code).**
> The code, structure, and docs were generated collaboratively with an AI assistant.

## Project layout

```
mindcontrol/
├── main.go            # the bot itself (replies to messages, /ping)
├── sheets.go          # writing a row to the Google Sheet (ready for later use)
├── docs/              # the answer form (Telegram Mini App) — served by GitHub Pages
│   ├── index.html
│   ├── style.css
│   └── app.js
├── .env.example       # template for the token file
├── .gitignore         # keeps secrets out of git
└── README.md
```

## Secrets (never committed to git)

These files are deliberately hidden via `.gitignore` — they must not be made public:

- `.env` — the bot token from [@BotFather](https://t.me/BotFather)
- `google-cloud-key.json` — the Google service account key

## Running the bot locally

1. Install [Go](https://go.dev/dl/).
2. Copy `.env.example` to `.env` and set your own `BOT_TOKEN`.
3. Put `google-cloud-key.json` in the project root.
4. Run:
   ```
   go run .
   ```
5. Send the bot `/ping` in Telegram — it should reply with "pong".

## The form (Telegram Mini App) via GitHub Pages

The form files live in the `docs/` folder. To turn them into a live site over HTTPS:

1. Push the project to a GitHub repository.
2. In the repo: **Settings → Pages**.
3. Under **Build and deployment**: Source — `Deploy from a branch`,
   branch `main`, folder `/docs`. Save.
4. After a minute the form will be available at
   `https://<your-username>.github.io/mindcontrol/`.
5. That URL is later attached to the bot as a Mini App button.

You can also open the form in a regular browser (to check the layout) — on submit it will
just show the collected answers.
