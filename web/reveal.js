/* burnbox — reveal page.
   The key is read from location.hash and never sent anywhere.

   Decryption happens AFTER the burn, deliberately. Decrypting first would let
   anyone probe an ID with a junk key to learn whether a secret exists without
   consuming it — a free existence oracle. The cost of this ordering is that a
   corrupted link destroys the secret, which the page says plainly rather than
   hiding. */

function fromB64url(s) {
  s = s.replace(/-/g, "+").replace(/_/g, "/");
  while (s.length % 4) s += "=";
  return Uint8Array.from(atob(s), c => c.charCodeAt(0));
}

const shareID = location.pathname.split("/").pop();

// Read fresh every time: editing the part after # does not reload the page,
// so a value captured at load time goes stale.
function currentKey() {
  return location.hash.slice(1);
}

const stageEl      = document.getElementById("stage");
const revealBtn    = document.getElementById("reveal");
const missingKeyEl = document.getElementById("missingKey");
const resultEl     = document.getElementById("result");
const secretEl     = document.getElementById("secret");
const failureEl    = document.getElementById("failure");
const failTitleEl  = document.getElementById("failTitle");
const failTextEl   = document.getElementById("failText");
const warnBoxEl    = document.getElementById("warnBox");
const warnTextEl   = document.getElementById("warnText");
const copyBtn      = document.getElementById("copySecret");

/* Nothing left to try: replace the whole stage. */
function fail(title, text) {
  stageEl.classList.add("hidden");
  resultEl.classList.add("hidden");
  failTitleEl.textContent = title;
  failTextEl.textContent = text;
  failureEl.classList.remove("hidden");
}

/* Recoverable: leave the button alive. */
function warn(text) {
  warnTextEl.textContent = text;
  warnBoxEl.classList.remove("hidden");
}

function clearWarn() {
  warnBoxEl.classList.add("hidden");
}

/* ---------- missing key, before anything is burned ---------- */

if (!currentKey()) {
  missingKeyEl.classList.remove("hidden");
}

// Fires when the fragment is edited. No request is made and the script does not
// re-run, so this is the only signal we get.
window.addEventListener("hashchange", () => {
  if (currentKey()) missingKeyEl.classList.add("hidden");
});

/* ---------- reveal ---------- */

revealBtn.addEventListener("click", async () => {
  const keyText = currentKey();

  if (!keyText) {
    missingKeyEl.classList.remove("hidden");
    return;
  }

  clearWarn();
  revealBtn.disabled = true;
  revealBtn.textContent = "Opening…";

  try {
    const res = await fetch(`/api/secrets/${shareID}/burn`, { method: "POST" });

    if (res.status === 404) {
      fail("This secret is no longer available.",
        "It has already been read, it expired, or the link was never valid. " +
        "If you were expecting it, treat it as intercepted and ask the sender to rotate the credential.");
      return;
    }

    if (res.status === 429) {
      warn("Too many requests from your network. Wait a minute and try again — nothing has been destroyed.");
      revealBtn.disabled = false;
      revealBtn.textContent = "Reveal the secret";
      return;
    }

    if (!res.ok) {
      warn("The service is temporarily unavailable. Try again in a moment — nothing has been destroyed.");
      revealBtn.disabled = false;
      revealBtn.textContent = "Reveal the secret";
      return;
    }

    const { ciphertext } = await res.json();

    const blob = fromB64url(ciphertext);
    const iv   = blob.slice(0, 12);
    const ct   = blob.slice(12);

    const key = await crypto.subtle.importKey(
      "raw", fromB64url(keyText), { name: "AES-GCM" }, false, ["decrypt"]
    );

    const plain = await crypto.subtle.decrypt({ name: "AES-GCM", iv }, key, ct);

    secretEl.textContent = new TextDecoder().decode(plain);
    stageEl.classList.add("hidden");
    resultEl.classList.remove("hidden");

  } catch (e) {
    fail("Could not decrypt this secret.",
      "The key in the link is wrong or damaged. The secret has now been destroyed and cannot be recovered — " +
      "ask the sender to create a new one and copy the link whole.");
  }
});

/* ---------- copy ---------- */

copyBtn.addEventListener("click", async () => {
  const original = copyBtn.textContent;

  try {
    await navigator.clipboard.writeText(secretEl.textContent);
    copyBtn.textContent = "Copied";
  } catch (e) {
    const range = document.createRange();
    range.selectNodeContents(secretEl);
    const sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(range);
    copyBtn.textContent = "Press Ctrl+C";
  }

  setTimeout(() => { copyBtn.textContent = original; }, 1600);
});
