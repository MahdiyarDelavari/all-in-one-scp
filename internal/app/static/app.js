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
const authButtons = document.querySelectorAll("[data-auth-mode]");
const testBtn = document.getElementById("testBtn");
const sshBtn = document.getElementById("sshBtn");
const downloadBtn = document.getElementById("downloadBtn");
const heroSshBtn = document.getElementById("heroSshBtn");
const heroDownloadBtn = document.getElementById("heroDownloadBtn");
const terminalOutput = document.getElementById("terminalOutput");
const terminalForm = document.getElementById("terminalForm");
const terminalInput = document.getElementById("terminalInput");
const terminalSendBtn = document.getElementById("terminalSendBtn");
const sshDisconnectBtn = document.getElementById("sshDisconnectBtn");

let sshConnected = false;

restoreForm();
updateAuthPanels();
connectLogs();
connectSSHLogs();
wirePersistence();
updateSSHUI();

authMode.addEventListener("change", syncAuthButtons);
authMode.addEventListener("change", updateAuthPanels);
document.getElementById("clearLogBtn").addEventListener("click", () => {
  logBox.textContent = "";
});

testBtn.addEventListener("click", () => runAction("/api/test", "Testing connection..."));
sshBtn.addEventListener("click", connectSSH);
downloadBtn.addEventListener("click", () => runAction("/api/download", "Downloading..."));
heroSshBtn.addEventListener("click", connectSSH);
heroDownloadBtn.addEventListener("click", () => runAction("/api/download", "Downloading..."));
document.getElementById("quitBtn").addEventListener("click", async () => {
  await fetch("/api/quit", { method: "POST" });
  statusLine.textContent = "App is shutting down.";
});
terminalForm.addEventListener("submit", sendSSHCommand);
sshDisconnectBtn.addEventListener("click", disconnectSSH);
for (const button of authButtons) {
  button.addEventListener("click", () => {
    authMode.value = button.dataset.authMode;
    syncAuthButtons();
    updateAuthPanels();
    saveForm();
  });
}

function updateAuthPanels() {
  document.getElementById("auth-key").classList.toggle("hidden", authMode.value !== "key");
  document.getElementById("auth-password").classList.toggle("hidden", authMode.value !== "password");
  document.getElementById("auth-passwordEnv").classList.toggle("hidden", authMode.value !== "passwordEnv");
}

function syncAuthButtons() {
  for (const button of authButtons) {
    button.classList.toggle("auth__choice--active", button.dataset.authMode === authMode.value);
  }
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

async function connectSSH() {
  setBusy(true);
  statusLine.textContent = "Connecting SSH...";
  terminalOutput.textContent = "";

  try {
    const response = await fetch("/api/ssh/connect", {
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

    sshConnected = true;
    updateSSHUI();
    terminalInput.focus();
    statusLine.textContent = result.message || "SSH connected.";
    showToast(result.message || "SSH connected.", "success");
  } catch (error) {
    const message = "Error: " + error.message;
    statusLine.textContent = message;
    showToast(message, "error");
  } finally {
    setBusy(false);
  }
}

async function sendSSHCommand(event) {
  event.preventDefault();

  if (!sshConnected) {
    showToast("Error: connect SSH first.", "error");
    return;
  }

  const input = terminalInput.value.trim();
  if (!input) {
    return;
  }

  terminalSendBtn.disabled = true;

  try {
    const response = await fetch("/api/ssh/input", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ input }),
    });

    const result = await response.json();
    if (!result.ok) {
      const message = "Error: " + result.error;
      statusLine.textContent = message;
      showToast(message, "error");
      return;
    }

    terminalInput.value = "";
    statusLine.textContent = "Command sent.";
  } catch (error) {
    const message = "Error: " + error.message;
    statusLine.textContent = message;
    showToast(message, "error");
  } finally {
    terminalSendBtn.disabled = false;
    terminalInput.focus();
  }
}

async function disconnectSSH() {
  try {
    const response = await fetch("/api/ssh/disconnect", {
      method: "POST",
    });

    const result = await response.json();
    if (!result.ok) {
      const message = "Error: " + result.error;
      statusLine.textContent = message;
      showToast(message, "error");
      return;
    }

    sshConnected = false;
    updateSSHUI();
    statusLine.textContent = result.message || "SSH disconnected.";
    showToast(result.message || "SSH disconnected.", "success");
  } catch (error) {
    const message = "Error: " + error.message;
    statusLine.textContent = message;
    showToast(message, "error");
  }
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
  heroSshBtn.disabled = busy;
  heroDownloadBtn.disabled = busy;
}

function updateSSHUI() {
  sshBtn.textContent = sshConnected ? "Reconnect SSH" : "Connect SSH";
  heroSshBtn.textContent = sshConnected ? "Reconnect SSH" : "Connect SSH";
  terminalInput.disabled = !sshConnected;
  terminalSendBtn.disabled = !sshConnected;
  sshDisconnectBtn.disabled = !sshConnected;
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

function connectSSHLogs() {
  const stream = new EventSource("/api/ssh/events");
  stream.onmessage = (event) => {
    terminalOutput.textContent += event.data;
    terminalOutput.scrollTop = terminalOutput.scrollHeight;

    if (event.data.includes("[SSH session closed")) {
      sshConnected = false;
      updateSSHUI();
    }
  };

  stream.onerror = () => {
    showToast("Error: SSH stream disconnected.", "error");
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

  syncAuthButtons();
}
