.PHONY: openapi

# Generates the OpenAPI (Swagger) specification using swaggo/swag.
openapi:
	@echo "+ Generating OpenAPI spec"
	@go install github.com/swaggo/swag/cmd/swag@latest
	@swag init -g cmd/main.go -o openapi
	@echo "Swagger files written to ./openapi"

.PHONY: client

# Generates a Python client library from the freshly generated OpenAPI specification.
client: openapi
	@echo "+ Generating Python client"
	@python -m pip install --quiet --upgrade openapi-python-client
	@openapi-python-client generate --path openapi/swagger.json --output python_client --force
	@echo "Python client written to ./python_client"