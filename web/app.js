"use strict";

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => Array.from(document.querySelectorAll(selector));
const clone = (value) => structuredClone(value);
const clamp = (value, min, max) => Math.min(max, Math.max(min, value));

const moduleTitles = {
  clock: "时钟",
  date: "日期与星期",
  lunar: "农历与节气",
  quote: "每日内容",
  current_weather: "当前天气",
  forecast: "三日预报",
  bottom_right: "环境数据卡",
  footer: "底部状态栏",
  weather_source: "天气数据源",
  frontlight: "Kindle 前光",
  device: "显示与同步",
};

const positionPaths = {
  clock: ["clock_x_off", "clock_y_off"],
  date: ["date_x_off", "date_y_off"],
  quote: ["quote_x_off", "quote_y_off"],
};

const sizePaths = {
  clock: ["clock_size", 40, 320],
  date: ["date_size", 20, 120],
  quote: ["quote_size", 16, 60],
};

const modulePaths = {
  clock: ["clock_font", "clock_size", "clock_x_off", "clock_y_off"],
  date: ["date_font", "date_size", "date_x_off", "date_y_off"],
  lunar: ["show_lunar"],
  quote: ["show_quote", "quote_type", "quote_font", "quote_size", "quote_x_off", "quote_y_off"],
  current_weather: ["show_current_weather"],
  forecast: ["show_forecast"],
  bottom_right: ["bottom_right_module"],
  footer: ["show_footer"],
  weather_source: ["location", "timezone", "weather_url", "weather_refresh_seconds", "offline_grace_minutes"],
  frontlight: ["frontlight_mode", "day_brightness", "night_brightness", "day_start_hour", "night_start_hour"],
  device: ["rotate", "ghost_clear_minutes", "listen_addr", "kindle_ssh"],
};

const state = {
  config: null,
  defaults: null,
  draft: null,
  layout: null,
  active: "clock",
  view: "landscape",
  dirty: false,
  previewSeq: 0,
  previewTimer: null,
  toastTimer: null,
  drag: null,
};

function getPath(object, path) {
  return path.split(".").reduce((value, key) => value?.[key], object);
}

