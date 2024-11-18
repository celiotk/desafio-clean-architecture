createmigration:
	migrate create -ext=sql -dir=db/migrations -seq init

migrateup:
	migrate -path=db/migrations -database "mysql://root:root@tcp(localhost:3306)/orders" up

migratedown:
	migrate -path=db/migrations -database "mysql://root:root@tcp(localhost:3306)/orders" down

.PHONY: createmigration migrateup migratedown run

run: migrateup
	# Start the Go application
	go run cmd/ordersystem/main.go cmd/ordersystem/wire_gen.go