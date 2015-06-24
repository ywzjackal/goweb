package dao

import "database/sql"

type Dao interface {
	FindById(id interface{}) (model interface{}, err error)
	FindAll() (model interface{}, err error)
	SaveOrUpdate(model interface{}) (err error)
	Delete(model interface{}) (err error)
	Load(model interface{}) (err error)
	Loads(model interface{}) (err error)
}

type daoPostgre struct {
	Dao
	db         *sql.DB
	model      modelWrap
	qsFindById string
	qsFindAll  string
}

func NewDaoPostgres(db *sql.DB, tableName string, model interface{}) Dao {
	return newDaoPostgres(db, tableName, model)
}

func newDaoPostgres(db *sql.DB, tableName string, model interface{}) *daoPostgre {
	dao := &daoPostgre{}
	dao.model.Init(model, tableName)
	dao.qsFindById = genFindById(dao.model.TableName, dao.model.Fields)
	dao.qsFindAll = genFindAll(dao.model.TableName, dao.model.Fields)
	return dao
}

func (d *daoPostgre) FindById(id interface{}) (model interface{}, err error) {
	var (
		row *sql.Row     = nil
		tar *modelTarget = d.model.NewTarget()
	)
	row = d.db.QueryRow(d.qsFindById, id)
	if err = row.Scan(tar.ScanInter...); err != nil {
		return
	}
	model = tar.Inter
	return
}

func (d *daoPostgre) FindAll() (models interface{}, err error) {
	var (
		rows *sql.Rows     = nil
		tars []interface{} = make([]interface{}, 0, 0)
	)
	rows, err = d.db.Query(d.qsFindById)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		tar := d.model.NewTarget()
		if err = rows.Scan(tar.ScanInter...); err != nil {
			return
		}
		tars = append(tars, tar.Inter)
	}
	models = tars
	return
}

func (d *daoPostgre) SaveOrUpdate(model interface{}) (err error) {
	return
}

func (d *daoPostgre) Delete(model interface{}) (err error) {
	return
}

func genFindById(tableName string, fields []*modelField) string {
	var (
		qs string = "SELECT"
	)
	for i, f := range fields {
		selec_t := f.Select
		if selec_t == "" {
			selec_t = "\"" + f.Name + "\""
		}
		if i == 0 {
			qs += " " + selec_t
		} else {
			qs += ", " + selec_t
		}
	}
	qs += " FROM \"" + tableName + "\" " + "WHERE \"id\" = $1;"
	return qs
}

func genFindAll(tableName string, fields []*modelField) string {
	var (
		qs string = "SELECT"
	)
	for i, f := range fields {
		selec_t := f.Select
		if selec_t == "" {
			selec_t = "\"" + f.Name + "\""
		}
		if i == 0 {
			qs += " " + selec_t
		} else {
			qs += ", " + selec_t
		}
	}
	qs += " FROM \"" + tableName + "\";"
	return qs
}

func genSaveOrUpdate(tableName string, fields []modelField) string {
	var (
		qs string = "SELECT"
	)
	for i, f := range fields {
		selec_t := f.Select
		if selec_t == "" {
			selec_t = "\"" + f.Name + "\""
		}
		if i == 0 {
			qs += " " + selec_t
		} else {
			qs += ", " + selec_t
		}
	}
	qs += " FROM \"" + tableName + "\";"
	return qs
}
