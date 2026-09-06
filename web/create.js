/* burnbox — create page.
   Everything security-relevant happens in createSecret(): the key is made here,
   used here, and never put in the request body. */

const MAX_CHARS = 10000;

/* ---------- bytes <-> URL-safe text ---------- */

function toB64url(buf) {
  const b = new Uint8Array(buf);
  let bin = "";
  for (let i = 0; i < b.length; i++) bin += String.fromCharCode(b[i]);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/* ---------- the crypto ---------- */

async function createSecret(plaintext) {
  const key = await crypto.subtle.generateKey(
    { name: "AES-GCM", length: 256 }, true, ["encrypt", "decrypt"]
  );

  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ct = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv }, key, new TextEncoder().encode(plaintext)
  );

  const blob = new Uint8Array(12 + ct.byteLength);
  blob.set(iv, 0);
  blob.set(new Uint8Array(ct), 12);

  const res = await fetch("/api/secrets", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ciphertext: toB64url(blob) }),
  });

  if (res.status === 429) throw new Error("Too many requests from your network. Wait a minute and try again.");
  if (res.status === 413) throw new Error("That secret is too large.");
  if (res.status === 503) throw new Error("The service is temporarily unavailable. Try again in a moment.");
  if (!res.ok)            throw new Error("The server responded with " + res.status + ".");

  const { share_id, creator_id } = await res.json();
  const keyText = toB64url(await crypto.subtle.exportKey("raw", key));

  return {
    shareLink: `${location.origin}/s/${share_id}#${keyText}`,
    creatorLink: `${location.origin}/c/${creator_id}`,
  };
}

/* ---------- elements ---------- */

const secretEl    = document.getElementById("secret");
const counterEl   = document.getElementById("counter");
const meterEl     = document.getElementById("meter");
const meterFillEl = document.getElementById("meterFill");
const createBtn   = document.getElementById("create");
const outEl       = document.getElementById("out");
const shareEl     = document.getElementById("shareLink");
const creatorEl   = document.getElementById("creatorLink");
const errEl       = document.getElementById("createError");
const errTextEl   = document.getElementById("createErrorText");
const anotherBtn  = document.getElementById("another");

/* ---------- character counter ---------- */

function updateCounter() {
  const used = secretEl.value.length;
  const left = MAX_CHARS - used;
  const ratio = Math.min(used / MAX_CHARS, 1);

  counterEl.textContent = left >= 0
    ? left.toLocaleString() + " characters left"
    : Math.abs(left).toLocaleString() + " characters over the limit";

  meterFillEl.style.width = (ratio * 100).toFixed(1) + "%";

  const state = left < 0 ? "over" : ratio > 0.85 ? "near" : "";
  counterEl.className = "counter " + state;
  meterEl.className   = "meter " + state;
  secretEl.classList.toggle("over", left < 0);

  createBtn.disabled = used === 0 || left < 0;
}

secretEl.addEventListener("input", updateCounter);

/* ---------- copy buttons ---------- */

function wireCopy(buttonId, sourceEl) {
  const btn = document.getElementById(buttonId);

  btn.addEventListener("click", async () => {
    const text = sourceEl.textContent;
    const original = btn.textContent;

    try {
      await navigator.clipboard.writeText(text);
      btn.textContent = "Copied";
    } catch (e) {
      // Clipboard access can be refused; select the text so it can be copied by hand.
      const range = document.createRange();
      range.selectNodeContents(sourceEl);
      const sel = window.getSelection();
      sel.removeAllRanges();
      sel.addRange(range);
      btn.textContent = "Press Ctrl+C";
    }

    setTimeout(() => { btn.textContent = original; }, 1600);
  });
}

wireCopy("copyShare", shareEl);
wireCopy("copyCreator", creatorEl);

/* ---------- create ---------- */

function showError(message) {
  errTextEl.textContent = message;
  errEl.classList.remove("hidden");
}

createBtn.addEventListener("click", async () => {
  const plaintext = secretEl.value;
  if (!plaintext) return;

  errEl.classList.add("hidden");

  if (plaintext.length > MAX_CHARS) {
    showError("Shorten it to " + MAX_CHARS.toLocaleString() + " characters or fewer.");
    return;
  }

  createBtn.disabled = true;
  createBtn.textContent = "Encrypting…";

  try {
    if (!window.crypto || !crypto.subtle) {
      throw new Error("This browser won't provide encryption over an insecure connection. Use HTTPS.");
    }

    const links = await createSecret(plaintext);

    shareEl.textContent = links.shareLink;
    creatorEl.textContent = links.creatorLink;
    outEl.classList.remove("hidden");

    // Don't leave the plaintext sitting on screen.
    secretEl.value = "";
    updateCounter();

    outEl.scrollIntoView({ behavior: "smooth", block: "nearest" });

  } catch (e) {
    showError(e.message || "Something went wrong.");
  } finally {
    createBtn.textContent = "Create secret link";
    updateCounter();
  }
});

anotherBtn.addEventListener("click", () => {
  outEl.classList.add("hidden");
  shareEl.textContent = "";
  creatorEl.textContent = "";
  secretEl.focus();
  secretEl.scrollIntoView({ behavior: "smooth", block: "center" });
});

updateCounter();
