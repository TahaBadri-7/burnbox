function toB64url(buf) {
  const b = new Uint8Array(buf);
  let bin = "";
  for (let i = 0; i < b.length; i++) bin += String.fromCharCode(b[i]);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

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

  if (!res.ok) throw new Error("server said " + res.status);

  const { share_id, creator_id } = await res.json();
  const keyText = toB64url(await crypto.subtle.exportKey("raw", key));

  return {
    shareLink: `${location.origin}/s/${share_id}#${keyText}`,
    creatorLink: `${location.origin}/c/${creator_id}`,
  };
}

const btn = document.getElementById("create");

btn.addEventListener("click", async () => {
  const plaintext = document.getElementById("secret").value;
  if (!plaintext) return;

  btn.disabled = true;

  try {
    const links = await createSecret(plaintext);
    document.getElementById("shareLink").textContent = links.shareLink;
    document.getElementById("creatorLink").textContent = links.creatorLink;
    document.getElementById("out").style.display = "block";
    document.getElementById("secret").value = "";
  } catch (e) {
    alert("Could not create link: " + e.message);
  } finally {
    btn.disabled = false;
  }
});
