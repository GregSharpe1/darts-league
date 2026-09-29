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
    winningLegs: 2
  };

  const open = XMLHttpRequest.prototype.open;
  const send = XMLHttpRequest.prototype.send;
  const INSTALL_INDICATOR_ID = "autodarts-wrapper-installed-indicator";
  const log = (...args) => console.log("[SCORE-SCRAPE]", ...args);
  const buildMatchValidationRules = (settings) => [
    {
      name: `baseScore`,
      test: (state) => state.settings?.baseScore === settings.baseScore
    },
    {
      name: `winningLegs`,
      test: (state) => Array.isArray(state.scores) && state.scores.some((score) => (score?.legs ?? 0) >= settings.winningLegs)
    }
  ];

  const getCookie = (name) => {
    const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
    return match ? decodeURIComponent(match[1]) : null;
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
        return parsed;
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

    onMount?.(dialog);

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
      close(submitButton.onClick(dialog));
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
        <input name="baseScore" type="number" required />
      </label>
      <label style="display: flex; flex-direction: column; gap: 4px; font-size: 13px;">
        Required Winning Legs
        <input name="winningLegs" type="number" required />
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

  const settingsPromise = Promise.resolve(loadSettingsFromCookie() ?? showSettingsDialog(DEFAULT_SETTINGS));

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

  const toPlayerSummary = (state, index) => ({
    name: state.players?.[index]?.name ?? null,
    legsWon: state.scores?.[index]?.legs ?? null,
    matchAverage:
      state.stats?.[index]?.matchStats?.average ??
      state.matchStats?.find((entry) => entry.playerId === state.players?.[index]?.id)?.average ??
      state.matchStats?.[index]?.average ??
      null
  });

  const buildFinishedSummary = (state) => ({
    matchId: state.id ?? null,
    player1: toPlayerSummary(state, 0),
    player2: toPlayerSummary(state, 1)
  });

  const formatAverage = (value) => {
    if (typeof value !== "number" || !Number.isFinite(value)) {
      return "n/a";
    }

    return value.toFixed(2);
  };

  const toConfirmationMessage = (summary) => {
    const player1Name = summary.player1?.name ?? "Player 1";
    const player2Name = summary.player2?.name ?? "Player 2";
    const player1Legs = summary.player1?.legsWon ?? "n/a";
    const player2Legs = summary.player2?.legsWon ?? "n/a";
    const player1Average = formatAverage(summary.player1?.matchAverage);
    const player2Average = formatAverage(summary.player2?.matchAverage);

    return [
      "This game meets league match criteria.",
      "",
      `Match ID: ${summary.matchId ?? "n/a"}`,
      `${player1Name}: ${player1Legs} legs (avg ${player1Average})`,
      `${player2Name}: ${player2Legs} legs (avg ${player2Average})`,
      "",
      "Is this correct, and should the scores be submitted?"
    ].join("\n");
  };

  const evaluateMatchValidation = (state, settings) => {
    const failedRules = buildMatchValidationRules(seattings).filter((rule) => {
      try {
        return !rule.test(state);
      } catch {
        return true;
      }
    }).map((rule) => rule.name);

    return {
      isValid: failedRules.length === 0,
      failedRules
    };
  };

  XMLHttpRequest.prototype.open = function (method, url, ...rest) {
    this._trackedUrl = url;
    this._trackedMethod = method;
    return open.call(this, method, url, ...rest);
  };

  XMLHttpRequest.prototype.send = function (body) {
    const url = this._trackedUrl || "";
    const match = /\/as\/v0\/matches\/[^/]+\/stats(?:\?|$)/.test(url);

    if (match) {
      this.addEventListener("load", function () {
        const parsedResponse = safeParseJson(this.responseText);

        if (!parsedResponse || typeof parsedResponse !== "object") {
          return;
        }

        settingsPromise.then((settings) => {
          const validation = evaluateMatchValidation(parsedResponse, settings);

          if (!validation.isValid) {
            log("Match ignored (failed validation):", {
              matchId: parsedResponse.id ?? null,
              failedRules: validation.failedRules
            });
            return;
          }

          const summary = buildFinishedSummary(parsedResponse);
          const confirmationMessage = toConfirmationMessage(summary);

          showConfirmDialog(confirmationMessage).then((shouldSubmit) => {
            if (!shouldSubmit) {
              log("Valid match rejected by user confirmation:", {
                matchId: summary.matchId
              });
              return;
            }

            log("Valid match confirmed for submission:", {
              matchId: summary.matchId
            });
            log("Match summary:", summary);

            fetch(settings.endpoint, {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify(summary)
            })
              .then((response) => {
                if (!response.ok) {
                  throw new Error(`Submission endpoint responded with ${response.status}`);
                }

                log("Match summary sent for queueing:", { matchId: summary.matchId });
              })
              .catch((error) => {
                log("Failed to send match summary:", { matchId: summary.matchId, error: String(error) });
              });
          });
        });
      });
    }

    return send.call(this, body);
  };

  renderInstallIndicator();
})();