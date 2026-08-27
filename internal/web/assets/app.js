const state = { aggregate: null, windows: [], conflicts: [], productions: [] };
const $ = (selector) => document.querySelector(selector);

function key(prefix) {
  return `${prefix}-${Date.now()}-${crypto.getRandomValues(new Uint32Array(1))[0]}`;
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
  });
  const contentType = response.headers.get("content-type") || "";
  const body = contentType.includes("json") ? await response.json() : await response.text();
  if (!response.ok) {
    throw new Error(body?.error?.message || `请求失败（${response.status}）`);
  }
  return body;
}

function notify(message, isError = false) {
  const node = $("#notice");
  node.textContent = message;
  node.className = isError ? "show error" : "show";
  clearTimeout(notify.timer);
  notify.timer = setTimeout(() => { node.className = ""; }, 3200);
}

function unwrap(result) {
  return result && Object.prototype.hasOwnProperty.call(result, "value") ? result.value : result;
}

function latestCues() {
  const byID = new Map();
  for (const cue of state.aggregate?.cues || []) {
	if (cue.status === "WITHDRAWN" || cue.withdrawn_at) continue;
    const old = byID.get(cue.id);
    if (!old || cue.version > old.version) byID.set(cue.id, cue);
  }
  return [...byID.values()].sort((a, b) => a.window_start_ms - b.window_start_ms);
}

function render() {
  const aggregate = state.aggregate;
  if (!aggregate) return;
  $("#workspace").hidden = false;
  $("#project-title").textContent = aggregate.production.title;
  $("#project-state").textContent = aggregate.production.state;
  $("#project-revision").textContent = aggregate.production.revision;
  $("#project-duration").textContent = `${(aggregate.production.duration_ms / 1000).toFixed(1)} 秒`;
  document.querySelectorAll(".state-strip span").forEach((node) => {
    node.classList.toggle("active", node.textContent === aggregate.production.state);
  });
  const stateName = aggregate.production.state;
  const timelineEditable = stateName === "DRAFT" || stateName === "TIMELINED";
  const cueEditable = stateName === "TIMELINED" || stateName === "WRITING" || stateName === "REVISING";
  ["#segment-form"].forEach((selector) => { const node = $(selector); if (node) node.querySelectorAll("input,select,textarea,button").forEach((control) => { control.disabled = !timelineEditable; }); });
  ["#finalize-button"].forEach((selector) => { const node = $(selector); if (node) node.disabled = !timelineEditable; });
  ["#cue-form"].forEach((selector) => { const node = $(selector); if (node) node.querySelectorAll("input,select,textarea,button").forEach((control) => { control.disabled = !cueEditable; }); });
  ["#validate-button"].forEach((selector) => { const node = $(selector); if (node) node.disabled = !cueEditable; });
  const rehearsing = stateName === "REHEARSING";
  const reviewing = stateName === "REVIEWING";
  const approved = stateName === "APPROVED";
  ["#rehearse-button"].forEach((selector) => { const node = $(selector); if (node) node.disabled = !rehearsing; });
  ["#accept-button", "#approve-button"].forEach((selector) => { const node = $(selector); if (node) node.disabled = !reviewing; });
  ["#review-form"].forEach((selector) => { const node = $(selector); if (node) node.querySelectorAll("input,select,textarea,button").forEach((control) => { control.disabled = !reviewing; }); });
  ["#release-button"].forEach((selector) => { const node = $(selector); if (node) node.disabled = !approved; });
  renderTimeline();
  renderCues();
  renderReview();
  renderRehearsalForm();
  renderValidation();
  if (aggregate.release) renderRelease();
  if (aggregate.production.state === "APPROVED") loadReleasePreview();
}

