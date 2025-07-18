package hook

import (
	"errors"
	"net/http"

	"github.com/daodao97/xgo/xadmin"
	"github.com/daodao97/xgo/xdb"
)

func RegHook() {
	xadmin.RegBeforeUpdate("operator", OperatorBeforeUpdateHook)
	xadmin.RegBeforeCreate("operator", OperatorBeforeCreateHook)
	xadmin.RegAfterGet("operator", OperatorAfterGetHook)
}

func OperatorBeforeCreateHook(r *http.Request, createData xdb.Record) (xdb.Record, error) {
	password := createData.GetString("password")
	if password == "" {
		return nil, errors.New("密码不能为空")
	}
	createData["password"] = xadmin.PasswordHash(password)
	return createData, nil
}

func OperatorAfterGetHook(r *http.Request, record xdb.Record) xdb.Record {
	record["password"] = ""
	return record
}

func OperatorBeforeUpdateHook(r *http.Request, updateData xdb.Record) (xdb.Record, error) {
	if updateData.GetString("password") == "" {
		return updateData, nil
	}
	password := updateData.GetString("password")
	if password == "" {
		return updateData, nil
	}
	updateData["password"] = xadmin.PasswordHash(password)

	return updateData, nil
}
