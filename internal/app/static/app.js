const FORM_STORAGE_KEY = "all-in-one-scp-form";
const PROFILE_STORAGE_KEY = "all-in-one-scp-profiles";
const HISTORY_STORAGE_KEY = "all-in-one-scp-history";
const MAX_HISTORY_ITEMS = 6;
const MAX_PROFILE_ITEMS = 8;

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
  "destinationPath",
  "excludes",
  "transferMode",
  "destHost",
  "destPort",
  "destUser",
  "destInsecure",
  "destAuthMode",
  "destKeyPath",
  "destPasswordEnv",
];

const scopeConfig = {
  source: {
    mode: document.getElementById("authMode"),
    buttons: document.querySelectorAll('[data-auth-scope="source"]'),
    panels: {
      key: document.getElementById("auth-key"),
      password: document.getElementById("auth-password"),
      passwordEnv: document.getElementById("auth-passwordEnv"),
    },
    ids: {
      host: "host",
      port: "port",
      user: "user",
      insecure: "insecure",
      keyPath: "keyPath",
      password: "password",
      passwordEnv: "passwordEnv",
    },
    profileList: document.getElementById("sourceProfiles"),
    profileSaveBtn: document.getElementById("sourceProfileSaveBtn"),
  },
  destination: {
    mode: document.getElementById("destAuthMode"),
    buttons: document.querySelectorAll('[data-auth-scope="destination"]'),
    panels: {
      key: document.getElementById("dest-auth-key"),
      password: document.getElementById("dest-auth-password"),
      passwordEnv: document.getElementById("dest-auth-passwordEnv"),
    },
    ids: {
      host: "destHost",
      port: "destPort",
      user: "destUser",
      insecure: "destInsecure",
      keyPath: "destKeyPath",
      password: "destPassword",
      passwordEnv: "destPasswordEnv",
    },
    profileList: document.getElementById("destinationProfiles"),
    profileSaveBtn: document.getElementById("destProfileSaveBtn"),
  },
};

const transferMode = document.getElementById("transferMode");
const transferModeButtons = document.querySelectorAll("[data-transfer-mode]");
const logBox = document.getElementById("log");
const statusLine = document.getElementById("status");
const toastContainer = document.getElementById("toastContainer");
const testBtn = document.getElementById("testBtn");
const sshBtn = document.getElementById("sshBtn");
const downloadBtn = document.getElementById("downloadBtn");
const remoteCopyBtn = document.getElementById("remoteCopyBtn");
const heroSshBtn = document.getElementById("heroSshBtn");
const heroDownloadBtn = document.getElementById("heroDownloadBtn");
const heroRemoteCopyBtn = document.getElementById("heroRemoteCopyBtn");
const terminalOutput = document.getElementById("terminalOutput");
const terminalForm = document.getElementById("terminalForm");
const terminalInput = document.getElementById("terminalInput");
const terminalSendBtn = document.getElementById("terminalSendBtn");
const sshDisconnectBtn = document.getElementById("sshDisconnectBtn");
const destinationServerCard = document.getElementById("destinationServerCard");
const localPathRow = document.getElementById("localPathRow");
const destinationPathRow = document.getElementById("destinationPathRow");
const recentTransfers = document.getElementById("recentTransfers");
const clearHistoryBtn = document.getElementById("clearHistoryBtn");

let sshConnected = false;
let profiles = loadProfiles();
let transferHistory = loadHistory();

restoreForm();
ensureDefaults();
syncAllAuthButtons();
updateAllAuthPanels();
syncTransferButtons();
updateTransferModeUI();
renderProfiles();
renderHistory();
connectLogs();
connectSSHLogs();
wirePersistence();
updateSSHUI();
setStatus("Ready.", "neutral");

for (const [scope, config] of Object.entries(scopeConfig)) {
  config.mode.addEventListener("change", () => {
    syncAuthButtons(scope);
    updateAuthPanels(scope);
  });

  config.profileSaveBtn.addEventListener("click", () => {
    saveProfile(scope);
  });

  for (const button of config.buttons) {
    button.addEventListener("click", () => {
      config.mode.value = button.dataset.authMode;
      syncAuthButtons(scope);
      updateAuthPanels(scope);
      saveForm();
    });
  }
}

transferMode.addEventListener("change", () => {
  syncTransferButtons();
  updateTransferModeUI();
});

for (const button of transferModeButtons) {
  button.addEventListener("click", () => {
    setTransferMode(button.dataset.transferMode);
  });
}