function renderTimeline() {
  const node = $("#timeline-output");
  const segments = state.aggregate.segments || [];
  const conflictIDs = new Set(state.conflicts.flatMap((c) => [c.first_id, c.second_id]).filter(Boolean));
  const sceneSelect = $("#scene-select");
  if (sceneSelect) sceneSelect.innerHTML = '<option value="">自动匹配</option>' + segments.filter((s) => s.kind === "SCENE").map((s) => `<option value="${escapeHTML(s.id)}">${escapeHTML(s.label)}</option>`).join("");
  const segmentCards = [...segments].sort((a,b) => a.start_ms-b.start_ms || a.end_ms-b.end_ms || a.id.localeCompare(b.id)).map((s) => `<div id="segment-${escapeHTML(s.id)}" class="card${conflictIDs.has(s.id) ? " conflict" : ""}"><span><strong>${escapeHTML(s.label)}</strong> · ${s.kind}</span><span>${s.start_ms}—${s.end_ms}ms <button data-edit-segment="${escapeHTML(s.id)}" title="编辑区间">编辑</button> <button data-delete-segment="${escapeHTML(s.id)}" title="删除区间">删除</button></span></div>`);
  const windowCards = state.windows.map((w) => `<div class="card"><span>候选窗口 · 场景 ${escapeHTML(w.scene_id)}</span><span>${w.start_ms}—${w.end_ms}ms（${w.usable_ms}ms）</span></div>`);
  const conflictCards = state.conflicts.map((c) => `<div class="card conflict"><span>冲突 <button data-focus-segment="${escapeHTML(c.first_id)}">${escapeHTML(c.first_id)}</button>${c.second_id ? ` ↔ <button data-focus-segment="${escapeHTML(c.second_id)}">${escapeHTML(c.second_id)}</button>` : ""}</span><span>${escapeHTML(c.message)}${c.start_ms !== undefined ? `（${c.start_ms}—${c.end_ms}ms）` : ""}</span></div>`);
  node.innerHTML = [...conflictCards, ...segmentCards, ...windowCards].join("") || '<p class="hint">尚未录入时间轴。</p>';
  document.querySelectorAll("[data-delete-segment]").forEach((button) => { button.disabled = !(state.aggregate.production.state === "DRAFT" || state.aggregate.production.state === "TIMELINED"); button.addEventListener("click", async () => {
    try { await mutate(`/api/productions/${state.aggregate.production.id}/segments/${button.dataset.deleteSegment}`, { expectedRevision: state.aggregate.production.revision, idempotencyKey: key("delete-segment") }, "DELETE"); await loadWindows(); } catch (error) { notify(error.message, true); }
  }); });
  document.querySelectorAll("[data-focus-segment]").forEach((button) => button.addEventListener("click", () => document.getElementById(`segment-${button.dataset.focusSegment}`)?.scrollIntoView({ behavior: "smooth", block: "center" })));
  document.querySelectorAll("[data-edit-segment]").forEach((button) => { button.disabled = !(state.aggregate.production.state === "DRAFT" || state.aggregate.production.state === "TIMELINED"); button.addEventListener("click", () => {
    const segment = segments.find((item) => item.id === button.dataset.editSegment);
    if (!segment) return;
    const form = $("#segment-form");
    form.elements.segment_id.value = segment.id;
    form.elements.kind.value = segment.kind;
    form.elements.scene_id.value = segment.scene_id || "";
    form.elements.start_ms.value = segment.start_ms;
    form.elements.end_ms.value = segment.end_ms;
    form.elements.label.value = segment.label;
    form.scrollIntoView({ behavior: "smooth", block: "center" });
  }); });
}

