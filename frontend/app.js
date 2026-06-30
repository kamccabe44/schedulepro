// ── Config ───────────────────────────────────────────────────────────────────
// Set API_URL to your deployed API Gateway URL (from SAM deploy output)
const API_URL = window.SCHEDULEPRO_API_URL || "https://YOUR_API_GATEWAY_URL/prod";

// ── State ─────────────────────────────────────────────────────────────────────
let currentDate = new Date();
let view = "month";
let events = [];
let selectedColor = "#3b82f6";

// ── Init ──────────────────────────────────────────────────────────────────────
document.addEventListener("DOMContentLoaded", () => {
  document.getElementById("prevBtn").onclick = () => navigate(-1);
  document.getElementById("nextBtn").onclick = () => navigate(1);
  document.getElementById("todayBtn").onclick = () => { currentDate = new Date(); render(); };
  document.getElementById("viewSelect").onchange = (e) => { view = e.target.value; render(); };
  document.getElementById("newEventBtn").onclick = () => openModal();
  document.getElementById("closeModal").onclick = closeModal;
  document.getElementById("cancelBtn").onclick = closeModal;
  document.getElementById("modalBackdrop").onclick = closeModal;
  document.getElementById("deleteEventBtn").onclick = deleteCurrentEvent;
  document.getElementById("eventForm").onsubmit = saveEvent;
  document.getElementById("colorOptions").onclick = (e) => {
    const swatch = e.target.closest(".color-swatch");
    if (!swatch) return;
    document.querySelectorAll(".color-swatch").forEach(s => s.classList.remove("active"));
    swatch.classList.add("active");
    selectedColor = swatch.dataset.color;
    document.getElementById("color").value = selectedColor;
  };
  render();
});

// ── Navigation ────────────────────────────────────────────────────────────────
function navigate(dir) {
  if (view === "month") currentDate.setMonth(currentDate.getMonth() + dir);
  else if (view === "week") currentDate.setDate(currentDate.getDate() + dir * 7);
  else currentDate.setDate(currentDate.getDate() + dir);
  render();
}

// ── Render ────────────────────────────────────────────────────────────────────
async function render() {
  const { start, end } = getViewRange();
  await fetchEvents(start, end);
  updatePeriodLabel();

  const container = document.getElementById("calendarContainer");
  container.innerHTML = "";
  const cal = document.createElement("div");
  cal.className = "calendar";
  if (view === "month") cal.appendChild(buildMonthView());
  else if (view === "week") cal.appendChild(buildWeekView());
  else cal.appendChild(buildDayView());
  container.appendChild(cal);
}

function getViewRange() {
  const d = new Date(currentDate);
  if (view === "month") {
    const start = new Date(d.getFullYear(), d.getMonth(), 1);
    const end = new Date(d.getFullYear(), d.getMonth() + 1, 0);
    return { start: fmt(start), end: fmt(end) };
  } else if (view === "week") {
    const dow = d.getDay();
    const start = new Date(d); start.setDate(d.getDate() - dow);
    const end = new Date(start); end.setDate(start.getDate() + 6);
    return { start: fmt(start), end: fmt(end) };
  } else {
    return { start: fmt(d), end: fmt(d) };
  }
}

function updatePeriodLabel() {
  const el = document.getElementById("currentPeriod");
  const d = currentDate;
  if (view === "month") {
    el.textContent = d.toLocaleDateString("en-US", { month: "long", year: "numeric" });
  } else if (view === "week") {
    const dow = d.getDay();
    const start = new Date(d); start.setDate(d.getDate() - dow);
    const end = new Date(start); end.setDate(start.getDate() + 6);
    el.textContent = `${start.toLocaleDateString("en-US", { month: "short", day: "numeric" })} – ${end.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" })}`;
  } else {
    el.textContent = d.toLocaleDateString("en-US", { weekday: "long", month: "long", day: "numeric", year: "numeric" });
  }
}