function setPath(object, path, value) {
  const keys = path.split(".");
  const last = keys.pop();
  const parent = keys.reduce((value, key) => value[key], object);
  parent[last] = value;
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function selectField(path, label, options, help = "") {
  const current = getPath(state.draft, path);
  const items = options.map(([value, text]) =>
    `<option value="${escapeHTML(value)}" ${String(value) === String(current) ? "selected" : ""}>${escapeHTML(text)}</option>`
  ).join("");
  return `<label class="field field-stacked"><span>${label}</span><select data-path="${path}">${items}</select>${help ? `<small>${help}</small>` : ""}</label>`;
}

function textField(path, label, options = {}) {
  const value = getPath(state.draft, path) ?? "";
  const type = options.type || "text";
  const attributes = [
    `type="${type}"`, `data-path="${path}"`, `value="${escapeHTML(value)}"`,
  ];
  if (options.min !== undefined) attributes.push(`min="${options.min}"`);
  if (options.max !== undefined) attributes.push(`max="${options.max}"`);
  if (options.step !== undefined) attributes.push(`step="${options.step}"`);
  if (type === "number") attributes.push(`data-number="${options.float ? "float" : "int"}"`);
  return `<label class="field field-stacked"><span>${label}</span><input ${attributes.join(" ")}>${options.help ? `<small>${options.help}</small>` : ""}</label>`;
}

function rangeField(path, label, min, max, step = 1, help = "") {
  const value = Number(getPath(state.draft, path));
  return `<div class="field"><span class="field-label">${label}</span><div class="range-row"><input type="range" data-path="${path}" data-number="int" min="${min}" max="${max}" step="${step}" value="${value}"><input type="number" data-path="${path}" data-number="int" min="${min}" max="${max}" step="${step}" value="${value}"></div>${help ? `<small>${help}</small>` : ""}</div>`;
}

function toggleField(path, label, help = "") {
  const checked = getPath(state.draft, path) ? "checked" : "";
  return `<div class="toggle-row"><div><label>${label}</label>${help ? `<small>${help}</small>` : ""}</div><label class="switch"><input type="checkbox" data-path="${path}" ${checked}><span></span></label></div>`;
}

function section(title, content) {
  return `<section class="form-section"><h3>${title}</h3>${content}</section>`;
}

function offsetField(path, label, min = -180, max = 180, step = 1) {
  const value = Number(getPath(state.draft, path)) || 0;
  return `
    <div class="field offset-field">
      <div class="field-header">
        <span class="field-label">${label}</span>
        <button type="button" class="btn-zero" data-zero="${path}" title="点击快速归零居中">归零 (0)</button>
      </div>
      <div class="range-row">
        <input type="range" data-path="${path}" data-number="int" min="${min}" max="${max}" step="${step}" value="${value}">
        <input type="number" data-path="${path}" data-number="int" min="${min}" max="${max}" step="${step}" value="${value}">
      </div>
    </div>
  `;
}

function positionEditorFor(prefix) {
  return section("位置调整 (滑块实时预览 / 支持画布鼠标拖拽)",
    offsetField(`${prefix}_x_off`, "水平位置偏移 (X轴)", -180, 180) +
    offsetField(`${prefix}_y_off`, "垂直位置偏移 (Y轴)", -180, 180) +
    `<p class="info-card">💡 <strong>所见即所得</strong>：滑动上方滑块可实时预览排版；也可以<strong>在左侧画布上直接按住鼠标拖拽</strong>此模块。</p>`
  );
}

function inspectorMarkup(module) {
  switch (module) {
    case "clock":
      return section("字体与尺寸",
        selectField("clock_font", "数字字体", [["bebas", "Bebas Neue"], ["jetbrains", "JetBrains Mono"], ["montserrat", "Montserrat"], ["playfair", "Playfair Display"], ["noto", "Noto Sans SC"]]) +
        rangeField("clock_size", "时钟字号", 0, 320, 5, "0 表示按可用空间自动适配。")) + positionEditorFor("clock");
    case "date":
      return section("字体与尺寸",
        selectField("date_font", "日期字体", [["bold", "Noto Sans · 粗体"], ["med", "Noto Sans · 中等"], ["reg", "Noto Sans · 常规"], ["serif", "ZCOOL 宋体"], ["bebas", "Bebas Neue"], ["jetbrains", "JetBrains Mono"], ["montserrat", "Montserrat"]]) +
        rangeField("date_size", "日期字号", 0, 120, 1, "0 表示自动适配；日期和星期作为整体防溢出。")) + positionEditorFor("date");
    case "lunar":
      return section("模块显示", toggleField("show_lunar", "显示农历与节气", "关闭后，每日内容卡会使用释放出的空间。")) +
        section("内容说明", `<p class="info-card">农历、节气和传统节日均在 主机 本地计算，不依赖外部接口。</p>`);
    case "quote":
      return section("模块显示", toggleField("show_quote", "显示每日内容卡")) +
        section("内容与样式",
          selectField("quote_type", "内容类型", [["mao", "毛选摘录"], ["almanac", "黄历宜忌"], ["famous", "名人名言"], ["inspirational", "励志语录"], ["poetry", "古诗词"]]) +
          selectField("quote_font", "正文字体", [["bold", "Noto Sans · 粗体"], ["med", "Noto Sans · 中等"], ["reg", "Noto Sans · 常规"], ["kai", "马善政毛笔"], ["wei", "ZCOOL 小薇"], ["xing", "龙藏手写"], ["lively", "ZCOOL 快乐体"]]) +
          rangeField("quote_size", "正文字号", 0, 60, 1, "0 表示自动适配。")) + positionEditorFor("quote");
    case "current_weather":
      return section("模块显示", toggleField("show_current_weather", "显示当前天气", "包括天气图标、温度、体感、湿度和缓存状态。")) +
        section("数据位置", `<p class="info-card">当前城市：<strong>${escapeHTML(state.draft.location.name)}</strong><br>城市与经纬度在“天气数据源”中修改。</p>`);
    case "forecast":
      return section("模块显示", toggleField("show_forecast", "显示三日预报", "从今天日期开始取数据，跨午夜不会把昨天标成今天。")) +
        section("数据完整性", `<p class="info-card">天气接口缺少必要数组时会保留缓存并提示异常，不使用不完整的预报。</p>`);
    case "bottom_right":
      return section("数据卡内容", selectField("bottom_right_module", "卡片类型", [["solar", "太阳轨迹与月相"], ["air", "空气质量与环境"]], "切换后立即使用同一数据快照重新渲染。")) +
        section("相关模块", toggleField("show_current_weather", "显示当前天气") + toggleField("show_forecast", "显示三日预报")) +
        section("数据说明", `<p class="info-card">空气质量不可用时显示“--”，不会把请求失败显示为 0。夜间紫外线 0 是合法值。</p>`);
    case "footer":
      return section("模块显示", toggleField("show_footer", "显示底部状态栏", "包含日期、服务状态或 Kindle 电量信息。"));
    case "weather_source":
      return section("位置",
          textField("location.name", "显示城市") +
          textField("location.city_code", "国内天气城市代码", { help: "例如 101110101；用于国内天气和空气质量数据源。" }) +
          `<div class="field-row">${textField("location.lat", "纬度", { type: "number", min: -90, max: 90, step: 0.0001, float: true })}${textField("location.lon", "经度", { type: "number", min: -180, max: 180, step: 0.0001, float: true })}</div>`) +
        section("更新策略",
          textField("timezone", "IANA 时区", { help: "默认 Asia/Shanghai；时区数据已内嵌。" }) +
          rangeField("weather_refresh_seconds", "天气刷新周期（秒）", 60, 7200, 60) +
          textField("offline_grace_minutes", "离线宽限（分钟）", { type: "number", min: 1, max: 10080 }));
    case "frontlight":
      return section("前光策略", selectField("frontlight_mode", "工作模式", [["schedule", "按时间自动调节"], ["on", "始终使用白天亮度"], ["off", "始终关闭"]])) +
        section("亮度", rangeField("day_brightness", "白天亮度", 0, 24) + rangeField("night_brightness", "夜间亮度", 0, 24)) +
        section("时间", `<div class="field-row">${textField("day_start_hour", "白天开始（0–23）", { type: "number", min: 0, max: 23 })}${textField("night_start_hour", "夜间开始（0–23）", { type: "number", min: 0, max: 23 })}</div><p class="info-card">0 是合法亮度和合法小时。设置随 Kindle 下次 HTTP 拉取下发生效。</p>`);
    case "device":
      return section("Kindle 显示",
          selectField("rotate", "横放方向", [[90, "90° · 顶部边在右"], [270, "270° · 顶部边在左"]]) +
          rangeField("ghost_clear_minutes", "全刷清残影（分钟）", 0, 720, 10, "0 表示关闭定时全刷。")) +
        section("主机 服务", textField("listen_addr", "监听地址", { help: "默认 :8088。修改监听地址后需要重启 主机 服务。" })) +
        section("可选 SSH 立即刷新",
          toggleField("kindle_ssh.enabled", "启用 SSH 立即刷新", "关闭时不影响 Kindle 通过 HTTP 自动更新。") +
          textField("kindle_ssh.host", "Kindle IP 地址") +
          `<div class="field-row">${textField("kindle_ssh.port", "SSH 端口", { type: "number", min: 1, max: 65535 })}${textField("kindle_ssh.user", "SSH 用户")}</div>` +
          `<p class="info-card">私钥和 known-hosts 路径只在 主机 本地配置中维护。网页只能触发固定刷新命令，不能写入 Shell 配置。</p>`);
    default:
      return "";
  }
}

function renderInspector() {
  $("#inspector-title").textContent = moduleTitles[state.active] || "模块设置";
  $("#inspector-form").innerHTML = inspectorMarkup(state.active);
  $("#reset-module").disabled = !modulePaths[state.active];
}

function renderNavigation() {
  $$(".nav-item").forEach((button) => button.classList.toggle("active", button.dataset.module === state.active));
}

function selectModule(module) {
  if (!moduleTitles[module]) return;
  state.active = module;
  renderNavigation();
  renderInspector();
  $$(".module-zone").forEach((zone) => {
    zone.classList.toggle("active", zone.dataset.module === module);
  });
}

function setDirty(value = true) {
  state.dirty = value;
  const badge = $("#dirty-badge");
  badge.textContent = value ? "有未保存修改" : "已保存";
  badge.className = `badge ${value ? "badge-dirty" : "badge-saved"}`;
  $("#undo-button").disabled = !value;
}

function syncInspectorInputs(paths) {
  const wanted = new Set(paths);
  $$("#inspector-form [data-path]").forEach((input) => {
    if (!wanted.has(input.dataset.path)) return;
    const value = getPath(state.draft, input.dataset.path);
    if (input.type === "checkbox") {
      input.checked = Boolean(value);
    } else if (input.value !== String(value)) {
      input.value = value;
    }
  });
}

function updateFromInput(input) {
  const path = input.dataset.path;
  if (!path) return;
  let value;
  if (input.type === "checkbox") value = input.checked;
  else if (input.dataset.number === "int") {
    if (input.value === "") return;
    value = Number.parseInt(input.value, 10);
  } else if (input.dataset.number === "float") {
    if (input.value === "") return;
    value = Number.parseFloat(input.value);
  } else value = input.value;
  if (typeof value === "number" && !Number.isFinite(value)) return;

  setPath(state.draft, path, value);
  syncInspectorInputs([path]);
  setDirty(true);
  // 滑块或偏移量调整使用 40ms 超低延迟，真正做到滑动画布即时跟随，所见即所得
  const delay = path.endsWith("_off") || path.endsWith("_size") ? 40 : 120;
  schedulePreview(delay);
}

function encodeDraft(value) {
  const bytes = new TextEncoder().encode(JSON.stringify(value));
  let binary = "";
  for (let index = 0; index < bytes.length; index += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(index, index + 0x8000));
  }
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replaceAll("=", "");
}