function renderCues() {
  const cues = latestCues();
  const cueEditable = ["TIMELINED", "WRITING", "REVISING"].includes(state.aggregate.production.state);
  $("#review-cue").innerHTML = cues.map((cue) => `<option value="${escapeHTML(cue.id)}">${escapeHTML(cue.id)} · v${cue.version}</option>`).join("");
  $("#cue-output").innerHTML = cues.map((cue) => {
    const diff = cue.version > 1 ? `<button data-diff="${escapeHTML(cue.id)}" data-version="${cue.version}">对比 v${cue.version - 1}→v${cue.version}</button>` : "";
    return `<div id="cue-${escapeHTML(cue.id)}" class="card"><span><strong>${escapeHTML(cue.id)}</strong> v${cue.version} · ${escapeHTML(cue.text)}<br>${cue.window_start_ms}—${cue.window_end_ms}ms · ${cue.status}</span><span>${diff}<button data-edit-cue="${escapeHTML(cue.id)}">编辑</button><button data-withdraw-cue="${escapeHTML(cue.id)}" title="撤回提示">撤回</button></span></div>`;
  }).join("") || '<p class="hint">完成时间轴后，在候选窗口内保存提示。</p>';
  document.querySelectorAll("[data-diff]").forEach((button) => button.addEventListener("click", showDiff));
  document.querySelectorAll("[data-edit-cue]").forEach((button) => {
    button.disabled = !cueEditable;
    button.addEventListener("click", () => {
      const cue = cues.find((item) => item.id === button.dataset.editCue);
      if (!cue) return;
      const form = $("#cue-form");
      form.elements.cue_id.value = cue.id;
      form.elements.window_start_ms.value = cue.window_start_ms;
      form.elements.window_end_ms.value = cue.window_end_ms;
      form.elements.intent.value = cue.intent;
      form.elements.text.value = cue.text;
      form.elements.planned_chars_per_second.value = cue.planned_chars_per_second;
      form.elements.pause_budget_ms.value = cue.pause_budget_ms;
      form.scrollIntoView({ behavior: "smooth", block: "center" });
    });
  });
  document.querySelectorAll("[data-withdraw-cue]").forEach((button) => {
    button.disabled = !cueEditable;
    button.addEventListener("click", async () => {
      if (!cueEditable) return;
      try {
        await mutate(`/api/productions/${state.aggregate.production.id}/cues/${button.dataset.withdrawCue}`, { expectedRevision: state.aggregate.production.revision, idempotencyKey: key("withdraw") });
        notify("提示已撤回");
      } catch (error) {
        notify(error.message, true);
      }
    });
  });
}

function renderReview() {
  const take = (state.aggregate.rehearsals || []).at(-1);
  const decisions = state.aggregate.review_decisions || [];
  const pieces = [];
  for (const rehearsal of (state.aggregate.rehearsals || [])) {
    const comparisons = rehearsal.comparisons || [];
    const comparison = comparisons.map((item) => item.comparable ? `${item.cue_id} 时长 ${item.spoken_duration_delta_ms}ms，窗口结束偏差 ${item.window_end_delta_delta_ms}ms，问题 ${item.finding_count_delta}` : `${item.cue_id} 不可直接比较`).join("；");
    pieces.push(`<div class="card"><span>第 ${rehearsal.round} 轮排演 · ${rehearsal.findings.length} 个问题</span><span>${rehearsal.invalidated_at ? "已失效" : "有效"}${comparison ? ` · ${escapeHTML(comparison)}` : ""}</span></div>`);
  }
  const cueDecisions = new Map(decisions.filter((d) => !d.finding_id).map((d) => [d.cue_id, d]));
  for (const cue of latestCues()) {
    const d = cueDecisions.get(cue.id);
    pieces.push(`<div class="card"><span>${escapeHTML(cue.id)} · 提示处置</span><span>${d ? escapeHTML(d.action) : "待处置"}</span></div>`);
  }
  for (const finding of (take && !take.invalidated_at ? take.findings : [])) {
    const d = decisions.find((item) => item.finding_id === finding.id);
    pieces.push(`<div class="card"><span>${escapeHTML(finding.id)} · ${escapeHTML(finding.code)}</span><span>${d ? escapeHTML(d.action) : "待处置"}</span></div>`);
  }
  $("#review-output").innerHTML = pieces.join("") || '<p class="hint">校验通过后可登记排演。</p>';
}

function renderRehearsalForm() {
  const form = $("#rehearsal-form");
  if (!form) return;
  const active = state.aggregate.production.state === "REHEARSING";
  form.hidden = !active;
  if (!active) return;
  $("#rehearsal-inputs").innerHTML = latestCues().map((cue) => `<div class="measurement-row" data-cue-id="${escapeHTML(cue.id)}" data-cue-version="${cue.version}"><strong>${escapeHTML(cue.id)} · v${cue.version}</strong><label>实际开始<input data-measure="start" type="number" min="0" value="${cue.window_start_ms}"></label><label>实际结束<input data-measure="end" type="number" min="1" value="${cue.window_end_ms}"></label><label>实读时长<input data-measure="spoken" type="number" min="1" value="${Math.max(1, cue.window_end_ms - cue.window_start_ms)}"></label><label>停顿<input data-measure="pause" type="number" min="0" value="0"></label><label>可懂度问题<input data-measure="finding" placeholder="可选"></label></div>`).join("") || '<p class="hint">当前没有可排演提示。</p>';
}

