db_down:
	goose postgres postgres://postgres:postgres@localhost:5432/chirpy down --dir sql/schema
db_up:
	goose postgres postgres://postgres:postgres@localhost:5432/chirpy up --dir sql/schema
