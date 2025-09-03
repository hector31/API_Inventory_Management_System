module github.com/melibackend/store-s1

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.10
	github.com/melibackend/shared v0.0.0
)

require github.com/joho/godotenv v1.5.1 // indirect

replace github.com/melibackend/shared => ../../shared
