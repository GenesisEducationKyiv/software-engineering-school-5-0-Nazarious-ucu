module github.com/Nazarious-ucu/weather-subscription-api/notification

go 1.24.5

replace github.com/Nazarious-ucu/weather-subscription-api/pkg => ../pkg

require (
	github.com/Nazarious-ucu/weather-subscription-api/pkg v0.0.0-00010101000000-000000000000
	github.com/joho/godotenv v1.5.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/stretchr/testify v1.10.0
	github.com/wagslane/go-rabbitmq v0.15.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
