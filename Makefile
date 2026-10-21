DB_PATH ?= ./recipes.db
SOURCE  ?= https://www.liveeatlearn.com/vegetable-pot-pie-casserole/
PORT    ?= 8080
DEBUG   ?= false
HOST    ?= 100.56.133.20
ROOT	?= www.liveeatlearn.com

.PHONY: run-api run-scraper build-ts docker-build docker-up docker-down docker-scraper deploy tf-init tf-plan tf-apply tf-destroy

run-api:
	DB_PATH=$(DB_PATH) go run ./cmd/recipe-api

run-scraper:
	DB_PATH=$(DB_PATH) go run ./cmd/scraper -source=$(SOURCE) -debug=$(DEBUG) -rootHost=$(ROOT)

## Deploy
provision:
	./scripts/provision.sh $(HOST)

push-db:
	sqlite3 $(DB_PATH) "INSERT INTO recipe_search(recipe_search) VALUES('rebuild'); PRAGMA wal_checkpoint(TRUNCATE);"
	scp $(DB_PATH) ec2-user@$(HOST):/tmp/recipes.db
	ssh ec2-user@$(HOST) "sudo systemctl stop recipe-app && \
		sudo rm -f /var/lib/recipe-app/recipes.db-shm /var/lib/recipe-app/recipes.db-wal && \
		sudo mv /tmp/recipes.db /var/lib/recipe-app/recipes.db && \
		sudo chown recipe-app:recipe-app /var/lib/recipe-app/recipes.db && \
		sudo systemctl start recipe-app"

deploy:
	GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o /tmp/recipe-api ./cmd/recipe-api
	scp /tmp/recipe-api ec2-user@$(HOST):/tmp/recipe-api
	ssh ec2-user@$(HOST) "sudo mv /tmp/recipe-api /opt/recipe-app/recipe-api && sudo chmod +x /opt/recipe-app/recipe-api && sudo systemctl restart recipe-app"

## Terraform
TF_DIR := terraform/components/application
TF_VARS := $(TF_DIR)/workspaces/production.tfvars

tf-init:
	cd $(TF_DIR) && terraform init

tf-plan:
	cd $(TF_DIR) && terraform plan -var-file=workspaces/production.tfvars

tf-apply:
	cd $(TF_DIR) && terraform apply -var-file=workspaces/production.tfvars

tf-destroy:
	cd $(TF_DIR) && terraform destroy -var-file=workspaces/production.tfvars

## Docker
docker-build:
	docker build --build-arg CMD=recipe-api -t recipe-api .
	docker build --build-arg CMD=scraper    -t recipe-scraper .

docker-up:
	docker compose up --build api

docker-scraper:
	docker compose --profile scraper up scraper

docker-down:
	docker compose down

## curl examples
# Create a recipe:
#   curl -X POST http://localhost:8080/recipe \
#        -H 'Content-Type: application/json' \
#        -d '{
#              "name": "Avocado Toast",
#              "ingredients": [
#                {"name": "1 ripe avocado"},
#                {"name": "2 slices sourdough bread"},
#                {"name": "pinch of salt"}
#              ],
#              "instructions": [
#                {"step": "Toast the bread until golden."},
#                {"step": "Halve the avocado and remove the pit."},
#                {"step": "Mash the avocado onto the toast and season with salt."}
#              ]
#            }'
#
# Get a recipe by ID:
#   curl 'http://localhost:8080/recipe/1'
#
# Search recipes:
#   curl 'http://localhost:8080/search?q=avocado'
