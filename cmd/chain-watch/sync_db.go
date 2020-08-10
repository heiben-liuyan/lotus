package main

import (
	"database/sql"

	"github.com/gwaylib/database"
	"github.com/gwaylib/errors"
)

func GetCurHeight() (int64, error) {
	mdb := GetDB()
	height := sql.NullInt64{}
	if err := database.QueryElem(mdb,
		&height,
		"SELECT max(height) FROM sync_proc",
	); err != nil {
		return 0, errors.As(err, height)
	}
	return height.Int64, nil
}

func AddCurHeight(height int64) error {
	mdb := GetDB()
	if _, err := mdb.Exec("INSERT INTO sync_proc(height)VALUES(?)", height); err != nil {
		return errors.As(err, height)
	}
	return nil
}