document.getElementById("clearLogBtn").addEventListener("click", () => {
  logBox.textContent = "";
});

clearHistoryBtn.addEventListener("click", () => {
  transferHistory = [];
  persistHistory();
  renderHistory();
  setStatus("Recent transfers cleared.", "neutral");
  showToast("Recent transfers cleared.", "success");
});

testBtn.addEventListener("click", () => runAction("/api/test", "Testing connection...", { recordHistory: false }));
sshBtn.addEventListener("click", connectSSH);
downloadBtn.addEventListener("click", () => {
  setTransferMode("download");
  runAction("/api/download", "Downloading...", { recordHistory: true });
});
remoteCopyBtn.addEventListener("click", () => {
  setTransferMode("remoteCopy");
  runAction("/api/remote-copy", "Copying to server 2...", { recordHistory: true });
});
heroSshBtn.addEventListener("click", connectSSH);
heroDownloadBtn.addEventListener("click", () => {
  setTransferMode("download");
  runAction("/api/download", "Downloading...", { recordHistory: true });
});
heroRemoteCopyBtn.addEventListener("click", () => {
  setTransferMode("remoteCopy");
  runAction("/api/remote-copy", "Copying to server 2...", { recordHistory: true });
});
document.getElementById("quitBtn").addEventListener("click", async () => {
  await fetch("/api/quit", { method: "POST" });
  setStatus("App is shutting down.", "neutral");
});
terminalForm.addEventListener("submit", sendSSHCommand);
sshDisconnectBtn.addEventListener("click", disconnectSSH);

function ensureDefaults() {
  if (!transferMode.value) {
    transferMode.value = "download";
  }
}

function syncAllAuthButtons() {
  syncAuthButtons("source");
  syncAuthButtons("destination");
}

function updateAllAuthPanels() {
  updateAuthPanels("source");
  updateAuthPanels("destination");
}

function syncAuthButtons(scope) {
  const config = scopeConfig[scope];
  for (const button of config.buttons) {
    button.classList.toggle("mode-pill--active", button.dataset.authMode === config.mode.value);
  }
}

function updateAuthPanels(scope) {
  const config = scopeConfig[scope];
  config.panels.key.classList.toggle("hidden", config.mode.value !== "key");
  config.panels.password.classList.toggle("hidden", config.mode.value !== "password");
  config.panels.passwordEnv.classList.toggle("hidden", config.mode.value !== "passwordEnv");
}

function setTransferMode(mode, persist = true) {
  transferMode.value = mode;
  syncTransferButtons();
  updateTransferModeUI();

  if (persist) {
    saveForm();
  }
}

function syncTransferButtons() {
  for (const button of transferModeButtons) {
    button.classList.toggle("mode-pill--active", button.dataset.transferMode === transferMode.value);
  }
}

function updateTransferModeUI() {
  const isRemoteCopy = transferMode.value === "remoteCopy";
  destinationPathRow.classList.toggle("hidden", !isRemoteCopy);
  localPathRow.classList.toggle("hidden", isRemoteCopy);
  remoteCopyBtn.classList.toggle("hidden", !isRemoteCopy);
  downloadBtn.classList.toggle("hidden", isRemoteCopy);
  destinationServerCard.classList.toggle("panel--inactive", !isRemoteCopy);

  for (const element of destinationServerCard.querySelectorAll("input, button, select")) {
    if (element.id === "destProfileSaveBtn") {
      element.disabled = !isRemoteCopy;
      continue;
    }

    if (element.closest(".profile-card")) {
      element.disabled = !isRemoteCopy;
      continue;
    }

    element.disabled = !isRemoteCopy;
  }

  heroDownloadBtn.classList.toggle("shortcut-btn--active", !isRemoteCopy);
  heroRemoteCopyBtn.classList.toggle("shortcut-btn--active", isRemoteCopy);
}

function setStatus(message, tone = "neutral") {
  statusLine.textContent = message;
  statusLine.className = `status-banner status-banner--${tone}`;
}

function buildConnectionState(scope, includePassword = true) {
  const config = scopeConfig[scope];
  const ids = config.ids;

  return {
    authMode: config.mode.value,
    host: document.getElementById(ids.host).value.trim(),
    port: Number(document.getElementById(ids.port).value || "22"),
    user: document.getElementById(ids.user).value.trim(),
    insecure: document.getElementById(ids.insecure).checked,
    keyPath: document.getElementById(ids.keyPath).value.trim(),
    password: includePassword ? document.getElementById(ids.password).value : "",
    passwordEnv: document.getElementById(ids.passwordEnv).value.trim(),
  };
}