function renderValidation() {
  const issues = state.aggregate.validation_issues || [];
  const groups = new Map();
  for (const item of issues) { const key = item.cue_id || "项目"; if (!groups.has(key)) groups.set(key, []); groups.get(key).push(item); }
  const count = state.aggregate.validation_blocking_count || issues.filter((item) => item.severity === "BLOCKING").length;
  $("#validation-output").innerHTML = (issues.length ? `<strong>阻断项 ${count} 个，共 ${issues.length} 项</strong>` : "") + ([...groups.entries()].map(([cueID, items]) => `<div class="validation-group"><strong>${escapeHTML(cueID)} · ${items.length} 项问题</strong>${items.map((item) => `<div class="card"><span><button data-focus-cue="${escapeHTML(item.cue_id || "")}">${escapeHTML(item.code)}</button></span><span>${escapeHTML(item.message)}（窗口 ${item.window_start_ms}—${item.window_end_ms}ms；预计 ${item.estimated_ms}ms / 可用 ${item.usable_ms}ms）</span></div>`).join("")}</div>`).join("") || '<p class="hint">尚未生成校验报告。</p>');
  document.querySelectorAll("[data-focus-cue]").forEach((button) => button.addEventListener("click", () => document.getElementById(`cue-${button.dataset.focusCue}`)?.scrollIntoView({ behavior: "smooth", block: "center" })));
}

async function loadReleasePreview() {
  try {
    const preview = await api(`/api/productions/${state.aggregate.production.id}/release/preview`);
    $("#release-preview").textContent = JSON.stringify({ cue_count: preview.cue_count, estimated_total_ms: preview.estimated_total_ms, window_total_ms: preview.window_total_ms, decision_count: preview.decision_count, production_revision: preview.production_revision }, null, 2);
  } catch (error) { $("#release-preview").textContent = error.message; }
}

function renderRelease() {
  const snapshot = state.aggregate.release;
  $("#release-output").textContent = JSON.stringify({ id: snapshot.id, content_hash: snapshot.content_hash, cue_count: snapshot.approved_cues.length, released_at: snapshot.released_at }, null, 2);
  for (const [selector, suffix] of [["#json-download", "release.json"], ["#vtt-download", "release.vtt"]]) {
    const link = $(selector);
    link.href = `/api/productions/${state.aggregate.production.id}/${suffix}`;
    link.classList.remove("disabled");
  }
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>'"]/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" }[ch]));
}

async function mutate(path, body, method = "POST") {
  const result = await api(path, { method, body: JSON.stringify(body) });
  state.aggregate = unwrap(result);
  render();
  await loadProductions();
  return result;
}

async function loadWindows() {
  const result = await api(`/api/productions/${state.aggregate.production.id}/windows`);
  state.windows = result.windows || [];
  state.conflicts = result.conflicts || [];
  renderTimeline();
  return result;
}

async function loadProductions() {
  try {
    const result = await api("/api/productions");
    state.productions = result.productions || [];
    $("#project-list-error").textContent = "";
    const select = $("#project-list");
    select.innerHTML = state.productions.map((p) => `<option value="${escapeHTML(p.id)}">${escapeHTML(p.title)} · ${p.state} · r${p.revision} · ${new Date(p.updated_at).toLocaleString()}</option>`).join("") || '<option value="">暂无项目，请先建立项目</option>';
    const remembered = localStorage.getItem("production_id");
    if (remembered && state.productions.some((p) => p.id === remembered)) select.value = remembered;
    else if (remembered) { localStorage.removeItem("production_id"); $("#project-list-error").textContent = "上次选择的项目已不存在，请重新选择。"; }
  } catch (error) { $("#project-list-error").textContent = error.message; }
}

