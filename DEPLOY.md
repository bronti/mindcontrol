# Deploying Makhi-Bot to a Google Cloud free VM

This is the step-by-step recipe to run the bot **24/7** on a Google Cloud
**`e2-micro` Always Free** VM. It assumes the VM already exists (see the
free-tier settings at the bottom if you still need to create it).

The VM authenticates to Google Sheets with **no key file** — it uses the
**service account attached to the VM** (Application Default Credentials). So
you never copy `google-cloud-key.json` to the server.

---

## 0. One-time account prerequisites (before creating the VM)

1. **Billing** → link a billing account (add a card). A correctly configured
   `e2-micro` still bills **$0**; billing just has to be *enabled*.
2. **Enable the "Compute Engine API"** (the Sheets API is already on).
3. When creating the VM, under **Identity and API access → Service account**,
   pick your **existing Sheets service account** (the one already sharing the
   sheet) and set **Access scopes → "Allow full access to all Cloud APIs"**.
   This is what makes the keyless auth work.

---

## 1. SSH into the VM

In the Cloud Console, on the VM Instances list, click the **`SSH`** button next
to the instance. A browser terminal opens. Run everything below there.

---

## 2. Install Go and git

Debian 13 (trixie) ships an older Go than this project needs, so install Go
from the official tarball rather than `apt`.

```bash
# git is fine from apt:
sudo apt-get update && sudo apt-get install -y git

# Install a current Go from the official tarball:
GO_VERSION=1.26.4
curl -LO https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
export PATH=$PATH:/usr/local/go/bin
go version    # should print go1.26.4
```

---

## 3. Create the app folder and a dedicated user

The bot runs as its own unprivileged user `makhi` (not root, not your login) —
that's what the systemd unit expects.

```bash
sudo useradd --system --create-home --home-dir /opt/makhi-bot --shell /usr/sbin/nologin makhi
```

---

## 4. Clone the repo

```bash
sudo git clone https://github.com/bronti/mindcontrol.git /opt/makhi-bot
sudo chown -R makhi:makhi /opt/makhi-bot
```

---

## 5. Upload and edit `.env` (the only secret on the server)

`.env` is gitignored, so cloning does NOT bring it — you must add it by hand.

1. In the browser SSH window, click the **gear / upload** icon (top right) and
   upload your local **`.env`**. It lands in your home dir (e.g. `~/.env`).
2. Move it into the app folder and fix ownership:
   ```bash
   sudo mv ~/.env /opt/makhi-bot/.env
   sudo chown makhi:makhi /opt/makhi-bot/.env
   sudo chmod 600 /opt/makhi-bot/.env
   ```
3. Edit it for the server:
   ```bash
   sudo -u makhi nano /opt/makhi-bot/.env
   ```
   - **Comment out (or delete) the `GOOGLE_APPLICATION_CREDENTIALS` line.**
     On the VM, auth falls back to the attached service account. If this line
     is present, the bot looks for a key file that isn't there and fails.
     ```
     # GOOGLE_APPLICATION_CREDENTIALS=google-cloud-key.json
     ```
   - Set **`TIMEZONE=Asia/Tbilisi`** so reminders (21:00 / 14:00) fire on
     your local clock, not the VM's UTC.
   - Confirm `BOT_TOKEN`, `OWNER_ID`, `WEB_APP_URL`, `MEDICATIONS` are correct.

   Do **NOT** upload `google-cloud-key.json` — no key file belongs on the VM.

---

## 6. Build the bot

```bash
cd /opt/makhi-bot
sudo -u makhi env PATH=$PATH go build -o makhi-bot .
```

If the build gets **killed / OOM** (only 1 GB RAM, and `google.golang.org/api`
is heavy), add a temporary swap file and rebuild:

```bash
sudo fallocate -l 2G /swapfile && sudo chmod 600 /swapfile
sudo mkswap /swapfile && sudo swapon /swapfile
# rebuild, then optionally: sudo swapoff /swapfile && sudo rm /swapfile
```

(Alternative: cross-compile on Windows and upload just the binary —
`$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o makhi-bot .` — then you can
skip installing Go on the VM entirely.)

---

## 7. Install the systemd service

The unit file lives in the repo at `deploy/makhi-bot.service`.

```bash
sudo cp /opt/makhi-bot/deploy/makhi-bot.service /etc/systemd/system/makhi-bot.service
sudo systemctl daemon-reload
sudo systemctl enable makhi-bot     # start automatically on every boot
sudo systemctl start makhi-bot      # start it now
```

---

## 8. Verify it's running

```bash
sudo systemctl status makhi-bot     # should say "active (running)"
journalctl -u makhi-bot -f          # live logs — Ctrl-C to stop watching
```

Then message the bot on Telegram. If `OWNER_ID` was empty, the log prints your
Telegram id — paste it into `.env` and restart (step 9).

---

## 9. Everyday operations

```bash
sudo systemctl restart makhi-bot    # after editing .env
sudo systemctl stop makhi-bot       # stop the bot
journalctl -u makhi-bot -f          # watch logs
journalctl -u makhi-bot --since "1 hour ago"
```

### Updating to new code later

```bash
cd /opt/makhi-bot
sudo -u makhi git pull
sudo -u makhi env PATH=$PATH go build -o makhi-bot .
sudo systemctl restart makhi-bot
```

(Frontend/form changes go live through GitHub Pages on their own — only Go
changes need this rebuild + restart.)

---

## Free-tier settings recap (when creating the VM)

Get any one of these wrong and the VM stops being free:

| Setting        | Must be                                            |
| -------------- | -------------------------------------------------- |
| Region         | `us-central1` (or `us-west1` / `us-east1`)         |
| Machine type   | `e2-micro`                                         |
| OS             | Debian GNU/Linux 13 (trixie)                       |
| Boot disk      | **Standard** persistent disk, ≤ 30 GB (10 is fine) |
| Network tier   | **Standard** (not Premium)                         |
| Count          | only **1** such VM per billing account             |
| Service account| your existing **Sheets** service account, full API access scope |

No firewall rule / "Allow HTTP(S)" is needed — the bot only makes **outbound**
connections (Telegram long-polling + Sheets); nothing connects *to* the VM
except your SSH.
