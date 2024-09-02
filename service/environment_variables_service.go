package service

import database "github.com/nifetency/nife.io/internal/pkg/db/mysql"

func GetEnvironmentVariables(keyVar string) (string, error) {
	query := "SELECT key_variable, value FROM environment_variables where key_variable = ?"

	selDB, err := database.Db.Query(query, keyVar)
	if err != nil {
		return "", err
	}
	var keyVariable string
	var value string

	defer selDB.Close()
	selDB.Next()
	err = selDB.Scan(&keyVariable, &value)
	if err != nil {
		return "", err
	}
	return value, nil
}