async function resumeProject() {
  const id = $("#project-list").value;
  if (!id) return;
  state.aggregate = null;
  state.windows = [];
  state.conflicts = [];
  $("#workspace").hidden = true;
  $("#create-panel").hidden = false;
  try {
    state.aggregate = await api(`/api/productions/${id}`);
    localStorage.setItem("production_id", id);
    $("#project-list-error").textContent = "";
    try { await loadWindows(); } catch (error) { state.windows = []; state.conflicts = []; $("#project-list-error").textContent = `项目已恢复，但时间轴读取失败：${error.message}`; }
    render(); notify("已恢复项目");
  }
  catch (error) {
    localStorage.removeItem("production_id");
    await loadProductions();
    $("#project-list-error").textContent = error.message;
  }
}

$("#create-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  try {
    const result = await api("/api/productions", { method: "POST", body: JSON.stringify({
      title: form.get("title"), language: form.get("language"), duration_ms: Number(form.get("duration_ms")), frame_rate: Number(form.get("frame_rate")),
      participants: [{ name: "林编剧", role: "WRITER" }, { name: "周排演", role: "PERFORMER" }, { name: "顾审校", role: "REVIEWER" }], idempotencyKey: key("create"),
    }) });
    state.aggregate = unwrap(result); localStorage.setItem("production_id", state.aggregate.production.id); await loadProductions(); render(); notify("项目已建立");
  } catch (error) { notify(error.message, true); }
});

$("#segment-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  try {
    const segmentID = form.get("segment_id");
    const body = { expectedRevision: state.aggregate.production.revision, idempotencyKey: key(segmentID ? "update-segment" : "segment"), kind: form.get("kind"), scene_id: form.get("scene_id"), start_ms: Number(form.get("start_ms")), end_ms: Number(form.get("end_ms")), label: form.get("label") };
    await mutate(segmentID ? `/api/productions/${state.aggregate.production.id}/segments/${segmentID}` : `/api/productions/${state.aggregate.production.id}/segments`, body, segmentID ? "PUT" : "POST");
    event.currentTarget.elements.segment_id.value = "";
    notify(segmentID ? "时间轴区间已更新" : "时间轴区间已添加");
    await loadWindows();
  } catch (error) { notify(error.message, true); }
});

$("#windows-button").addEventListener("click", async () => {
  try {
    const result = await api(`/api/productions/${state.aggregate.production.id}/windows`);
    state.windows = result.windows || []; state.conflicts = result.conflicts || []; renderTimeline();
    notify(result.conflicts?.length ? `发现 ${result.conflicts.length} 个冲突` : `得到 ${state.windows.length} 个候选窗口`, Boolean(result.conflicts?.length));
  } catch (error) { notify(error.message, true); }
});

$("#finalize-button").addEventListener("click", async () => {
  try {
    await mutate(`/api/productions/${state.aggregate.production.id}/timeline/finalize`, { expectedRevision: state.aggregate.production.revision, idempotencyKey: key("timeline") });
    notify("时间轴已完成");
  } catch (error) { notify(error.message, true); }
});

$("#cue-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  try {
    await mutate(`/api/productions/${state.aggregate.production.id}/cues`, {
      expectedRevision: state.aggregate.production.revision, idempotencyKey: key("cue"), cue_id: form.get("cue_id") || undefined,
      window_start_ms: Number(form.get("window_start_ms")), window_end_ms: Number(form.get("window_end_ms")), text: form.get("text"), intent: form.get("intent"),
      planned_chars_per_second: Number(form.get("planned_chars_per_second")), pause_budget_ms: Number(form.get("pause_budget_ms")),
    });
    const cue = latestCues().at(-1); event.currentTarget.elements.cue_id.value = cue.id; notify(`提示 ${cue.id} 的 v${cue.version} 已保存`);
  } catch (error) { notify(error.message, true); }
});

async function showDiff(event) {
  const button = event.currentTarget;
  const version = Number(button.dataset.version);
  try {
    const result = await api(`/api/productions/${state.aggregate.production.id}/cues/${button.dataset.diff}/diff?from=${version - 1}&to=${version}`);
    const node = $("#diff-output"); node.hidden = false; node.textContent = `保留：${result.unchanged_prefix}\n删除：${result.removed}\n新增：${result.added}\n后缀：${result.unchanged_suffix}`;
  } catch (error) { notify(error.message, true); }
}

