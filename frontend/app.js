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

document.addEventListener("DOMContentLoaded", async () => {
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

  // Role badge + show relevant tabs
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
  if (isBarberOrAdmin()) setupScheduleView();
  if (isAdmin()) setupStaffPanel();
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
  const barbers = await api("GET", "/barbers");
  const barberSel = document.getElementById("barberPicker");
  picker.min = today;
  picker.value = today;
  picker.onchange = () => { selectedSlot = null; document.getElementById("bookingConfirm").classList.add("hidden"); loadSlots(picker.value); };

  barbers.forEach(b => {
    const opt = document.createElement("option:");
    opt.value = b.userId;
    opt.dataset.name = b.name || b.email;
    opt.textContent = b.name || b.email;
    barberSel.appendChild(opt);
  });

  try {
    const svcs = await api("GET", "/services");
    const sel = document.getElementById("servicePicker");
    svcs.sort((a, b) => a.name.localeCompare(b.name)).forEach(s => {
      const opt = document.createElement("option");
      opt.value = s.id;
      opt.textContent = `${s.name} — ${s.price} (${s.duration} min)`;
      sel.appendChild(opt);
    });
    sel.onchange = () => { selectedSlot = null; updateConfirm(); loadSlots(picker.value); };
  } catch { showToast("Could not load services", true); }

  document.getElementById("confirmBookBtn").onclick = confirmBooking;
  loadSlots(today);
}

async function loadSlots(date) {
  const slotsSection = document.getElementById("slotsSection");
  const grid = document.getElementById("slotsGrid");
  try {
    const slots = await api("GET", `/slots?date=${date}`);
    grid.innerHTML = "";
    slotsSection.classList.remove("hidden");
    if (slots.length === 0) { grid.innerHTML = `<p class="muted">Shop is closed on this day.</p>`; return; }
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
  if (selectedSlot && serviceId) {
    confirmDiv.classList.remove("hidden");
    document.getElementById("confirmSummary").innerHTML = `<strong>${serviceText}</strong><br>${formatDate(selectedSlot.date)} at ${formatTime(selectedSlot.timeSlot)}`;
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
  if (!selectedSlot || !serviceId) return;
  try {
    await api("POST", "/appointments", { date: selectedSlot.date, timeSlot: selectedSlot.timeSlot, service: serviceId, notes });
    showToast("Appointment booked!");
    selectedSlot = null;
    document.getElementById("bookingConfirm").classList.add("hidden");
    document.getElementById("bookingNotes").value = "";
    document.querySelectorAll(".slot-btn").forEach(b => b.classList.remove("selected"));
    loadSlots(document.getElementById("datePicker").value);
    loadMyAppointments();
  } catch (err) { showToast(err.message, true); }

  await api("POST", "/appointments", {
    date: selectedSlot.date,
    timeSlot: selectedSlot.timeSlot,
    service: serviceId,
    notes,
    barberId,
    barberName
  });

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
      <div class="appt-service">${capitalize(appt.service.replace(/_/g, " "))}</div>
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
  try {
    const appts = await api("GET", `/admin/appointments?date=${date}`);
    if (appts.length === 0) { list.innerHTML = `<p class="muted">No appointments for this day.</p>`; return; }
    const active = appts.filter(a => a.status !== "cancelled").sort((a, b) => a.timeSlot.localeCompare(b.timeSlot));
    if (active.length === 0) { list.innerHTML = `<p class="muted">No active appointments for this day.</p>`; return; }
    list.innerHTML = "";
    active.forEach(a => {
      const row = document.createElement("div");
      row.className = "schedule-row";
      row.innerHTML = `
        <div class="sched-time">${formatTime(a.timeSlot)}</div>
        <div class="sched-info">
          <div class="sched-customer">${a.userName || a.userEmail}</div>
          <div class="sched-service">${capitalize(a.service.replace(/_/g, " "))}</div>
          ${a.notes ? `<div class="appt-notes">${a.notes}</div>` : ""}
        </div>
        <span class="appt-status ${a.status}">${a.status}</span>`;
      list.appendChild(row);
    });
  } catch (err) { list.innerHTML = `<p class="muted error">Could not load schedule.</p>`; }
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

// ── API client ────────────────────────────────────────────────────────────────

async function api(method, path, body) {
  const opts = { method, headers: { "Content-Type": "application/json" } };
  const token = getToken();
  if (token) opts.headers["Authorization"] = `Bearer ${token}`;
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(API + path, opts);
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
