package main

import (
	"fmt"
	"os"

	"github.com/paulstuart/dbutil"
)

var (
	datastore   dbutil.DBU
	errNoDB     = fmt.Errorf("no database")
	errReadOnly = fmt.Errorf("database is read only")
	readOnly    = false
)

func readable() error {
	if datastore.DB == nil {
		return errNoDB
	}
	return nil
}

func writable() error {
	if datastore.DB == nil {
		return errNoDB
	}
	if readOnly {
		return errReadOnly
	}
	return nil
}

func dbExec(query string, args ...interface{}) error {
	if err := writable(); err != nil {
		return err
	}
	_, err := datastore.DB.Exec(query, args...)
	return err
}

func dbRows(q string, args ...interface{}) (results []string, err error) {
	if err := readable(); err != nil {
		return []string{}, err
	}
	return datastore.Rows(q, args...)
}

func dbRow(query string, args ...interface{}) ([]string, error) {
	if err := readable(); err != nil {
		return []string{}, err
	}
	return datastore.Row(query, args...)
}

func dbDebug(enable bool) {
	dbutil.Debug(enable)
}

func dbList(kind dbutil.DBObject) (interface{}, error) {
	if err := readable(); err != nil {
		return nil, errNoDB
	}
	return datastore.List(kind)
}

/*
func dbObjectList(kind interface{}) (interface{}, error) {
	if err := readable(); err != nil {
		return nil, err
	}
	return datastore.ObjectList(kind)
}

func dbObjectListQuery(kind interface{}, extra string, args ...interface{}) (interface{}, error) {
	if err := readable(); err != nil {
		return nil, err
	}
	return datastore.ObjectListQuery(kind, extra, args...)
}

func dbStats() []string {
	if err := readable(); err != nil {
		return []string{}
	}
	return datastore.Stats()
}

func dbPragmas() (map[string]string, error) {
	return datastore.Pragmas()
}

*/

func dbOpen(file string, create bool) (err error) {
	datastore, err = dbutil.Open(file, true)
	return err
}

func dbPrep() {
	var fresh bool
	var err error
	//log.Println("DBFILE:", dbFile)
	if _, err = os.Stat(dbFile); os.IsNotExist(err) {
		fresh = true
	}
	datastore, err = dbutil.Open(dbFile, true)
	if err != nil {
		panic(err)
	}
	if fresh {
		err = datastore.File(sqlSchema)
		if err != nil {
			panic(err)
		}
	}
	if err := dbExec("PRAGMA foreign_keys = ON"); err != nil {
		panic(err)
	}
}

/*
func backups(freq int, to string) {
	if _, err := os.Stat(to); err != nil {
		to = filepath.Join(execDir, to)
		if _, err := os.Stat(to); err != nil {
			log.Fatal(err)
		}
	}
	layout := "2006-01-02_15-04-05"
	t := time.NewTicker(time.Minute * time.Duration(freq))
	for {
		select {
		case now := <-t.C:
			to := filepath.Join(to, now.Format(layout)+".db")
			dbBackup(to)
		}
	}

}

func dbBackup(to string) error {
	if datastore.DB == nil {
		return errNoDB
	}
	return datastore.Backup(to, nil)
}
*/

func dbAdd(o dbutil.DBObject) error {
	if err := writable(); err != nil {
		return err
	}
	return datastore.Add(o)
}

func dbClose() error {
	if datastore.DB == nil {
		return errNoDB
	}
	return datastore.DB.Close()
}

func dbFindByID(o dbutil.DBObject, id interface{}) error {
	if err := readable(); err != nil {
		return err
	}
	return datastore.FindByID(o, id)
}

func dbSave(o dbutil.DBObject) error {
	if err := writable(); err != nil {
		return err
	}
	return datastore.Save(o)
}

func dbObjectLoad(obj interface{}, extra string, args ...interface{}) error {
	if datastore.DB == nil {
		return errNoDB
	}
	return datastore.ObjectLoad(obj, extra, args...)
}

func dbDelete(o dbutil.DBObject) error {
	if err := writable(); err != nil {
		return err
	}
	return datastore.Delete(o)
}

/*
func dbObjectUpdate(obj interface{}) error {
	if err := writable(); err != nil {
		return err
	}
	return datastore.ObjectUpdate(obj)
}

func dbObjectInsert(obj interface{}) (int64, error) {
	if err := writable(); err != nil {
		return -1, err
	}
	return datastore.ObjectInsert(obj)
}

func dbStream(fn func([]string, int, []interface{}), query string, args ...interface{}) error {
	if err := readable(); err != nil {
		return err
	}
	return datastore.Stream(fn, query, args...)
}

/*
//func (db DBU) Stream(fn func([]string, int, []interface{}), query string, args ...interface{}) error {
func dbStreamTab(w io.Writer, query string, args ...interface{}) error {
	if err := readable(); err != nil {
		return err
	}
	return datastore.StreamTab(w, query, args...)
}
func dbBackedUp() {}

func dbChanged() {}

func dbCmd() {}
*/
/*
func dbFind(o dbutil.DBObject, keys dbutil.QueryKeys) error {
	return datastore.Find(o, keys)
}

func dbFindSelf(o dbutil.DBObject) error {
	if err := readable(); err != nil {
		return err
	}
	return datastore.FindSelf(o)
}

func dbLoad(o dbutil.DBObject) error {
	if err := readable(); err != nil {
		return err
	}
	return datastore.FindSelf(o)
}

func dbGetInt(q string, args ...interface{}) (int, error) {
	if err := readable(); err != nil {
		return -1, err
	}
	return datastore.GetInt(q, args...)
}

func dbInsert(q string, args ...interface{}) (i int64, e error) {
	if err := writable(); err != nil {
		return -1, err
	}
	return datastore.Insert(q, args...)
}

func dbObjectDelete(obj interface{}) error {
	if err := writable(); err != nil {
		return err
	}
	return datastore.ObjectDelete(obj)
}

func dbReplace(o dbutil.DBObject) error {
	if err := writable(); err != nil {
		return err
	}
	return datastore.Replace(o)
}



func dbStreamJSON(w io.Writer, query string, args ...interface{}) error {
	if err := readable(); err != nil {
		return err
	}
	return datastore.StreamJSON(w, query, args...)
}
func dbStreamCSV(w io.Writer, query string, args ...interface{}) error {
	if err := readable(); err != nil {
		return err
	}
	return datastore.StreamCSV(w, query, args...)
}


func dbTable(query string, args ...interface{}) (*dbutil.Table, error) {
	if err := readable(); err != nil {
		return &dbutil.Table{}, err
	}
	return datastore.Table(query, args...)
}

func dbUpdate(query string, args ...interface{}) (i int64, e error) {
	if err := writable(); err != nil {
		return -1, err
	}
	return datastore.Update(query, args...)
}

func dbVersion() {}

func trimDate(s string) string {
	const date = len(dateLayout)
	if len(s) < date {
		return s
	}
	s = s[:date]
	if s == "0001-01-01" {
		return ""
	}
	return s
}

func trimTime(s string) string {
	const t = len(timeLayout)
	if len(s) < t {
		return s
	}
	s = s[:t]
	if s == "0001-01-01" {
		return ""
	}
	return s
}
*/
