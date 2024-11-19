createmigration:
	migrate create -ext=sql -dir=db/migrations -seq init

.PHONY: migrateup