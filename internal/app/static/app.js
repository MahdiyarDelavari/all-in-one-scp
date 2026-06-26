const fields = [
  "host",
  "port",
  "user",
  "insecure",
  "authMode",
  "keyPath",
  "passwordEnv",
  "remotePath",
  "localPath",
  "excludes",
];

const logBox = document.getElementById("log");
const statusLine = document.getElementById("status");
const toastContainer = document.getElementById("toastContainer");
const authMode = document.getElementById("authMode");
const testBtn = document.getElementById("testBtn");
const sshBtn = document.getElementById("sshBtn");
const downloadBtn = document.getElementById("downloadBtn");

restoreForm();
updateAuthPanels();
connectLogs();
wirePersistence();

authMode.addEventListener("change", updateAuthPanels);
document.getElementById("clearLogBtn").addEventListener("click", () => {
  logBox.textContent = "";
});

testBtn.addEventListener("click", () => runAction("/api/test", "Testing connection..."));
sshBtn.addEventListener("click", () => runAction("/api/open-ssh", "Opening SSH terminal..."));
downloadBtn.addEventListener("click", () => runAction("/api/download", "Downloading..."));
document.getElementById("quitBtn").addEventListener("click", async () => {
  await fetch("/api/quit", { method: "POST" });
  statusLine.textContent = "App is shutting down.";
});

function updateAuthPanels() {
  document.getElementById("auth-key").classList.toggle("hidden", authMode.value !== "key");
  document.getElementById("auth-password").classList.toggle("hidden", authMode.value !== "password");
  document.getElementById("auth-passwordEnv").classList.toggle("hidden", authMode.value !== "passwordEnv");
}

function buildPayload() {
  const mode = authMode.value;

  return {
    host: document.getElementById("host").value.trim(),
    port: Number(document.getElementById("port").value || "22"),
    user: document.getElementById("user").value.trim(),
    insecure: document.getElementById("insecure").checked,
    keyPath: mode === "key" ? document.getElementById("keyPath").value.trim() : "",
    password: mode === "password" ? document.getElementById("password").value : "",
    passwordEnv: mode === "passwordEnv" ? document.getElementById("passwordEnv").value.trim() : "",
    remotePath: document.getElementById("remotePath").value.trim(),
    localPath: document.getElementById("localPath").value.trim(),
    excludes: document.getElementById("excludes").value.trim(),
  };
}

async function runAction(url, busyText) {
  setBusy(true);
  statusLine.textContent = busyText;

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(buildPayload()),
    });

    const result = await response.json();
    if (!result.ok) {
      const message = "Error: " + result.error;
      statusLine.textContent = message;
      showToast(message, "error");
      return;
    }

    statusLine.textContent = result.message || "Done.";
    showToast(result.message || "Done.", "success");
  } catch (error) {
    const message = "Error: " + error.message;
    statusLine.textContent = message;
    showToast(message, "error");
  } finally {
    setBusy(false);
  }
}

function setBusy(busy) {
  testBtn.disabled = busy;
  sshBtn.disabled = busy;
  downloadBtn.disabled = busy;
}

function showToast(message, type) {
  const toast = document.createElement("div");
  toast.className = `toast toast--${type}`;
  toast.textContent = message;

  toastContainer.appendChild(toast);

  requestAnimationFrame(() => {
    toast.classList.add("toast--visible");
  });

  const hideToast = () => {
    toast.classList.remove("toast--visible");
    toast.addEventListener(
      "transitionend",
      () => {
        toast.remove();
      },
      { once: true },
    );
  };

  setTimeout(hideToast, 4000);
}

function connectLogs() {
  const stream = new EventSource("/api/events");
  stream.onmessage = (event) => {
    logBox.textContent += event.data + "\n";
    logBox.scrollTop = logBox.scrollHeight;
  };

  stream.onerror = () => {
    statusLine.textContent = "Log stream disconnected.";
    showToast("Log stream disconnected.", "error");
  };
}

function wirePersistence() {
  for (const fieldId of fields) {
    const element = document.getElementById(fieldId);
    const eventName = element.type === "checkbox" ? "change" : "input";
    element.addEventListener(eventName, saveForm);
  }
}

function saveForm() {
  const payload = {};
  for (const fieldId of fields) {
    const element = document.getElementById(fieldId);
    payload[fieldId] = element.type === "checkbox" ? element.checked : element.value;
  }
  localStorage.setItem("all-in-one-scp-form", JSON.stringify(payload));
}

function restoreForm() {
  const raw = localStorage.getItem("all-in-one-scp-form");
  if (!raw) {
    return;
  }

  try {
    const payload = JSON.parse(raw);
    for (const fieldId of fields) {
      const element = document.getElementById(fieldId);
      if (!(fieldId in payload)) {
        continue;
      }

      if (element.type === "checkbox") {
        element.checked = Boolean(payload[fieldId]);
      } else {
        element.value = payload[fieldId];
      }
    }
  } catch (_error) {
    localStorage.removeItem("all-in-one-scp-form");
  }
}
