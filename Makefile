createmigration:
	migrate create -ext=sql -dir=db/migrations -seq init

migrate:
	migrate -path=db/migrations -database "mysql://root:root@tcp(localhost:3306)/orders" up

migratedown:
	migrate -path=db/migrations -database "mysql://root:root@tcp(localhost:3306)/orders" down

.PONY: createmigration migrate migratedown