function toRequestConnection(connection) {
  return {
    host: connection.host,
    port: connection.port,
    user: connection.user,
    insecure: connection.insecure,
    keyPath: connection.authMode === "key" ? connection.keyPath : "",
    password: connection.authMode === "password" ? connection.password : "",
    passwordEnv: connection.authMode === "passwordEnv" ? connection.passwordEnv : "",
  };
}

function sanitizeConnection(connection) {
  return {
    authMode: connection.authMode,
    host: connection.host,
    port: connection.port,
    user: connection.user,
    insecure: connection.insecure,
    keyPath: connection.keyPath,
    passwordEnv: connection.passwordEnv,
  };
}

function buildSnapshot(includePasswords = true) {
  const source = buildConnectionState("source", includePasswords);
  const destination = buildConnectionState("destination", includePasswords);

  return {
    transferMode: transferMode.value,
    remotePath: document.getElementById("remotePath").value.trim(),
    localPath: document.getElementById("localPath").value.trim(),
    destinationPath: document.getElementById("destinationPath").value.trim(),
    excludes: document.getElementById("excludes").value.trim(),
    source,
    destination,
  };
}

function buildRequestPayload() {
  const snapshot = buildSnapshot(true);

  return {
    source: toRequestConnection(snapshot.source),
    destination: toRequestConnection(snapshot.destination),
    remotePath: snapshot.remotePath,
    localPath: snapshot.localPath,
    destinationPath: snapshot.destinationPath,
    excludes: snapshot.excludes,
    transferMode: snapshot.transferMode,
  };
}

function buildHistoryEntry() {
  const snapshot = buildSnapshot(false);
  return {
    id: crypto.randomUUID(),
    timestamp: new Date().toISOString(),
    transferMode: snapshot.transferMode,
    remotePath: snapshot.remotePath,
    localPath: snapshot.localPath,
    destinationPath: snapshot.destinationPath,
    excludes: snapshot.excludes,
    source: sanitizeConnection(snapshot.source),
    destination: sanitizeConnection(snapshot.destination),
  };
}

function applyConnectionState(scope, state) {
  const config = scopeConfig[scope];
  const ids = config.ids;

  config.mode.value = state.authMode || "key";
  document.getElementById(ids.host).value = state.host || "";
  document.getElementById(ids.port).value = state.port || 22;
  document.getElementById(ids.user).value = state.user || "";
  document.getElementById(ids.insecure).checked = Boolean(state.insecure);
  document.getElementById(ids.keyPath).value = state.keyPath || "";
  document.getElementById(ids.password).value = "";
  document.getElementById(ids.passwordEnv).value = state.passwordEnv || "SSH_PASSWORD";

  syncAuthButtons(scope);
  updateAuthPanels(scope);
}

function applyTransferSnapshot(entry) {
  setTransferMode(entry.transferMode || "download", false);
  document.getElementById("remotePath").value = entry.remotePath || "";
  document.getElementById("localPath").value = entry.localPath || "";
  document.getElementById("destinationPath").value = entry.destinationPath || "";
  document.getElementById("excludes").value = entry.excludes || "";
  applyConnectionState("source", entry.source || {});
  applyConnectionState("destination", entry.destination || {});
  saveForm();
}

function saveProfile(scope) {
  const connection = sanitizeConnection(buildConnectionState(scope, false));
  if (!connection.host || !connection.user) {
    showToast("Error: host and username are required for a profile.", "error");
    return;
  }

  const idBase = `${connection.user}@${connection.host}:${connection.port}`;
  const nextProfile = {
    id: idBase,
    name: `${connection.user}@${connection.host}`,
    connection,
  };

  profiles = [nextProfile, ...profiles.filter((profile) => profile.id !== nextProfile.id)].slice(0, MAX_PROFILE_ITEMS);
  persistProfiles();
  renderProfiles();

  const passwordNote = connection.authMode === "password" ? " Password is not saved." : "";
  setStatus(`Saved profile ${nextProfile.name}.`, "success");
  showToast(`Saved profile ${nextProfile.name}.${passwordNote}`, "success");
}

function deleteProfile(profileId) {
  profiles = profiles.filter((profile) => profile.id !== profileId);
  persistProfiles();
  renderProfiles();
  setStatus("Profile removed.", "neutral");
  showToast("Profile removed.", "success");
}