// ── Month View ────────────────────────────────────────────────────────────────
function buildMonthView() {
  const frag = document.createDocumentFragment();
  const grid = document.createElement("div");
  grid.className = "month-grid";

  ["Sun","Mon","Tue","Wed","Thu","Fri","Sat"].forEach(d => {
    const h = document.createElement("div");
    h.className = "day-header";
    h.textContent = d;
    grid.appendChild(h);
  });

  const year = currentDate.getFullYear(), month = currentDate.getMonth();
  const firstDay = new Date(year, month, 1).getDay();
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const daysInPrev = new Date(year, month, 0).getDate();
  const today = fmt(new Date());

  const totalCells = Math.ceil((firstDay + daysInMonth) / 7) * 7;

  for (let i = 0; i < totalCells; i++) {
    let cellDate, isOther = false;
    if (i < firstDay) { cellDate = new Date(year, month - 1, daysInPrev - firstDay + i + 1); isOther = true; }
    else if (i >= firstDay + daysInMonth) { cellDate = new Date(year, month + 1, i - firstDay - daysInMonth + 1); isOther = true; }
    else { cellDate = new Date(year, month, i - firstDay + 1); }

    const cell = document.createElement("div");
    cell.className = "day-cell" + (isOther ? " other-month" : "") + (fmt(cellDate) === today ? " today" : "");
    cell.onclick = (e) => { if (e.target === cell || e.target.className === "day-number") openModal(null, fmt(cellDate)); };

    const num = document.createElement("div");
    num.className = "day-number";
    num.textContent = cellDate.getDate();
    cell.appendChild(num);

    const dayStr = fmt(cellDate);
    const dayEvents = events.filter(ev => ev.startTime?.startsWith(dayStr)).slice(0, 3);
    dayEvents.forEach(ev => { cell.appendChild(makeChip(ev)); });
    if (events.filter(ev => ev.startTime?.startsWith(dayStr)).length > 3) {
      const more = document.createElement("div");
      more.className = "more-link";
      more.textContent = `+${events.filter(ev => ev.startTime?.startsWith(dayStr)).length - 3} more`;
      cell.appendChild(more);
    }
    grid.appendChild(cell);
  }

  frag.appendChild(grid);
  return frag;
}

// ── Week/Day Views ────────────────────────────────────────────────────────────
function buildWeekView() {
  const dow = currentDate.getDay();
  const days = Array.from({ length: 7 }, (_, i) => {
    const d = new Date(currentDate); d.setDate(currentDate.getDate() - dow + i); return d;
  });
  return buildTimeGrid(days);
}

function buildDayView() {
  return buildTimeGrid([new Date(currentDate)]);
}

function buildTimeGrid(days) {
  const today = fmt(new Date());
  const wrapper = document.createDocumentFragment();
  const grid = document.createElement("div");
  grid.className = "time-grid";

  // Time labels
  const labels = document.createElement("div");
  labels.className = "time-labels";
  labels.appendChild(document.createElement("div")); // spacer for header row
  labels.children[0].style.height = "41px";
  for (let h = 0; h < 24; h++) {
    const lbl = document.createElement("div");
    lbl.className = "time-label";
    lbl.textContent = h === 0 ? "" : `${h % 12 || 12}${h < 12 ? "am" : "pm"}`;
    labels.appendChild(lbl);
  }
  grid.appendChild(labels);

  // Day columns container
  const cols = document.createElement("div");
  cols.className = "day-columns";
  cols.style.gridTemplateColumns = `repeat(${days.length}, 1fr)`;

  days.forEach(day => {
    const dayStr = fmt(day);
    const col = document.createElement("div");
    col.className = "day-column";

    const header = document.createElement("div");
    header.className = "day-col-header" + (dayStr === today ? " today" : "");
    header.textContent = day.toLocaleDateString("en-US", { weekday: "short", month: "numeric", day: "numeric" });
    col.appendChild(header);

    for (let h = 0; h < 24; h++) {
      const slot = document.createElement("div");
      slot.className = "hour-slot";
      slot.onclick = () => openModal(null, `${dayStr}T${String(h).padStart(2,"0")}:00`);
      col.appendChild(slot);
    }

    // Place events
    events.filter(ev => ev.startTime?.startsWith(dayStr)).forEach(ev => {
      const start = new Date(ev.startTime);
      const end = new Date(ev.endTime);
      const topPct = (start.getHours() + start.getMinutes() / 60) / 24 * 100;
      const heightPct = Math.max((end - start) / (24 * 3600000) * 100, 2);
      const chip = document.createElement("div");
      chip.className = "timed-event";
      chip.style.cssText = `background:${ev.color || "#3b82f6"};top:calc(41px + ${topPct}%);height:${heightPct}%`;
      chip.textContent = ev.title;
      chip.onclick = (e) => { e.stopPropagation(); openModal(ev); };
      col.appendChild(chip);
    });

    cols.appendChild(col);
  });
  grid.appendChild(cols);
  wrapper.appendChild(grid);
  return wrapper;
}

