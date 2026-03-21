CHART_DIR := helm/darts-league
VALUES_FILE ?= helm/darts-league/values-dev.yaml
RELEASE_NAME ?= darts-league
NAMESPACE ?= darts-league
HELM ?= helm

.PHONY: help helm-lint helm-template helm-diff helm-deploy helm-uninstall helm-test

help:
	@printf "Available targets:\n"
	@printf "  helm-lint      Lint the Helm chart with the selected values file\n"
	@printf "  helm-template  Render the Helm chart locally\n"
	@printf "  helm-diff      Show the Helm diff for the release\n"
	@printf "  helm-deploy    Upgrade or install the Helm release\n"
	@printf "  helm-uninstall Remove the Helm release\n"
	@printf "  helm-test      Run Helm test hooks for the release\n"
	@printf "\nOverrides:\n"
	@printf "  VALUES_FILE=%s\n" "$(VALUES_FILE)"
	@printf "  RELEASE_NAME=%s\n" "$(RELEASE_NAME)"
	@printf "  NAMESPACE=%s\n" "$(NAMESPACE)"

helm-lint:
	$(HELM) lint $(CHART_DIR) -f $(VALUES_FILE)

helm-template:
	$(HELM) template $(RELEASE_NAME) $(CHART_DIR) -n $(NAMESPACE) -f $(VALUES_FILE)

helm-diff:
	$(HELM) diff upgrade --install $(RELEASE_NAME) $(CHART_DIR) -n $(NAMESPACE) -f $(VALUES_FILE)

helm-deploy:
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_DIR) -n $(NAMESPACE) --create-namespace -f $(VALUES_FILE)

helm-uninstall:
	$(HELM) uninstall $(RELEASE_NAME) -n $(NAMESPACE)

helm-test:
	$(HELM) test $(RELEASE_NAME) -n $(NAMESPACE)
