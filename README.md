# Unified Backend on PocketBase — Self-Hosted on Raspberry Pi 5

All backend services for my applications are powered by **[PocketBase](https://pocketbase.io/)** — a lightweight, blazing-fast, all-in-one backend written in Go.

I deploy **a single PocketBase instance** on a **Raspberry Pi 5**, hosted right on my home network.

## 🌐 How It Works

- **Frontend apps** (e.g. Svelte, React) are served from structured subpaths (`/frontend1`, `/frontend2`, etc) within PocketBase's `pb_public/` folder.
- All apps share a unified backend served from `https://api.bfrisco.com`.
- I use **Cloudflare Tunnel** to securely expose my Raspberry Pi to the internet.
- Each app connects to the same PocketBase instance, using namespaced collections and rules to stay logically isolated.

## 🧰 What PocketBase Provides

- REST + Realtime APIs
- Authentication & Authorization
- File storage
- Built-in admin dashboard
- Single binary deployment (no Node, no Docker, no database server)

## 💡 Why It's Awesome
- 🪶 Super lightweight (runs on a Pi!)
- 🛡️ Secure (no open ports, protected via Cloudflare)
- ⚡ Instant APIs for all my side projects
- 🔌 No monthly hosting bills

---

### 🧪 Live Backend
👉 [https://api.bfrisco.com](https://api.bfrisco.com)
