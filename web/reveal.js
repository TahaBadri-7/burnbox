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

const btn  = document.getElementById("reveal");
const out  = document.getElementById("secret");
const warn = document.getElementById("warn");

const MISSING_KEY =
  "This link is missing its key — paste the whole link, including the part after the #.";

// Nothing left to try: hide the button.
function fail(msg) {
  warn.textContent = msg;
  warn.style.display = "block";
  btn.style.display = "none";
}

// Recoverable: leave the button alive.
function warnOnly(msg) {
  warn.textContent = msg;
  warn.style.display = "block";
}

if (!currentKey()) {
  warnOnly(MISSING_KEY);
}

// Fires when the fragment is edited. No request is made and the script does not
// re-run, so this is the only signal we get.
window.addEventListener("hashchange", () => {
  if (currentKey()) warn.style.display = "none";
});

btn.addEventListener("click", async () => {
  const keyText = currentKey();

  if (!keyText) {
    warnOnly(MISSING_KEY);
    return;
  }

  btn.disabled = true;

  try {
    const res = await fetch(`/api/secrets/${shareID}/burn`, { method: "POST" });

    if (res.status === 404) {
      fail("This secret has already been read, or it never existed.");
      return;
    }

    if (!res.ok) {
      warnOnly("Something went wrong. Try again in a moment.");
      btn.disabled = false;
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

    out.textContent = new TextDecoder().decode(plain);
    out.style.display = "block";
    btn.style.display = "none";

  } catch (e) {
    fail("Could not decrypt — the key in the link may be wrong or damaged. The secret has now been destroyed.");
  }
});
