const CFG = window.SCHEDPRO_CONFIG || {};
const API = CFG.apiUrl || "";

// ── Auth (OAuth2 Authorization Code + PKCE) ───────────────────────────────────

function base64urlEncode(buffer) {
  return btoa(String.fromCharCode(...new Uint8Array(buffer)))
    .replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

function generateVerifier() {
  const buf = new Uint8Array(32);
  crypto.getRandomValues(buf);
  return base64urlEncode(buf);
}

async function generateChallenge(verifier) {
  const buf = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  return base64urlEncode(buf);
}

async function startLogin() {
  const verifier = generateVerifier();
  const challenge = await generateChallenge(verifier);
  sessionStorage.setItem("pkce_verifier", verifier);
  const params = new URLSearchParams({
    response_type: "code",
    client_id: CFG.cognitoClientId,
    redirect_uri: CFG.cognitoRedirectUri,
    scope: "openid email profile",
    code_challenge_method: "S256",
    code_challenge: challenge,
  });
  window.location.href = `${CFG.cognitoDomain}/oauth2/authorize?${params}`;
}

async function handleCallback(code) {
  const verifier = sessionStorage.getItem("pkce_verifier");
  sessionStorage.removeItem("pkce_verifier");
  const res = await fetch(`${CFG.cognitoDomain}/oauth2/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "authorization_code",
      client_id: CFG.cognitoClientId,
      redirect_uri: CFG.cognitoRedirectUri,
      code,
      code_verifier: verifier,
    }),
  });
  if (!res.ok) { showToast("Login failed. Please try again.", true); return; }
  const tokens = await res.json();
  sessionStorage.setItem("access_token", tokens.access_token);
  sessionStorage.setItem("id_token", tokens.id_token);
  window.history.replaceState({}, "", window.location.pathname);
  initApp();
}

function getToken() { return sessionStorage.getItem("id_token"); }

function logout() {
  sessionStorage.clear();
  const params = new URLSearchParams({ client_id: CFG.cognitoClientId, logout_uri: CFG.cognitoRedirectUri });
  window.location.href = `${CFG.cognitoDomain}/logout?${params}`;
}

function parseIdToken() {
  const token = sessionStorage.getItem("id_token");
  if (!token) return null;
  try { return JSON.parse(atob(token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/"))); }
  catch { return null; }
}

function getUserGroups() {
  const claims = parseIdToken();
  if (!claims) return [];
  const g = claims["cognito:groups"];
  if (!g) return [];
  return Array.isArray(g) ? g : [g];
}

function isBarberOrAdmin() { const g = getUserGroups(); return g.includes("barbers") || g.includes("admins"); }
function isAdmin() { return getUserGroups().includes("admins"); }

// ── Init ──────────────────────────────────────────────────────────────────────

function applyBranding() {
  const name = CFG.siteName || "Book a Haircut";
  document.getElementById("pageTitle").textContent = `${name} — Book a Haircut`;
  document.getElementById("brandName").textContent = name;
  document.getElementById("shareQrTitle").textContent = `Share ${name}`;
}

document.addEventListener("DOMContentLoaded", async () => {
  applyBranding();
  setupSiteQr();
  const code = new URLSearchParams(window.location.search).get("code");
  if (code) { await handleCallback(code); return; }
  if (getToken()) { initApp(); } else { showLoggedOut(); }
});

function showLoggedOut() {
  document.getElementById("hero").classList.remove("hidden");
  document.getElementById("loginBtn").classList.remove("hidden");
  document.getElementById("userMenu").classList.add("hidden");
  document.getElementById("heroLoginBtn").onclick = startLogin;
  document.getElementById("loginBtn").onclick = startLogin;
}

function initApp() {
  const claims = parseIdToken();
  if (!claims) { showLoggedOut(); return; }

  document.getElementById("hero").classList.add("hidden");
  document.getElementById("app").classList.remove("hidden");
  document.getElementById("tabNav").classList.remove("hidden");
  document.getElementById("loginBtn").classList.add("hidden");
  document.getElementById("userMenu").classList.remove("hidden");
  document.getElementById("userName").textContent = claims.name || claims.email;
  document.getElementById("logoutBtn").onclick = logout;

  const badge = document.getElementById("userBadge");
  if (isAdmin()) {
    badge.textContent = "Admin";
    badge.className = "role-badge admin";
    document.querySelectorAll(".barber-tab, .admin-tab").forEach(t => t.classList.remove("hidden"));
  } else if (isBarberOrAdmin()) {
    badge.textContent = "Barber";
    badge.className = "role-badge barber";
    document.querySelectorAll(".barber-tab").forEach(t => t.classList.remove("hidden"));
  } else {
    badge.textContent = "Customer";
    badge.className = "role-badge customer";
  }

  setupTabs();
  setupBookingForm();
  loadMyAppointments();
  if (isBarberOrAdmin()) { setupScheduleView(); setupSettingsPanel(); }
  if (isAdmin()) { setupStaffPanel(); setupStatsPanel(); }
}

// ── Tabs ──────────────────────────────────────────────────────────────────────

function setupTabs() {
  document.querySelectorAll(".tab-btn").forEach(btn => {
    btn.onclick = () => {
      document.querySelectorAll(".tab-btn").forEach(b => b.classList.remove("active"));
      document.querySelectorAll(".tab-panel").forEach(p => p.classList.add("hidden"));
      btn.classList.add("active");
      document.getElementById("tab-" + btn.dataset.tab).classList.remove("hidden");
    };
  });
}

// ── Booking ───────────────────────────────────────────────────────────────────

let selectedSlot = null;

async function setupBookingForm() {
  const today = new Date().toISOString().slice(0, 10);
  const picker = document.getElementById("datePicker");
  const barberSel = document.getElementById("barberPicker");
  const serviceSel = document.getElementById("servicePicker");
  picker.min = today;
  picker.value = today;

  // Reload slots when date changes (only if a barber is already selected)
  picker.onchange = () => {
    selectedSlot = null;
    document.getElementById("bookingConfirm").classList.add("hidden");
    if (barberSel.value) loadSlots(picker.value, barberSel.value);
  };

  try {
    const barbers = await api("GET", "/barbers");
    barbers.forEach(b => {
      const opt = document.createElement("option");
      opt.value = b.userId;
      opt.dataset.name = b.name || b.email;
      opt.textContent = b.name || b.email;
      barberSel.appendChild(opt);
    });
  } catch { showToast("Could not load barbers", true); }

  // When barber changes: reload their services and slots
  barberSel.onchange = async () => {
    selectedSlot = null;
    document.getElementById("bookingConfirm").classList.add("hidden");
    document.getElementById("slotsSection").classList.add("hidden");
    serviceSel.innerHTML = `<option value="">— choose a service —</option>`;

    const barberID = barberSel.value;
    if (!barberID) return;

    try {
      const settings = await api("GET", `/barbers/${barberID}/settings`);
      (settings.services || []).sort((a, b) => a.name.localeCompare(b.name)).forEach(s => {
        const opt = document.createElement("option");
        opt.value = s.id;
        opt.textContent = `${s.name} — ${s.price} (${s.duration} min)`;
        serviceSel.appendChild(opt);
      });
    } catch { showToast("Could not load barber's services", true); }

    loadSlots(picker.value, barberID);
  };

  serviceSel.onchange = () => { selectedSlot = null; updateConfirm(); };
  document.getElementById("confirmBookBtn").onclick = confirmBooking;
}

async function loadSlots(date, barberID) {
  const slotsSection = document.getElementById("slotsSection");
  const grid = document.getElementById("slotsGrid");
  if (!barberID) { slotsSection.classList.add("hidden"); return; }
  try {
    const slots = await api("GET", `/slots?date=${date}&barberId=${barberID}`);
    grid.innerHTML = "";
    slotsSection.classList.remove("hidden");
    if (slots.length === 0) { grid.innerHTML = `<p class="muted">No availability on this day.</p>`; return; }
    slots.forEach(s => {
      const btn = document.createElement("button");
      btn.className = "slot-btn" + (s.available ? "" : " unavailable");
      btn.textContent = formatTime(s.timeSlot);
      btn.disabled = !s.available;
      btn.onclick = () => selectSlot(s, btn);
      grid.appendChild(btn);
    });
  } catch { slotsSection.classList.add("hidden"); }
}

function selectSlot(slot, btn) {
  document.querySelectorAll(".slot-btn").forEach(b => b.classList.remove("selected"));
  btn.classList.add("selected");
  selectedSlot = slot;
  updateConfirm();
}

function updateConfirm() {
  const confirmDiv = document.getElementById("bookingConfirm");
  const serviceId = document.getElementById("servicePicker").value;
  const serviceText = document.getElementById("servicePicker").selectedOptions[0]?.text;
  const barberName = document.getElementById("barberPicker").selectedOptions[0]?.dataset.name || "";
  if (selectedSlot && serviceId) {
    confirmDiv.classList.remove("hidden");
    document.getElementById("confirmSummary").innerHTML =
      `<strong>${serviceText}</strong> with ${barberName}<br>${formatDate(selectedSlot.date)} at ${formatTime(selectedSlot.timeSlot)}`;
  } else {
    confirmDiv.classList.add("hidden");
  }
}

async function confirmBooking() {
  const serviceId = document.getElementById("servicePicker").value;
  const notes = document.getElementById("bookingNotes").value;
  const barberOpt = document.getElementById("barberPicker").selectedOptions[0];
  const barberId = barberOpt?.value || "";
  const barberName = barberOpt?.dataset.name || "";
  if (!selectedSlot || !serviceId || !barberId) return;
  try {
    await api("POST", "/appointments", {
      date: selectedSlot.date,
      timeSlot: selectedSlot.timeSlot,
      service: serviceId,
      notes,
      barberId,
      barberName,
    });
    showToast("Appointment booked!");
    selectedSlot = null;
    document.getElementById("bookingConfirm").classList.add("hidden");
    document.getElementById("bookingNotes").value = "";
    document.querySelectorAll(".slot-btn").forEach(b => b.classList.remove("selected"));
    loadSlots(document.getElementById("datePicker").value, barberId);
    loadMyAppointments();
    // Show payment QR if barber has handles configured
    try {
      const settings = await api("GET", `/barbers/${barberId}/settings`);
      if (settings.venmoHandle || settings.cashAppHandle) {
        showPaymentQR(barberName, settings.venmoHandle, settings.cashAppHandle);
      }
    } catch { /* non-critical, skip */ }
  } catch (err) { showToast(err.message, true); }
}

// ── My Appointments ───────────────────────────────────────────────────────────

async function loadMyAppointments() {
  const list = document.getElementById("appointmentsList");
  try {
    const appts = await api("GET", "/appointments/me");
    if (appts.length === 0) { list.innerHTML = `<p class="muted">No upcoming appointments.</p>`; return; }
    list.innerHTML = "";
    appts.sort((a, b) => (a.date + a.timeSlot).localeCompare(b.date + b.timeSlot)).forEach(a => list.appendChild(appointmentCard(a)));
  } catch { list.innerHTML = `<p class="muted error">Could not load appointments.</p>`; }
}

function appointmentCard(appt) {
  const isCancelled = appt.status === "cancelled";
  const el = document.createElement("div");
  el.className = "appt-card" + (isCancelled ? " cancelled" : "");
  el.innerHTML = `
    <div class="appt-info">
      <div class="appt-datetime">${formatDate(appt.date)} &mdash; ${formatTime(appt.timeSlot)}</div>
      <div class="appt-service">${appt.barberName ? `with ${appt.barberName} &mdash; ` : ""}${capitalize(appt.service.replace(/_/g, " "))}</div>
      ${appt.notes ? `<div class="appt-notes">${appt.notes}</div>` : ""}
    </div>
    <div class="appt-actions">
      <span class="appt-status ${appt.status}">${appt.status}</span>
      ${!isCancelled ? `<button class="btn-danger-sm" data-id="${appt.id}">Cancel</button>` : ""}
    </div>`;
  if (!isCancelled) el.querySelector("button").onclick = () => cancelMyAppointment(appt.id);
  return el;
}

async function cancelMyAppointment(id) {
  if (!confirm("Cancel this appointment?")) return;
  try {
    await api("PUT", `/appointments/${id}/cancel`);
    showToast("Appointment cancelled");
    loadMyAppointments();
  } catch (err) { showToast(err.message, true); }
}

// ── Barber: Day Schedule ──────────────────────────────────────────────────────

function setupScheduleView() {
  const picker = document.getElementById("schedDatePicker");
  const today = new Date().toISOString().slice(0, 10);
  picker.value = today;
  picker.onchange = () => loadSchedule(picker.value);
  loadSchedule(today);
}

async function loadSchedule(date) {
  const list = document.getElementById("scheduleList");
  list.innerHTML = `<p class="muted">Loading...</p>`;
  const claims = parseIdToken();
  const myID = claims?.sub;
  try {
    const appts = await api("GET", `/admin/appointments?date=${date}`);
    if (appts.length === 0) { list.innerHTML = `<p class="muted">No appointments for this day.</p>`; return; }
    const active = appts.filter(a => a.status !== "cancelled").sort((a, b) => a.timeSlot.localeCompare(b.timeSlot));
    if (active.length === 0) { list.innerHTML = `<p class="muted">No active appointments for this day.</p>`; return; }
    list.innerHTML = "";
    active.forEach(a => {
      const row = document.createElement("div");
      row.className = "schedule-row";
      const canComplete = a.status === "booked" && (isAdmin() || a.barberId === myID);
      row.innerHTML = `
        <div class="sched-time">${formatTime(a.timeSlot)}</div>
        <div class="sched-info">
          <div class="sched-customer">${a.userName || a.userEmail}</div>
          <div class="sched-service">${capitalize(a.service.replace(/_/g, " "))}</div>
          ${a.notes ? `<div class="appt-notes">${a.notes}</div>` : ""}
        </div>
        <span class="appt-status ${a.status}">${a.status}</span>
        ${canComplete ? `<button class="btn-primary-sm">Mark Complete</button>` : ""}`;
      if (canComplete) row.querySelector("button").onclick = () => markComplete(a.id, date);
      list.appendChild(row);
    });
  } catch (err) { list.innerHTML = `<p class="muted error">Could not load schedule.</p>`; }
}

async function markComplete(id, date) {
  try {
    await api("PUT", `/appointments/${id}/complete`);
    showToast("Appointment marked complete");
    loadSchedule(date);
  } catch (err) { showToast(err.message, true); }
}

// ── Barber: My Settings ───────────────────────────────────────────────────────

const DAYS = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"];
let settingsServices = []; // local copy of the barber's services being edited

async function setupSettingsPanel() {
  // Build schedule rows
  const tbody = document.getElementById("scheduleRows");
  DAYS.forEach(day => {
    const tr = document.createElement("tr");
    tr.id = `day-${day}`;
    tr.innerHTML = `
      <td>${day}</td>
      <td><input type="checkbox" id="open-${day}" /></td>
      <td><input type="time" id="open-time-${day}" value="09:00" /></td>
      <td><input type="time" id="close-time-${day}" value="17:00" /></td>`;
    tbody.appendChild(tr);
  });

  document.getElementById("addSvcBtn").onclick = addServiceRow;
  document.getElementById("saveSettingsBtn").onclick = saveSettings;

  // Load existing settings
  const claims = parseIdToken();
  if (!claims) return;
  try {
    const settings = await api("GET", `/barbers/${claims.sub}/settings`);
    // Populate schedule
    DAYS.forEach(day => {
      const s = settings.schedule?.[day];
      if (s) {
        document.getElementById(`open-${day}`).checked = s.open;
        document.getElementById(`open-time-${day}`).value = `${String(s.openHour).padStart(2,"0")}:${String(s.openMinute).padStart(2,"0")}`;
        document.getElementById(`close-time-${day}`).value = `${String(s.closeHour).padStart(2,"0")}:${String(s.closeMinute).padStart(2,"0")}`;
      }
    });
    // Populate services
    settingsServices = settings.services || [];
    renderServiceRows();
    // Populate payment handles
    document.getElementById("venmoHandle").value = settings.venmoHandle || "";
    document.getElementById("cashAppHandle").value = settings.cashAppHandle || "";
  } catch { showToast("Could not load your settings", true); }
}

function renderServiceRows() {
  const list = document.getElementById("servicesList");
  list.innerHTML = "";
  settingsServices.forEach((svc, i) => {
    const row = document.createElement("div");
    row.className = "appt-card";
    row.style.marginBottom = "0.5rem";
    row.innerHTML = `
      <div class="appt-info">
        <div class="appt-datetime">${svc.name}</div>
        <div class="appt-service">${svc.price} &mdash; ${svc.duration} min</div>
      </div>
      <button class="btn-danger-sm">Remove</button>`;
    row.querySelector("button").onclick = () => {
      settingsServices.splice(i, 1);
      renderServiceRows();
    };
    list.appendChild(row);
  });
  if (settingsServices.length === 0) {
    list.innerHTML = `<p class="muted">No services added yet.</p>`;
  }
}

function addServiceRow() {
  const name = document.getElementById("newSvcName").value.trim();
  const duration = parseInt(document.getElementById("newSvcDuration").value, 10);
  const price = document.getElementById("newSvcPrice").value.trim();
  if (!name || !duration || duration < 1) { showToast("Name and duration are required", true); return; }
  settingsServices.push({ id: crypto.randomUUID(), name, duration, price: price || "TBD" });
  document.getElementById("newSvcName").value = "";
  document.getElementById("newSvcDuration").value = "";
  document.getElementById("newSvcPrice").value = "";
  renderServiceRows();
}

async function saveSettings() {
  const schedule = {};
  DAYS.forEach(day => {
    const open = document.getElementById(`open-${day}`).checked;
    const [openHour, openMinute] = document.getElementById(`open-time-${day}`).value.split(":").map(Number);
    const [closeHour, closeMinute] = document.getElementById(`close-time-${day}`).value.split(":").map(Number);
    schedule[day] = { open, openHour, openMinute, closeHour, closeMinute };
  });
  const venmoHandle = document.getElementById("venmoHandle").value.trim();
  const cashAppHandle = document.getElementById("cashAppHandle").value.trim();
  try {
    await api("PUT", "/barbers/me/settings", { schedule, services: settingsServices, venmoHandle, cashAppHandle });
    showToast("Settings saved!");
  } catch (err) { showToast(err.message, true); }
}

// ── Admin: Staff Management ───────────────────────────────────────────────────

function setupStaffPanel() {
  document.getElementById("addBarberBtn").onclick = addBarber;
  loadBarbers();
}

async function loadBarbers() {
  const list = document.getElementById("barbersList");
  try {
    const barbers = await api("GET", "/admin/barbers");
    if (barbers.length === 0) { list.innerHTML = `<p class="muted">No barbers added yet.</p>`; return; }
    list.innerHTML = "";
    barbers.forEach(b => {
      const row = document.createElement("div");
      row.className = "appt-card";
      row.innerHTML = `
        <div class="appt-info">
          <div class="appt-datetime">${b.name || "(no name)"}</div>
          <div class="appt-service">${b.email}</div>
        </div>
        <button class="btn-danger-sm" data-id="${b.userId}">Remove</button>`;
      row.querySelector("button").onclick = () => removeBarber(b.userId, b.email);
      list.appendChild(row);
    });
  } catch { list.innerHTML = `<p class="muted error">Could not load staff.</p>`; }
}

async function addBarber() {
  const email = document.getElementById("addBarberEmail").value.trim();
  if (!email) return;
  try {
    await api("POST", "/admin/barbers", { email });
    showToast("Barber added successfully");
    document.getElementById("addBarberEmail").value = "";
    loadBarbers();
  } catch (err) { showToast(err.message, true); }
}

async function removeBarber(userId, email) {
  if (!confirm(`Remove ${email} as a barber?`)) return;
  try {
    await api("DELETE", `/admin/barbers/${userId}`);
    showToast("Barber removed");
    loadBarbers();
  } catch (err) { showToast(err.message, true); }
}

// ── Admin: Stats ──────────────────────────────────────────────────────────────

function setupStatsPanel() {
  const picker = document.getElementById("statsMonthPicker");
  const now = new Date();
  const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  picker.value = currentMonth;
  picker.onchange = () => loadStats(picker.value);
  loadStats(currentMonth);
}

async function loadStats(month) {
  const list = document.getElementById("statsList");
  list.innerHTML = `<p class="muted">Loading...</p>`;
  try {
    const stats = await api("GET", `/admin/stats?month=${month}`);
    if (stats.length === 0) { list.innerHTML = `<p class="muted">No appointments for this month.</p>`; return; }
    const totalBooked = stats.reduce((sum, s) => sum + s.booked, 0);
    list.innerHTML = "";
    stats.forEach(s => {
      const row = document.createElement("div");
      row.className = "appt-card";
      row.innerHTML = `
        <div class="appt-info">
          <div class="appt-datetime">${s.barberName || "(unknown)"}</div>
          <div class="appt-service">${s.cancelled} cancelled</div>
        </div>
        <span class="appt-status booked" style="font-size:1.1rem">${s.booked}</span>`;
      list.appendChild(row);
    });
    const totalRow = document.createElement("div");
    totalRow.className = "appt-card";
    totalRow.style.fontWeight = "700";
    totalRow.innerHTML = `<div class="appt-info"><div class="appt-datetime">Total</div></div><span>${totalBooked}</span>`;
    list.appendChild(totalRow);
  } catch (err) { list.innerHTML = `<p class="muted error">Could not load stats.</p>`; }
}

// ── Site QR code ──────────────────────────────────────────────────────────────

function setupSiteQr() {
  const url = window.location.origin + window.location.pathname.replace(/\/$/, "");
  let qrRendered = false;

  document.getElementById("shareQrBtn").onclick = () => {
    const modal = document.getElementById("siteQrModal");
    modal.classList.remove("hidden");
    document.getElementById("siteQrUrl").textContent = url;

    if (!qrRendered) {
      new QRCode(document.getElementById("siteQrCode"), {
        text: url,
        width: 200,
        height: 200,
        correctLevel: QRCode.CorrectLevel.M,
      });
      qrRendered = true;
    }

    // Show native share button if supported (works great on mobile)
    if (navigator.share) {
      const btn = document.getElementById("siteQrShare");
      btn.style.display = "";
      btn.onclick = () => navigator.share({ title: `${CFG.siteName || "Book a Haircut"} — Book a Haircut`, url });
    }
  };

  document.getElementById("siteQrClose").onclick = () =>
    document.getElementById("siteQrModal").classList.add("hidden");
  document.getElementById("siteQrModal").onclick = (e) => {
    if (e.target === document.getElementById("siteQrModal"))
      document.getElementById("siteQrModal").classList.add("hidden");
  };

  document.getElementById("siteQrDownload").onclick = () => {
    const canvas = document.querySelector("#siteQrCode canvas");
    if (!canvas) return;
    const a = document.createElement("a");
    a.href = canvas.toDataURL("image/png");
    a.download = "the-chair-qr.png";
    a.click();
  };
}

// ── Payment QR modal ──────────────────────────────────────────────────────────

function showPaymentQR(barberName, venmoHandle, cashAppHandle) {
  const modal = document.getElementById("qrModal");
  const codesDiv = document.getElementById("qrModalCodes");
  document.getElementById("qrModalTitle").textContent = `Pay ${barberName}`;
  codesDiv.innerHTML = "";

  function makeQR(label, url) {
    const wrap = document.createElement("div");
    wrap.className = "qr-code-wrap";
    const lbl = document.createElement("div");
    lbl.className = "qr-label";
    lbl.textContent = label;
    const canvas = document.createElement("div");
    wrap.appendChild(lbl);
    wrap.appendChild(canvas);
    codesDiv.appendChild(wrap);
    new QRCode(canvas, { text: url, width: 180, height: 180, correctLevel: QRCode.CorrectLevel.M });
  }

  if (venmoHandle) {
    const handle = venmoHandle.replace(/^@/, "");
    makeQR(`Venmo — @${handle}`, `https://venmo.com/${handle}`);
  }
  if (cashAppHandle) {
    const tag = cashAppHandle.startsWith("$") ? cashAppHandle : `$${cashAppHandle}`;
    makeQR(`Cash App — ${tag}`, `https://cash.app/${tag}`);
  }

  modal.classList.remove("hidden");
  document.getElementById("qrModalClose").onclick = () => modal.classList.add("hidden");
  modal.onclick = (e) => { if (e.target === modal) modal.classList.add("hidden"); };
}

// ── API client ────────────────────────────────────────────────────────────────

async function api(method, path, body) {
  const opts = { method, headers: { "Content-Type": "application/json" } };
  const token = getToken();
  if (token) opts.headers["Authorization"] = `Bearer ${token}`;
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(API + path, opts);
  if (res.status === 401) { sessionStorage.clear(); showLoggedOut(); throw new Error("Session expired. Please sign in again."); }
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Request failed");
  return data;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatTime(t) {
  const [h, m] = t.split(":").map(Number);
  return `${h % 12 || 12}:${String(m).padStart(2, "0")} ${h < 12 ? "AM" : "PM"}`;
}

function formatDate(d) {
  return new Date(d + "T12:00:00").toLocaleDateString("en-US", { weekday: "short", month: "short", day: "numeric" });
}

function capitalize(s) { return s.charAt(0).toUpperCase() + s.slice(1); }

function showToast(msg, isError = false) {
  const el = document.getElementById("toast");
  el.textContent = msg;
  el.className = "toast" + (isError ? " error" : "");
  clearTimeout(el._t);
  el._t = setTimeout(() => el.classList.add("hidden"), 3500);
}
