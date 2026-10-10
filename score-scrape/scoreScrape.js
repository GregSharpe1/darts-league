(() => {
  if (window.__scoreScrapeInstalled) {
    console.log("[SCORE-SCRAPE]", "Score scrape script already installed");
    return;
  }

  window.__scoreScrapeInstalled = true;

  const SETTINGS_COOKIE_NAME = "autodarts_score_scrape_settings";
  const SETTINGS_COOKIE_DAYS = 365;
  const DEFAULT_SETTINGS = {
    endpoint: "",
    baseScore: 501,
    winningLegs: 3
  };

  const open = XMLHttpRequest.prototype.open;
  const send = XMLHttpRequest.prototype.send;
  const INSTALL_INDICATOR_ID = "autodarts-wrapper-installed-indicator";
  const log = (...args) => console.log("[SCORE-SCRAPE]", ...args);
  const sandbox = window.__scoreScrapeSandbox !== false;
  const requests = new WeakMap();
  const pending = new Set();
  const submitted = new Set();
  let queue = Promise.resolve();
  const requireValue = (condition) => { if (!condition) throw new Error("Unsupported or invalid match data"); };
  const id = (value) => { requireValue(typeof value === "string" && /^[A-Za-z0-9_-]{1,128}$/.test(value)); return value; };
  const count = (value, max) => { requireValue(Number.isInteger(value) && value >= 0 && value <= max); return value; };
  const metric = (value, max, integer = true) => {
    if (value == null) return null;
    requireValue(Number.isFinite(value) && value >= 0 && value <= max && (!integer || Number.isInteger(value)));
    return value;
  };
  const timestamp = (value) => {
    if (value == null) return null;
    requireValue(typeof value === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$/.test(value) && Number.isFinite(Date.parse(value)));
    return value;
  };
  const matchURL = (value) => {
    try {
      const url = new URL(value, location.href);
      if (!["https://api.autodarts.com", "https://api.autodarts.io"].includes(url.origin) || url.username || url.password || url.hash) return null;
      return /^\/as\/v0\/matches\/([A-Za-z0-9_-]{1,128})\/stats$/.exec(url.pathname)?.[1] ?? null;
    } catch { return null; }
  };
  const project = (state, matchId) => {
    requireValue(state && state.id === matchId && state.variant === "X01" && state.settings?.baseScore === 501 && state.settings.outMode === "Double");
    requireValue(state.targetLegs === 3 && (state.targetSets === null || state.targetSets === 0) && timestamp(state.finishedAt) !== null);
    requireValue(Array.isArray(state.players) && state.players.length === 2 && Array.isArray(state.scores) && state.scores.length === 2);
    const ids = state.players.map(player => id(player.id));
    requireValue(new Set(ids).size === 2 && [0, 1].includes(state.winner));
    const scores = state.scores.map(score => count(score.legs, 3));
    requireValue(scores[state.winner] === 3 && scores[1 - state.winner] < 3);
    const reference = value => { requireValue(ids.includes(value)); return value; };
    const players = state.players.map((player, index) => {
      requireValue(typeof player.name === "string" && [...player.name].length >= 1 && [...player.name].length <= 80 && !/[\u0000-\u001f\u007f-\u009f]/.test(player.name));
      const rows = (state.matchStats ?? []).filter(row => row.playerId === player.id);
      requireValue(rows.length <= 1);
      const row = rows[0];
      const stats = row ? {
        match_average: metric(row.average, 180, false), points_scored: null,
        darts_thrown: metric(row.dartsThrown, 3000), checkout_hits: metric(row.checkoutsHit, 3), checkout_attempts: metric(row.checkouts, 3000)
      } : null;
      if (stats) {
        for (const [source, field, max, integer] of [
          ["first9Average", "first_nine_average", 180, false], ["averageUntil170", "average_until_170", 180, false],
          ["checkoutPoints", "highest_finish", 170, true], ["total180", "total_180", 1000, true],
          ["less60", "less_60", 1000, true], ["plus60", "plus_60", 1000, true],
          ["plus100", "plus_100", 1000, true], ["plus140", "plus_140", 1000, true], ["plus170", "plus_170", 1000, true]
        ]) {
          if (Object.hasOwn(row, source)) stats[field] = metric(row[source], max, integer);
        }
        requireValue(stats.checkout_hits == null || stats.checkout_attempts == null || stats.checkout_hits <= stats.checkout_attempts);
        requireValue(stats.checkout_attempts == null || stats.darts_thrown == null || stats.checkout_attempts <= stats.darts_thrown);
        requireValue(stats.darts_thrown !== 0 || (stats.match_average === null && (stats.points_scored === null || stats.points_scored === 0)));
      }
      return { match_player_id: player.id, account_id: player.userId == null ? null : id(player.userId), display_name: player.name, legs_won: scores[index], stats };
    });
    let detail = null;
    requireValue(state.games == null || Array.isArray(state.games));
    if (state.games != null && state.games.length) {
      requireValue(Array.isArray(state.games) && state.games.length <= 5);
      const legs = state.games.map(game => {
        requireValue(game.set === 0 && Array.isArray(game.turns) && game.turns.length <= 200);
        const visits = [...game.turns].sort((a, b) => a.turn - b.turn).map(turn => {
          requireValue(typeof turn.busted === "boolean" && Array.isArray(turn.throws) && turn.throws.length >= 1 && turn.throws.length <= 3);
          const end = count(turn.score, 501);
          const start = end + (turn.busted ? 0 : count(turn.points, 180));
          requireValue(start >= 2 && start <= 501);
          const throws = [...turn.throws].sort((a, b) => a.throw - b.throw).map((dart, index) => {
            requireValue(dart.throw === index);
            const beds = { Outside: "miss", Single: "single", SingleOuter: "single", Double: "double", Triple: "triple", OuterBull: "outer_bull", InnerBull: "inner_bull" };
            let bed = beds[dart.segment?.bed];
            if (dart.segment?.number === 25 && bed === "single") bed = "outer_bull";
            if (dart.segment?.number === 25 && bed === "double") bed = "inner_bull";
            requireValue(Boolean(bed));
            const number = bed === "miss" ? 0 : dart.segment.number;
            requireValue(bed === "miss" || (bed.endsWith("bull") ? number === 25 : Number.isInteger(number) && number >= 1 && number <= 20));
            const entry = ({ manual_coords: "manual", manual: "manual", manual_segment: "manual", auto: "automatic", automatic: "automatic" })[dart.entry] ?? "unknown";
            let position = null;
            if (dart.coords?.x != null && dart.coords?.y != null) {
              requireValue(Number.isFinite(dart.coords.x) && Number.isFinite(dart.coords.y));
              const manualCoordinates = dart.entry === "manual_coords";
              position = { x: dart.coords.x, y: dart.coords.y, units: manualCoordinates ? "board-radius" : null, origin: manualCoordinates ? "bull" : null, axis_orientation: manualCoordinates ? "x-right-y-up" : null, provenance: entry };
            }
            return { number: index + 1, segment: { bed, number }, entry_type: entry, position };
          });
          return { number: count(turn.turn, 199) + 1, player_id: reference(turn.playerId), start_remaining: start, end_remaining: end, bust: turn.busted, throws };
        });
        requireValue(new Set(visits.map(visit => visit.number)).size === visits.length);
        const completed = timestamp(game.finishedAt) !== null;
        return { number: count(game.leg, 4) + 1, completed, winner_id: completed ? reference(game.winnerPlayerId) : null, visits };
      }).sort((a, b) => a.number - b.number);
      requireValue(new Set(legs.map(leg => leg.number)).size === legs.length);
      // Source history may be truncated; the backend checks scoring semantics.
      detail = { coverage: "partial", legs };
    }
    const payload = { schema_version: "autodarts.import.v1", source: "autodarts", external_match_id: id(state.id), playedAt: timestamp(state.createdAt), settings: { base_score: 501, legs_to_win: 3, out: "double" }, completed: true, players, detail };
    requireValue(new TextEncoder().encode(JSON.stringify(payload)).length <= 128 * 1024);
    return payload;
  };

  const getCookie = (name) => {
    const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
    try { return match ? decodeURIComponent(match[1]) : null; } catch { return null; }
  };

  const setCookie = (name, value, days) => {
    const expires = new Date(Date.now() + days * 24 * 60 * 60 * 1000).toUTCString();
    document.cookie = `${name}=${encodeURIComponent(value)}; expires=${expires}; path=/; SameSite=Lax`;
  };

  const loadSettingsFromCookie = () => {
    const raw = getCookie(SETTINGS_COOKIE_NAME);
    if (!raw) {
      return null;
    }

    try {
      const parsed = JSON.parse(raw);
      if (typeof parsed.endpoint === "string" && parsed.endpoint && Number.isFinite(parsed.baseScore) && Number.isFinite(parsed.winningLegs)) {
        return { endpoint: parsed.endpoint, baseScore: parsed.baseScore, winningLegs: parsed.winningLegs };
      }
    } catch {
      /* fall through to null */
    }

    return null;
  };

  const createModalOverlay = () => {
    const overlay = document.createElement("div");
    Object.assign(overlay.style, {
      position: "fixed",
      inset: "0",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      backgroundColor: "rgba(0, 0, 0, 0.6)",
      zIndex: "2147483647"
    });

    const dialog = document.createElement("form");
    dialog.setAttribute("role", "dialog");
    dialog.setAttribute("aria-modal", "true");
    Object.assign(dialog.style, {
      backgroundColor: "#fff",
      color: "#111",
      padding: "20px",
      borderRadius: "8px",
      minWidth: "320px",
      fontFamily: "sans-serif",
      display: "flex",
      flexDirection: "column",
      gap: "10px",
      boxShadow: "0 10px 30px rgba(0, 0, 0, 0.4)"
    });

    overlay.appendChild(dialog);

    const append = () => document.body.appendChild(overlay);
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", append, { once: true });
    } else {
      append();
    }

    return { overlay, dialog };
  };

  const showDialog = ({ title, contentHtml, buttons, onMount }) => new Promise((resolve) => {
    const { overlay, dialog } = createModalOverlay();

    const buttonsHtml = buttons
      .map((button, index) => `<button type="${button.type ?? "button"}" data-index="${index}" style="padding: 8px; cursor: pointer;">${button.label}</button>`)
      .join("");

    dialog.innerHTML = `
      <h2 style="margin: 0 0 4px; font-size: 16px;">${title}</h2>
      ${contentHtml}
      <div style="display: flex; gap: 8px; justify-content: flex-end; margin-top: 6px;">
        ${buttonsHtml}
      </div>
    `;

    dialog.setAttribute("aria-label", title);
    onMount?.(dialog);
    dialog.querySelector("input, button")?.focus();

    const close = (value) => {
      overlay.remove();
      resolve(value);
    };

    buttons.forEach((button, index) => {
      if (button.type === "submit") {
        return;
      }

      dialog.querySelector(`button[data-index="${index}"]`).addEventListener("click", () => close(button.onClick(dialog)));
    });

    dialog.addEventListener("submit", (event) => {
      event.preventDefault();
      const submitButton = buttons.find((button) => button.type === "submit");
      if (submitButton) close(submitButton.onClick(dialog));
    });
  });

  const showSettingsDialog = (defaults) => showDialog({
    title: "Score Scrape Settings",
    contentHtml: `
      <label style="display: flex; flex-direction: column; gap: 4px; font-size: 13px;">
        Submit Endpoint URL
        <input name="endpoint" type="url" required />
      </label>
      <label style="display: flex; flex-direction: column; gap: 4px; font-size: 13px;">
        Required Base Score
        <input name="baseScore" type="number" value="501" readonly required />
      </label>
      <label style="display: flex; flex-direction: column; gap: 4px; font-size: 13px;">
        Required Winning Legs
        <input name="winningLegs" type="number" value="3" readonly required />
      </label>
    `,
    onMount: (dialog) => {
      dialog.querySelector('[name="endpoint"]').value = defaults.endpoint;
      dialog.querySelector('[name="baseScore"]').value = defaults.baseScore;
      dialog.querySelector('[name="winningLegs"]').value = defaults.winningLegs;
    },
    buttons: [
      {
        label: "Save",
        type: "submit",
        onClick: (dialog) => {
          const settings = {
            endpoint: dialog.querySelector('[name="endpoint"]').value.trim(),
            baseScore: Number(dialog.querySelector('[name="baseScore"]').value),
            winningLegs: Number(dialog.querySelector('[name="winningLegs"]').value)
          };
          setCookie(SETTINGS_COOKIE_NAME, JSON.stringify(settings), SETTINGS_COOKIE_DAYS);
          return settings;
        }
      }
    ]
  });

  const showConfirmDialog = (message) => showDialog({
    title: "Score Scrape",
    contentHtml: `<p style="margin: 0; font-size: 13px; white-space: pre-line;"></p>`,
    onMount: (dialog) => {
      dialog.querySelector("p").textContent = message;
    },
    buttons: [
      { label: "Cancel", type: "button", onClick: () => false },
      { label: "Submit", type: "submit", onClick: () => true }
    ]
  });

   const savedSettings = loadSettingsFromCookie();
   const settingsPromise = Promise.resolve(savedSettings?.baseScore === 501 && savedSettings?.winningLegs === 3
     ? savedSettings : showSettingsDialog({ ...DEFAULT_SETTINGS, endpoint: savedSettings?.endpoint ?? "" }));

  const renderInstallIndicator = () => {
    if (typeof document === "undefined") {
      return;
    }

    if (document.getElementById(INSTALL_INDICATOR_ID)) {
      return;
    }

    const indicator = document.createElement("div");
    indicator.id = INSTALL_INDICATOR_ID;
    indicator.title = "Wrapper script installed";

    Object.assign(indicator.style, {
      position: "fixed",
      bottom: "12px",
      left: "12px",
      width: "14px",
      height: "14px",
      borderRadius: "50%",
      backgroundColor: "#22c55e",
      border: "4px solid #166534",
      boxSizing: "content-box",
      zIndex: "2147483647",
      pointerEvents: "none"
    });

    const append = () => {
      if (!document.body) {
        return;
      }

      document.body.appendChild(indicator);
    };

    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", append, { once: true });
      return;
    }

    append();
  };

  const safeParseJson = (value) => {
    if (typeof value !== "string") {
      return value;
    }

    try {
      return JSON.parse(value);
    } catch {
      return value;
    }
  };

  const notice = (message, retry = false) => showDialog({
    title: "Score Scrape", contentHtml: "<p role='status' style='white-space:pre-line'></p>",
    onMount: dialog => { dialog.querySelector("p").textContent = message; },
    buttons: [{ label: "Close", onClick: () => false }, ...(retry ? [{ label: "Retry", onClick: () => true }] : [])]
  });
  const submit = async (payload, synthetic) => {
    const settings = await settingsPromise;
    let endpoint;
    try {
      endpoint = new URL(settings.endpoint);
      requireValue(endpoint.protocol === "https:" && !endpoint.username && !endpoint.password);
    } catch { await notice("Invalid endpoint: configure an HTTPS relay URL and reload."); return; }
    const summary = payload.players.map(player => `${player.display_name}: ${player.legs_won} legs (avg ${player.stats?.match_average ?? "n/a"})`).join("\n");
    if (!await showConfirmDialog(`${sandbox ? "SANDBOX: no data will be posted.\n" : ""}501, first to 3, double out\n${summary}\nPlayed: ${payload.playedAt ?? "unknown"}\nMatch: ${payload.external_match_id}\nSend detailed result to ${endpoint.href}?`)) return;
    if (sandbox || synthetic) {
      await notice(synthetic ? "Synthetic source blocked: no data posted." : "Sandbox preview complete: no data posted.");
      return;
    }
    const key = JSON.stringify(payload);
    do {
      try {
        const response = await fetch(endpoint.href, {
          method: "POST", headers: { "Content-Type": "application/json" }, body: key,
          credentials: "omit", referrerPolicy: "no-referrer", redirect: "error", signal: AbortSignal.timeout(30000)
        });
        if (!response.ok) throw new Error("Relay rejected submission");
        submitted.add(key);
        await notice("Accepted for delivery. Not yet approved or published by the league.");
        return;
      } catch {
        if (!await notice("Submission failed or delivery is uncertain. Retry sends the identical result; league ingestion deduplicates it.", true)) return;
      }
    } while (true);
  };

  XMLHttpRequest.prototype.open = function (method, url, ...rest) {
    const previous = requests.get(this);
    if (previous?.listener) this.removeEventListener("load", previous.listener);
    requests.set(this, { method: String(method).toUpperCase(), matchId: matchURL(url) });
    return open.call(this, method, url, ...rest);
  };

  XMLHttpRequest.prototype.send = function (body) {
    const request = requests.get(this);
    if (request?.method === "GET" && request.matchId) {
      request.listener = function () {
        try {
          if (this.status < 200 || this.status >= 300 || matchURL(this.responseURL) !== request.matchId) return;
          const state = this.responseType === "json" ? this.response : safeParseJson(this.responseText);
          const payload = project(state, request.matchId);
          const key = JSON.stringify(payload);
          if (pending.has(key) || submitted.has(key)) return;
          pending.add(key);
          const synthetic = state.synthetic === true || typeof state.provenance === "string" && /synthetic|fictitious|test-only/i.test(state.provenance);
          queue = queue.then(() => submit(payload, synthetic)).catch(() => log("Capture submission unavailable")).finally(() => pending.delete(key));
        } catch { log("Match ignored: unsupported or invalid data"); }
      };
      this.addEventListener("load", request.listener, { once: true });
    }

    return send.call(this, body);
  };

  renderInstallIndicator();
})();