function loadProfile(scope, profileId) {
  const profile = profiles.find((item) => item.id === profileId);
  if (!profile) {
    return;
  }

  applyConnectionState(scope, profile.connection);
  saveForm();
  setStatus(`Loaded ${profile.name}.`, "neutral");
  showToast(`Loaded ${profile.name} into ${scope === "source" ? "Server 1" : "Server 2"}.`, "success");
}

function renderProfiles() {
  renderProfileList(scopeConfig.source.profileList, "source");
  renderProfileList(scopeConfig.destination.profileList, "destination");
  updateTransferModeUI();
}

function renderProfileList(container, scope) {
  container.textContent = "";

  if (profiles.length === 0) {
    const empty = document.createElement("div");
    empty.className = "profile-empty";
    empty.textContent = "No profiles";
    container.appendChild(empty);
    return;
  }

  for (const profile of profiles) {
    const card = document.createElement("div");
    card.className = "profile-card";

    const loadButton = document.createElement("button");
    loadButton.className = "profile-card__load";
    loadButton.type = "button";
    loadButton.innerHTML = `<strong>${escapeHTML(profile.name)}</strong><span>${escapeHTML(profile.connection.authMode)} · ${escapeHTML(profile.connection.host)}</span>`;
    loadButton.addEventListener("click", () => loadProfile(scope, profile.id));

    const deleteButton = document.createElement("button");
    deleteButton.className = "profile-card__delete";
    deleteButton.type = "button";
    deleteButton.textContent = "×";
    deleteButton.setAttribute("aria-label", `Delete ${profile.name}`);
    deleteButton.addEventListener("click", () => deleteProfile(profile.id));

    card.appendChild(loadButton);
    card.appendChild(deleteButton);
    container.appendChild(card);
  }
}

function rememberTransfer(entry) {
  transferHistory = [entry, ...transferHistory.filter((item) => item.id !== entry.id)].slice(0, MAX_HISTORY_ITEMS);
  persistHistory();
  renderHistory();
}

function renderHistory() {
  recentTransfers.textContent = "";
  clearHistoryBtn.disabled = transferHistory.length === 0;

  if (transferHistory.length === 0) {
    const empty = document.createElement("div");
    empty.className = "history-empty";
    empty.textContent = "No transfers yet";
    recentTransfers.appendChild(empty);
    return;
  }

  for (const entry of transferHistory) {
    const card = document.createElement("div");
    card.className = "history-card";

    const top = document.createElement("div");
    top.className = "history-card__top";
    top.innerHTML = `<div class="history-card__badge">${entry.transferMode === "remoteCopy" ? "Server to Server" : "Download"}</div><div class="history-card__meta">${formatTime(entry.timestamp)}</div>`;

    const path = document.createElement("div");
    path.className = "history-card__path";
    path.innerHTML = `<strong>${escapeHTML(entry.remotePath || "-")}</strong><span>${escapeHTML(entry.transferMode === "remoteCopy" ? entry.destinationPath : entry.localPath || "-")}</span>`;

    const actions = document.createElement("div");
    actions.className = "history-card__actions";

    const loadBtn = document.createElement("button");
    loadBtn.type = "button";
    loadBtn.className = "history-card__btn";
    loadBtn.textContent = "Load";
    loadBtn.addEventListener("click", () => {
      applyTransferSnapshot(entry);
      setStatus("Transfer loaded.", "neutral");
      showToast("Transfer loaded.", "success");
    });

    const runBtn = document.createElement("button");
    runBtn.type = "button";
    runBtn.className = "history-card__btn history-card__btn--run";
    runBtn.textContent = "Run Again";
    runBtn.addEventListener("click", () => rerunHistoryEntry(entry));

    actions.appendChild(loadBtn);
    actions.appendChild(runBtn);

    card.appendChild(top);
    card.appendChild(path);
    card.appendChild(actions);
    recentTransfers.appendChild(card);
  }
}

function rerunHistoryEntry(entry) {
  applyTransferSnapshot(entry);

  if (requiresManualPassword(entry)) {
    setStatus("Password is required before rerun.", "error");
    showToast("Error: password is not saved. Enter it, then run again.", "error");
    return;
  }

  if (entry.transferMode === "remoteCopy") {
    runAction("/api/remote-copy", "Copying to server 2...", { recordHistory: true });
    return;
  }

  runAction("/api/download", "Downloading...", { recordHistory: true });
}

function requiresManualPassword(entry) {
  if (entry.source?.authMode === "password") {
    return true;
  }

  if (entry.transferMode === "remoteCopy" && entry.destination?.authMode === "password") {
    return true;
  }

  return false;
}

