package main

import (
	"github.com/pkg/errors"
	"github.com/go-sql-driver/mysql"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// these codes come from github.com/qaqcatz/impomysql.

// IsStarted: check whether the DBMS is started through Connector
func IsStarted(hostPort string, user string, password string, defaultDB string) error {
	port, err := strconv.Atoi(hostPort)
	if err != nil {
		panic("[IsStarted]parse port error: " + err.Error() + ": " + hostPort)
	}
	conn, err := NewConnector("127.0.0.1", port, user, password, "")
	if err != nil {
		return err
	}
	res := conn.ExecSQL("USE " + defaultDB + ";")
	if res.Err != nil {
		return res.Err
	}
	res = conn.ExecSQL("SELECT 1;")
	if res.Err != nil {
		return res.Err
	}
	return nil
}

// Connector: connect to MySQL, execute raw sql statements, return raw execution result or error.
type Connector struct {
	Host            string
	Port            int
	Username        string
	Password        string
	DbName          string
	db              *sql.DB
}

// NewConnector: create Connector. CREATE DATABASE IF NOT EXISTS dbname + USE dbname when dbname != ""
func NewConnector(host string, port int, username string, password string, dbname string) (*Connector, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?allowOldPasswords=true",
		username, password, host, port, "")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "[NewConnector]open dsn error")
	}
	conn := &Connector{
		Host:            host,
		Port:            port,
		Username:        username,
		Password:        password,
		DbName:          dbname,
		db:              db,
	}
	if dbname != "" {
		// CREATE DATABASE IF NOT EXISTS conn.DbName
		result := conn.ExecSQL("CREATE DATABASE IF NOT EXISTS " + conn.DbName)
		if result.Err != nil {
			return nil, result.Err
		}
		// USE conn.DbName
		result = conn.ExecSQL("USE " + conn.DbName)
		if result.Err != nil {
			return nil, result.Err
		}
	}
	return conn, nil
}

// Connector.ExecSQL: execute sql, return *Result.
func (conn *Connector) ExecSQL(sql string) *Result {
	startTime := time.Now()
	rows, err := conn.db.Query(sql)
	if err != nil {
		return &Result{
			Err: errors.Wrap(err, "[Connector.ExecSQL]execute sql error"),
		}
	}
	defer rows.Close()

	result := &Result{
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Rows: make([][]string, 0),
		Err: nil,
	}
	for rows.Next() {
		columnTypes, err := rows.ColumnTypes()
		if err != nil {
			return &Result{
				Err: errors.Wrap(err, "[Connector.ExecSQL]get column type error"),
			}
		}
		if len(result.ColumnNames) == 0 {
			for _, columnType := range columnTypes {
				result.ColumnNames = append(result.ColumnNames, columnType.Name())
				result.ColumnTypes = append(result.ColumnTypes, columnType.DatabaseTypeName())
			}
		} else {
			if len(columnTypes) != len(result.ColumnNames) {
				return &Result{
					Err: errors.New("[Connector.ExecSQL]|columnTypes|("+strconv.Itoa(len(columnTypes))+") != "+
						"|columnNames|("+strconv.Itoa(len(result.ColumnNames))+")"),
				}
			}
			for i, columnType := range columnTypes {
				if columnType.Name() != result.ColumnNames[i] {
					return &Result{
						Err: errors.New("[Connector.ExecSQL]columnType.Name()("+columnType.Name()+") != "+
							"result.ColumnNames[i]("+result.ColumnNames[i]+")"),
					}
				}
				if columnType.DatabaseTypeName() != result.ColumnTypes[i] {
					return &Result{
						Err: errors.New("[Connector.ExecSQL]columnType.DatabaseTypeName()("+columnType.DatabaseTypeName()+") != "+
							"result.ColumnTypes[i]("+result.ColumnTypes[i]+")"),
					}
				}
			}
		}

		// gorm cannot convert NULL to string, we should use []byte
		data := make([][]byte, len(columnTypes))
		dataI := make([]interface{}, len(columnTypes))
		for i, _ := range data {
			dataI[i] = &data[i]
		}
		err = rows.Scan(dataI...)
		if err != nil {
			return &Result{
				Err: errors.Wrap(err, "[Connector.ExecSQL]scan rows error"),
			}
		}

		dataS := make([]string, len(columnTypes))
		for i, _ := range data {
			if data[i] == nil {
				dataS[i] = "NULL"
			} else {
				dataS[i] = string(data[i])
			}
		}
		result.Rows = append(result.Rows, dataS)
	}
	if rows.Err() != nil {
		return &Result{
			Err: errors.Wrap(rows.Err(), "[Connector.ExecSQL]rows error"),
		}
	}

	result.Time = time.Since(startTime)
	return result
}

// Result:
//
// query result, for example:
//   +-----+------+------+
//   | 1+2 | ID   | NAME | -> ColumnNames: 1+2,    ID,  NAME
//   +-----+------+------+ -> ColumnTypes: BIGINT, INT, TEXT
//   |   3 |    1 | H    | -> Rows[0]:     3,      1,   H
//   |   3 |    2 | Z    | -> Rows[1]:     3,      2,   Z
//   |   3 |    3 | Y    | -> Rows[2]:     3,      3,   Y
//   +-----+------+------+
// or error, for example:
//  Err: ERROR 1054 (42S22): Unknown column 'T' in 'field list'
//
// note that:
//
// len(ColumnNames) = len(ColumnTypes) = len(Rows[i]);
//
// if the statement is not SELECT, then the ColumnNames, ColumnTypes and Rows are empty
type Result struct {
	ColumnNames []string
	ColumnTypes []string
	Rows [][]string
	Err error
	Time time.Duration // total time
}

func (result *Result) ToString() string {
	str := ""
	str += "ColumnName(ColumnType)s: "
	for i, columnName := range result.ColumnNames {
		str += " " + columnName + "(" + result.ColumnTypes[i] + ")"
	}
	str += "\n"
	for i, row := range result.Rows {
		str += "row " + strconv.Itoa(i) + ":"
		for _, data := range row {
			str += " " + data
		}
		str += "\n"
	}
	if result.Err != nil {
		str += "Error: " + result.Err.Error() + "\n"
	}

	str += result.Time.String()
	return str
}

func (result *Result) GetErrorCode() (int, error) {
	if result.Err == nil {
		return -1, errors.New("[Result.GetErrorCode]result.Err == nil")
	}
	rootCause := errors.Cause(result.Err)
	if driverErr, ok := rootCause.(*mysql.MySQLError); ok { // Now the error number is accessible directly
		return int(driverErr.Number), nil
	} else {
		return -1, errors.New("[Result.GetErrorCode]not *mysql.MySQLError " + reflect.TypeOf(rootCause).String())
	}
}