async function fetchJSON(url, options = {}) {
  const response = await fetch(url, options);
  const type = response.headers.get("content-type") || "";
  const body = type.includes("application/json") ? await response.json() : { message: await response.text() };
  if (!response.ok) {
    const message = body.error?.message || body.message || `HTTP ${response.status}`;
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }
  return body;
}

function loadImage(url) {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(url);
    image.onerror = () => reject(new Error("预览图片加载失败，请检查配置或 主机 服务"));
    image.src = url;
  });
}

function schedulePreview(delay = 180) {
  window.clearTimeout(state.previewTimer);
  state.previewTimer = window.setTimeout(refreshPreview, delay);
}

async function refreshPreview() {
  if (!state.draft) return;
  const seq = ++state.previewSeq;
  const encoded = encodeDraft(state.draft);
  const sample = $("#sample-toggle").checked ? "1" : "0";
  const imageEndpoint = state.view === "device" ? "/screen.png" : "/preview.png";
  const query = `draft=${encoded}&sample=${sample}&v=${seq}`;
  $("#preview-loading").hidden = false;

  try {
    const [layout, imageURL] = await Promise.all([
      fetchJSON(`/api/layout?${query}`),
      loadImage(`${imageEndpoint}?${query}`),
    ]);
    if (seq !== state.previewSeq) return;
    state.layout = layout;
    $("#preview-image").src = imageURL;
    $("#preview-stage").classList.toggle("device", state.view === "device");
    if (!state.drag) {
      renderOverlays();
    }
    $("#preview-message").textContent = state.view === "device"
      ? "758×1024 灰度图 · 与 Kindle 拉取格式一致"
      : `${layout.width}×${layout.height} · 选区来自真实布局`;
  } catch (error) {
    if (seq !== state.previewSeq) return;
    $("#preview-message").textContent = error.message;
    showToast(error.message, true);
  } finally {
    if (seq === state.previewSeq) $("#preview-loading").hidden = true;
  }
}

