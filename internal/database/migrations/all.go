package migrations

func All() []Migration {
	return []Migration{
		SchemaMigrations{},
	}
}
