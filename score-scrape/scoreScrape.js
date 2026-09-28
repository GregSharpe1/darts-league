(() => {
  if (window.__scoreScrapeInstalled) {
    console.log("[SCORE-SCRAPE]", "Score scrape script already installed");
    return;
  }

  window.__scoreScrapeInstalled = true;
  /* Backend endpoint that accepts a match summary and forwards it to the SQS queue */
  const SQS_SUBMIT_ENDPOINT_URL = "https://cicoc33r4h.execute-api.eu-west-1.amazonaws.com/results";
  const REQUIRED_BASE_SCORE = 501;
  const REQUIRED_WINNING_LEGS = 2;

  const open = XMLHttpRequest.prototype.open;
  const send = XMLHttpRequest.prototype.send;
  const INSTALL_INDICATOR_ID = "autodarts-wrapper-installed-indicator";
  const log = (...args) => console.log("[SCORE-SCRAPE]", ...args);
  const MATCH_VALIDATION_RULES = [
    {
      name: `baseScore`,
      test: (state) => state.settings?.baseScore === REQUIRED_BASE_SCORE
    },
    {
      name: `winningLegs`,
      test: (state) => Array.isArray(state.scores) && state.scores.some((score) => (score?.legs ?? 0) >= REQUIRED_WINNING_LEGS)
    }
  ];

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

  const evaluateMatchValidation = (state) => {
    const failedRules = MATCH_VALIDATION_RULES.filter((rule) => {
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

        const validation = evaluateMatchValidation(parsedResponse);

        if (!validation.isValid) {
          log("Match ignored (failed validation):", {
            matchId: parsedResponse.id ?? null,
            failedRules: validation.failedRules
          });
          return;
        }

        const summary = buildFinishedSummary(parsedResponse);
        const confirmationMessage = toConfirmationMessage(summary);
        const shouldSubmit = window.confirm(confirmationMessage);

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

        fetch(SQS_SUBMIT_ENDPOINT_URL, {
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
    }

    return send.call(this, body);
  };

  renderInstallIndicator();
})();