function renderOverlays() {
  const container = $("#module-overlays");
  container.innerHTML = "";
  container.classList.toggle("device-hidden", state.view === "device" || !state.layout);
  if (state.view === "device" || !state.layout) return;

  for (const module of state.layout.modules || []) {
    if (!moduleTitles[module.id] || ["weather_source", "frontlight", "device"].includes(module.id)) continue;
    const bounds = module.bounds || module;
    const width = bounds.width ?? bounds.w;
    const height = bounds.height ?? bounds.h;
    const zone = document.createElement("button");
    zone.type = "button";
    zone.className = `module-zone${module.id === state.active ? " active" : ""}${module.draggable ? " draggable" : ""}`;
    zone.dataset.module = module.id;
    zone.style.left = `${bounds.x / state.layout.width * 100}%`;
    zone.style.top = `${bounds.y / state.layout.height * 100}%`;
    zone.style.width = `${width / state.layout.width * 100}%`;
    zone.style.height = `${height / state.layout.height * 100}%`;
    zone.title = `${module.title || moduleTitles[module.id]}${module.draggable ? " · 可拖动" : ""}`;
    zone.setAttribute("aria-label", zone.title);
    zone.addEventListener("click", () => selectModule(module.id));
    zone.addEventListener("pointerdown", beginDrag);
    zone.addEventListener("keydown", handleZoneKeyboard);
    container.appendChild(zone);
  }
}

