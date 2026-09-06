const creatorID = location.pathname.split("/").pop();

const statusEl = document.getElementById("status");
const detailEl = document.getElementById("detail");
const btn      = document.getElementById("refresh");

function humanTime(iso) {
  const ms = new Date(iso) - new Date();
  if (ms <= 0) return "any moment now";

  const mins = Math.round(ms / 60000);
  if (mins < 60) return `in ${mins} minute${mins === 1 ? "" : "s"}`;

  const hours = Math.round(mins / 60);
  return `in ${hours} hour${hours === 1 ? "" : "s"}`;
}

function show(state, cls, detail) {
  statusEl.textContent = state;
  statusEl.className = "state " + cls;
  detailEl.textContent = detail;
}

async function check() {
  btn.disabled = true;

  try {
    const res = await fetch(`/api/secrets/${creatorID}/status`);

    if (res.status === 404) {
      show("Unknown link", "gone",
        "This secret has expired, or this link was never valid.");
      return;
    }

    if (!res.ok) {
      show("Can't check right now", "", "Try again in a moment.");
      btn.disabled = false;
      return;
    }

    const { read, expires_at } = await res.json();

    if (read) {
      show("Read", "read",
        "Someone opened this secret. It has been destroyed and cannot be read again.");
    } else {
      show("Not read yet", "unread",
        `Still waiting. It self-destructs ${humanTime(expires_at)} if nobody opens it.`);
    }

    btn.disabled = false;

  } catch (e) {
    show("Can't check right now", "", "Try again in a moment.");
    btn.disabled = false;
  }
}

check();
btn.addEventListener("click", check);
