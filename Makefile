.PHONY: openapi

# Generates the OpenAPI (Swagger) specification using swaggo/swag.
openapi:
	@echo "+ Generating OpenAPI spec"
	@go install github.com/swaggo/swag/cmd/swag@latest
	@swag init -g cmd/main.go -o openapi
	@echo "Swagger files written to ./openapi"