function beginDrag(event) {
  const id = event.currentTarget.dataset.module;
  const paths = positionPaths[id];
  if (!paths || state.view !== "landscape" || event.button !== 0) return;
  event.preventDefault();
  selectModule(id);
  const zone = event.currentTarget;
  zone.setPointerCapture(event.pointerId);
  zone.classList.add("dragging");
  state.drag = {
    id,
    zone,
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    offsetX: Number(getPath(state.draft, paths[0])) || 0,
    offsetY: Number(getPath(state.draft, paths[1])) || 0,
    left: Number.parseFloat(zone.style.left) || 0,
    top: Number.parseFloat(zone.style.top) || 0,
  };
  zone.addEventListener("pointermove", moveDrag);
  zone.addEventListener("pointerup", endDrag);
  zone.addEventListener("pointercancel", endDrag);
}

function moveDrag(event) {
  const drag = state.drag;
  if (!drag || event.pointerId !== drag.pointerId) return;
  const paths = positionPaths[drag.id];
  const rect = $("#preview-stage").getBoundingClientRect();
  if (!rect.width || !rect.height) return;
  const deltaX = (event.clientX - drag.startX) * state.layout.width / rect.width;
  const deltaY = (event.clientY - drag.startY) * state.layout.height / rect.height;
  setPath(state.draft, paths[0], clamp(Math.round(drag.offsetX + deltaX), -180, 180));
  setPath(state.draft, paths[1], clamp(Math.round(drag.offsetY + deltaY), -180, 180));
  drag.zone.style.left = `${drag.left + (event.clientX - drag.startX) / rect.width * 100}%`;
  drag.zone.style.top = `${drag.top + (event.clientY - drag.startY) / rect.height * 100}%`;
  syncInspectorInputs(paths);
  setDirty(true);
  schedulePreview(50);
}

function endDrag(event) {
  const drag = state.drag;
  if (!drag || event.pointerId !== drag.pointerId) return;
  drag.zone.classList.remove("dragging");
  drag.zone.removeEventListener("pointermove", moveDrag);
  drag.zone.removeEventListener("pointerup", endDrag);
  drag.zone.removeEventListener("pointercancel", endDrag);
  if (drag.zone.hasPointerCapture(event.pointerId)) drag.zone.releasePointerCapture(event.pointerId);
  state.drag = null;
  schedulePreview(0);
}

function handleZoneKeyboard(event) {
  const id = event.currentTarget.dataset.module;
  const paths = positionPaths[id];
  if (!paths) return;
  const step = event.shiftKey ? 10 : 1;
  const offsets = {
    ArrowLeft: [-step, 0], ArrowRight: [step, 0], ArrowUp: [0, -step], ArrowDown: [0, step],
  };
  if (offsets[event.key]) {
    event.preventDefault();
    const [dx, dy] = offsets[event.key];
    setPath(state.draft, paths[0], clamp(Number(getPath(state.draft, paths[0])) + dx, -200, 200));
    setPath(state.draft, paths[1], clamp(Number(getPath(state.draft, paths[1])) + dy, -200, 200));
  } else if (["+", "=", "-"].includes(event.key) && sizePaths[id]) {
    event.preventDefault();
    const [path, min, max] = sizePaths[id];
    let value = Number(getPath(state.draft, path));
    if (value === 0) value = Number(getPath(state.defaults, path)) || min;
    setPath(state.draft, path, clamp(value + (event.key === "-" ? -1 : 1), min, max));
    syncInspectorInputs([path]);
  } else return;
  syncInspectorInputs(paths);
  setDirty(true);
  schedulePreview();
}

