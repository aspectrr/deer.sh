.PHONY: demo-start demo-stop demo-reset

demo-start:
	bash scripts/start.sh

demo-stop:
	bash scripts/stop.sh

demo-reset:
	bash scripts/stop.sh || true
	limactl delete deer-source deer-sandbox --force 2>/dev/null || true
	docker compose -f docker-compose.yml down -v 2>/dev/null || true
	@echo "Demo fully reset. Run 'make demo-start' to start fresh."