function makeChip(ev) {
  const chip = document.createElement("div");
  chip.className = "event-chip";
  chip.style.background = ev.color || "#3b82f6";
  chip.textContent = ev.title;
  chip.onclick = (e) => { e.stopPropagation(); openModal(ev); };
  return chip;
}

// ── Modal ─────────────────────────────────────────────────────────────────────
function openModal(ev = null, dateStr = null) {
  document.getElementById("modal").classList.remove("hidden");
  document.getElementById("modalTitle").textContent = ev ? "Edit Event" : "New Event";
  document.getElementById("deleteEventBtn").classList.toggle("hidden", !ev);

  if (ev) {
    document.getElementById("eventId").value = ev.id;
    document.getElementById("title").value = ev.title;
    document.getElementById("startTime").value = toLocalInput(ev.startTime);
    document.getElementById("endTime").value = toLocalInput(ev.endTime);
    document.getElementById("location").value = ev.location || "";
    document.getElementById("description").value = ev.description || "";
    setColor(ev.color || "#3b82f6");
  } else {
    document.getElementById("eventForm").reset();
    document.getElementById("eventId").value = "";
    const base = dateStr ? new Date(dateStr) : new Date();
    if (!base.getHours()) base.setHours(9);
    const end = new Date(base); end.setHours(base.getHours() + 1);
    document.getElementById("startTime").value = toLocalInput(base.toISOString());
    document.getElementById("endTime").value = toLocalInput(end.toISOString());
    setColor("#3b82f6");
  }
}

function setColor(hex) {
  selectedColor = hex;
  document.getElementById("color").value = hex;
  document.querySelectorAll(".color-swatch").forEach(s => {
    s.classList.toggle("active", s.dataset.color === hex);
  });
}

function closeModal() { document.getElementById("modal").classList.add("hidden"); }

async function saveEvent(e) {
  e.preventDefault();
  const id = document.getElementById("eventId").value;
  const payload = {
    title: document.getElementById("title").value,
    startTime: toISO(document.getElementById("startTime").value),
    endTime: toISO(document.getElementById("endTime").value),
    location: document.getElementById("location").value,
    description: document.getElementById("description").value,
    color: document.getElementById("color").value,
  };

  try {
    if (id) {
      await api("PUT", `/events/${id}`, payload);
      toast("Event updated");
    } else {
      await api("POST", "/events", payload);
      toast("Event created");
    }
    closeModal();
    render();
  } catch (err) {
    toast(err.message, true);
  }
}

async function deleteCurrentEvent() {
  const id = document.getElementById("eventId").value;
  if (!id || !confirm("Delete this event?")) return;
  try {
    await api("DELETE", `/events/${id}`);
    toast("Event deleted");
    closeModal();
    render();
  } catch (err) {
    toast(err.message, true);
  }
}

// ── API ───────────────────────────────────────────────────────────────────────
async function fetchEvents(start, end) {
  try {
    events = await api("GET", `/events?start=${start}&end=${end}`);
  } catch (err) {
    toast("Failed to load events: " + err.message, true);
    events = [];
  }
}

async function api(method, path, body) {
  const opts = { method, headers: { "Content-Type": "application/json" } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(API_URL + path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Request failed");
  return data;
}

// ── Helpers ───────────────────────────────────────────────────────────────────
function fmt(d) { return d.toISOString().slice(0, 10); }

function toLocalInput(iso) {
  const d = new Date(iso);
  const pad = n => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function toISO(localStr) { return new Date(localStr).toISOString(); }

function toast(msg, isError = false) {
  const el = document.getElementById("toast");
  el.textContent = msg;
  el.className = "toast" + (isError ? " error" : "");
  setTimeout(() => el.classList.add("hidden"), 3000);
}