function showToast(message, error = false) {
  const toast = $("#toast");
  window.clearTimeout(state.toastTimer);
  toast.textContent = message;
  toast.classList.toggle("error", error);
  toast.hidden = false;
  state.toastTimer = window.setTimeout(() => { toast.hidden = true; }, error ? 6500 : 4000);
}

function authHeaders() {
  const headers = { "Content-Type": "application/json", "X-KDash-Client": "kdash" };
  const token = $("#settings-token").value;
  if (token) headers["X-KDash-Token"] = token;
  return headers;
}

async function saveSettings(refresh) {
  if (!state.draft) return;
  const buttons = [$("#save-button"), $("#apply-button")];
  buttons.forEach((button) => { button.disabled = true; });
  try {
    const result = await fetchJSON(`/api/config${refresh ? "?refresh=1" : ""}`, {
      method: "PUT",
      headers: authHeaders(),
      body: JSON.stringify(state.draft),
    });
    state.config = clone(result.config);
    state.draft = clone(result.config);
    setDirty(false);
    renderInspector();
    await refreshPreview();
    let message = result.message || "设置已保存";
    if (result.restart_required) message += "；监听地址已变化，请重启 kdash 服务";
    if (result.refresh?.message) message = result.refresh.message;
    showToast(message, result.refresh?.status === "error");
  } catch (error) {
    if (error.status === 401) {
      $(".token-panel").open = true;
      $("#settings-token").focus();
    }
    showToast(error.message, true);
  } finally {
    buttons.forEach((button) => { button.disabled = false; });
  }
}

async function loadConfig() {
  try {
    const result = await fetchJSON("/api/config");
    state.config = clone(result.config);
    state.draft = clone(result.config);
    state.defaults = clone(result.defaults);
    setDirty(false);
    renderNavigation();
    renderInspector();
    $("#server-status").textContent = "主机 渲染服务已连接";
    await refreshPreview();
  } catch (error) {
    $("#server-status").textContent = "无法读取服务端配置";
    showToast(error.message, true);
  }
}

$("#inspector-form").addEventListener("input", (event) => updateFromInput(event.target));
$("#inspector-form").addEventListener("change", (event) => {
  if (event.target.matches("select, input[type=checkbox]")) updateFromInput(event.target);
});
$("#inspector-form").addEventListener("click", (event) => {
  const zeroBtn = event.target.closest("[data-zero]");
  if (!zeroBtn) return;
  const path = zeroBtn.dataset.zero;
  if (!path) return;
  setPath(state.draft, path, 0);
  syncInspectorInputs([path]);
  setDirty(true);
  schedulePreview(40);
});

$$(".nav-item").forEach((button) => button.addEventListener("click", () => selectModule(button.dataset.module)));
$$("[data-view]").forEach((button) => button.addEventListener("click", () => {
  state.view = button.dataset.view;
  $$("[data-view]").forEach((item) => item.classList.toggle("active", item === button));
  refreshPreview();
}));
$("#sample-toggle").addEventListener("change", refreshPreview);
$("#save-button").addEventListener("click", () => saveSettings(false));
$("#apply-button").addEventListener("click", () => saveSettings(true));
$("#undo-button").addEventListener("click", () => {
  if (!state.config) return;
  state.draft = clone(state.config);
  setDirty(false);
  renderInspector();
  refreshPreview();
});
$("#reset-module").addEventListener("click", () => {
  const paths = modulePaths[state.active];
  if (!paths || !state.defaults) return;
  for (const path of paths) setPath(state.draft, path, clone(getPath(state.defaults, path)));
  setDirty(true);
  renderInspector();
  refreshPreview();
});
$("#settings-token").value = sessionStorage.getItem("kdash-settings-token") || "";
$("#settings-token").addEventListener("change", (event) => {
  if (event.target.value) sessionStorage.setItem("kdash-settings-token", event.target.value);
  else sessionStorage.removeItem("kdash-settings-token");
});
window.addEventListener("beforeunload", (event) => {
  if (!state.dirty) return;
  event.preventDefault();
  event.returnValue = "";
});

loadConfig();