$("#validate-button").addEventListener("click", async () => {
  try {
    await mutate(`/api/productions/${state.aggregate.production.id}/validation`, { expectedRevision: state.aggregate.production.revision, idempotencyKey: key("validation") });
    const blocking = state.aggregate.validation_blocking_count || 0;
    notify(blocking ? `校验发现 ${blocking} 个阻断项，请按报告修稿` : "校验通过，提示版本已固定", Boolean(blocking));
  } catch (error) { notify(error.message, true); }
});

$("#rehearse-button").addEventListener("click", () => { const form = $("#rehearsal-form"); if (form) { form.hidden = false; form.scrollIntoView({ behavior: "smooth", block: "center" }); } });

$("#rehearsal-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const measurements = [];
  const findings = [];
  event.currentTarget.querySelectorAll("[data-cue-id]").forEach((row) => {
    measurements.push({ cue_id: row.dataset.cueId, cue_version: Number(row.dataset.cueVersion), actual_start_ms: Number(row.querySelector('[data-measure="start"]').value), actual_end_ms: Number(row.querySelector('[data-measure="end"]').value), spoken_duration_ms: Number(row.querySelector('[data-measure="spoken"]').value), pause_ms: Number(row.querySelector('[data-measure="pause"]').value) });
    const message = row.querySelector('[data-measure="finding"]').value.trim();
    if (message) findings.push({ cue_id: row.dataset.cueId, code: "COMPREHENSIBILITY", message, severity: "ADVISORY" });
  });
  try { await mutate(`/api/productions/${state.aggregate.production.id}/rehearsals`, { expectedRevision: state.aggregate.production.revision, idempotencyKey: key("rehearsal"), measurements, findings }); notify("排演记录已形成"); }
  catch (error) { notify(error.message, true); }
});

$("#accept-button").addEventListener("click", async () => {
  try {
    for (const cue of latestCues()) {
      const already = (state.aggregate.review_decisions || []).some((d) => d.cue_id === cue.id && d.action === "ACCEPT");
      if (!already) await mutate(`/api/productions/${state.aggregate.production.id}/reviews`, { expectedRevision: state.aggregate.production.revision, idempotencyKey: key("review"), cue_id: cue.id, action: "ACCEPT", reviewer: "顾审校" });
    }
    notify("全部提示已逐项接受");
  } catch (error) { notify(error.message, true); }
});

$("#review-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const form = new FormData(event.currentTarget);
  try {
    await mutate(`/api/productions/${state.aggregate.production.id}/reviews`, {
      expectedRevision: state.aggregate.production.revision, idempotencyKey: key("review-item"), cue_id: form.get("cue_id"),
      action: form.get("action"), reason: form.get("reason"), reviewer: form.get("reviewer"),
    });
    notify(form.get("action") === "ACCEPT" ? "提示已接受" : "项目已进入修订流程");
  } catch (error) { notify(error.message, true); }
});

$("#approve-button").addEventListener("click", async () => {
  try { await mutate(`/api/productions/${state.aggregate.production.id}/approve`, { expectedRevision: state.aggregate.production.revision, idempotencyKey: key("approve") }); notify("脚本已批准"); }
  catch (error) { notify(error.message, true); }
});

$("#release-button").addEventListener("click", async () => {
  try { await mutate(`/api/productions/${state.aggregate.production.id}/release`, { expectedRevision: state.aggregate.production.revision, idempotencyKey: key("release"), released_by: "顾审校" }); notify("不可变发布快照已生成"); }
  catch (error) { notify(error.message, true); }
});

$("#resume-project").addEventListener("click", resumeProject);
async function boot() {
  const listPromise = loadProductions();
  api("/healthz").then(() => { $("#connection").textContent = "服务正常"; }).catch(() => { $("#connection").textContent = "服务不可用"; });
  await listPromise;
  const remembered = localStorage.getItem("production_id");
  if (remembered && state.productions.some((p) => p.id === remembered)) await resumeProject();
}
boot();
