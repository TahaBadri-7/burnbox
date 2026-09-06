/* burnbox — creator page.
   Reports read / unread only. It has no key and the key is not in this URL,
   so there is nothing here that could ever show the secret. */

const creatorID = location.pathname.split("/").pop();

const statusEl    = document.getElementById("status");
const statusTextEl= document.getElementById("statusText");
const detailEl    = document.getElementById("detail");
const metaStatus  = document.getElementById("metaStatus");
const metaExpiry  = document.getElementById("metaExpiry");
const interceptEl = document.getElementById("interceptNote");
const btn         = document.getElementById("refresh");

function humanTime(iso) {
  const ms = new Date(iso) - new Date();
  if (ms <= 0) return "any moment now";

  const mins = Math.round(ms / 60000);
  if (mins < 60) return `in ${mins} minute${mins === 1 ? "" : "s"}`;

  const hours = Math.round(mins / 60);
  if (hours < 24) return `in ${hours} hour${hours === 1 ? "" : "s"}`;

  const days = Math.round(hours / 24);
  return `in ${days} day${days === 1 ? "" : "s"}`;
}

function show(state, cls, detail) {
  statusTextEl.textContent = state;
  statusEl.className = "state " + cls;
  detailEl.textContent = detail;
  metaStatus.textContent = state;
}

async function check() {
  btn.disabled = true;

  try {
    const res = await fetch(`/api/secrets/${creatorID}/status`);

    if (res.status === 404) {
      show("Gone", "state-gone",
        "This secret has expired, or this link was never valid. Either way there is nothing left on the server.");
      metaExpiry.textContent = "Already gone";
      interceptEl.classList.add("hidden");
      return;
    }

    if (!res.ok) {
      show("Can't check", "state-gone", "The service is temporarily unavailable. Try again in a moment.");
      btn.disabled = false;
      return;
    }

    const { read, expires_at } = await res.json();

    if (read) {
      show("Opened", "state-read",
        "Somebody opened this secret. It has been destroyed and cannot be read again.");
      metaExpiry.textContent = "Record clears " + humanTime(expires_at);
      interceptEl.classList.remove("hidden");
    } else {
      show("Not read yet", "state-unread",
        "Still waiting. Nobody has opened the link, and the secret will delete itself if nobody does.");
      metaExpiry.textContent = humanTime(expires_at);
      interceptEl.classList.add("hidden");
    }

    btn.disabled = false;

  } catch (e) {
    show("Can't check", "state-gone", "The service is temporarily unavailable. Try again in a moment.");
    btn.disabled = false;
  }
}

check();
btn.addEventListener("click", check);
