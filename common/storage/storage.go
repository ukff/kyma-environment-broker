package storage

import (
	"database/sql"
	"fmt"
)

func StringToSQLNullString(input string) sql.NullString {
	fmt.Println("==StringToSQLNullString==")
	fmt.Println(input)
	result := sql.NullString{}

	if input != "" {
		result.String = input
		result.Valid = true
	}

	fmt.Println(result.String)
	fmt.Println("==StringToSQLNullString==")
	return result
}

func SQLNullStringToString(input sql.NullString) string {
	if input.Valid {
		return input.String
	}

	return ""
}