async function connectSSH() {
  setBusy(true);
  sshConnected = false;
  updateSSHUI();
  setStatus("Connecting SSH...", "busy");
  terminalOutput.textContent = "";

  try {
    const response = await fetch("/api/ssh/connect", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(buildRequestPayload()),
    });

    const result = await response.json();
    if (!result.ok) {
      const message = "Error: " + result.error;
      setStatus(message, "error");
      showToast(message, "error");
      return;
    }

    sshConnected = true;
    updateSSHUI();
    terminalInput.focus();
    setStatus(result.message || "SSH connected.", "success");
    showToast(result.message || "SSH connected.", "success");
    document.querySelector(".panel--console").scrollIntoView({ behavior: "smooth", block: "nearest" });
  } catch (error) {
    const message = "Error: " + error.message;
    setStatus(message, "error");
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
      setStatus(message, "error");
      showToast(message, "error");
      return;
    }

    terminalInput.value = "";
    setStatus("Command sent.", "neutral");
  } catch (error) {
    const message = "Error: " + error.message;
    setStatus(message, "error");
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
      setStatus(message, "error");
      showToast(message, "error");
      return;
    }

    sshConnected = false;
    updateSSHUI();
    setStatus(result.message || "SSH disconnected.", "neutral");
    showToast(result.message || "SSH disconnected.", "success");
  } catch (error) {
    const message = "Error: " + error.message;
    setStatus(message, "error");
    showToast(message, "error");
  }
}

async function runAction(url, busyText, options = {}) {
  setBusy(true);
  setStatus(busyText, "busy");
  const historyEntry = options.recordHistory ? buildHistoryEntry() : null;

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(buildRequestPayload()),
    });

    const result = await response.json();
    if (!result.ok) {
      const message = "Error: " + result.error;
      setStatus(message, "error");
      showToast(message, "error");
      return;
    }

    if (historyEntry) {
      rememberTransfer(historyEntry);
    }

    setStatus(result.message || "Done.", "success");
    showToast(result.message || "Done.", "success");
  } catch (error) {
    const message = "Error: " + error.message;
    setStatus(message, "error");
    showToast(message, "error");
  } finally {
    setBusy(false);
  }
}

function setBusy(busy) {
  testBtn.disabled = busy;
  sshBtn.disabled = busy;
  downloadBtn.disabled = busy;
  remoteCopyBtn.disabled = busy;
  heroSshBtn.disabled = busy;
  heroDownloadBtn.disabled = busy;
  heroRemoteCopyBtn.disabled = busy;
}

function updateSSHUI() {
  sshBtn.textContent = sshConnected ? "Reconnect SSH" : "Connect SSH";
  heroSshBtn.textContent = sshConnected ? "Reconnect SSH" : "SSH Console";
  heroSshBtn.classList.toggle("shortcut-btn--active", sshConnected);
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
    setStatus("Log stream disconnected.", "error");
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
    const eventName = element.type === "checkbox" || element.tagName === "SELECT" ? "change" : "input";
    element.addEventListener(eventName, saveForm);
  }
}

function saveForm() {
  const payload = {};
  for (const fieldId of fields) {
    const element = document.getElementById(fieldId);
    payload[fieldId] = element.type === "checkbox" ? element.checked : element.value;
  }
  localStorage.setItem(FORM_STORAGE_KEY, JSON.stringify(payload));
}

function restoreForm() {
  const raw = localStorage.getItem(FORM_STORAGE_KEY);
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
    localStorage.removeItem(FORM_STORAGE_KEY);
  }
}

function loadProfiles() {
  try {
    const raw = localStorage.getItem(PROFILE_STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch (_error) {
    localStorage.removeItem(PROFILE_STORAGE_KEY);
    return [];
  }
}

function persistProfiles() {
  localStorage.setItem(PROFILE_STORAGE_KEY, JSON.stringify(profiles));
}

function loadHistory() {
  try {
    const raw = localStorage.getItem(HISTORY_STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch (_error) {
    localStorage.removeItem(HISTORY_STORAGE_KEY);
    return [];
  }
}

function persistHistory() {
  localStorage.setItem(HISTORY_STORAGE_KEY, JSON.stringify(transferHistory));
}

function formatTime(iso) {
  try {
    return new Intl.DateTimeFormat(undefined, {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
    }).format(new Date(iso));
  } catch (_error) {
    return iso;
  }
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}
