package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/paulstuart/dbutil"
)

// MakeREST will generate a REST handler for a DBObject
func MakeREST(gen dbutil.DBGen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newREST(gen.NewObj().(dbutil.DBObject), w, r)
	}
}

func apiKey(w http.ResponseWriter, r *http.Request) (*user, bool, error) {
	query := r.URL.Query()
	debug := false
	apiKey := r.Header.Get("X-API-KEY")
	if len(apiKey) == 0 {
		apiKey = query.Get("X-API-KEY")
	}
	//log.Println("check api key:", apiKey)
	delete(query, "X-API-KEY")
	if dbq, ok := query["debug"]; ok {
		debug, _ = strconv.ParseBool(dbq[0])
		//log.Println("*** debug set to:", debug)
		delete(query, "debug")
	} /*else {
		log.Println("*** debug:", debug)
	}
	*/
	r.URL.RawQuery = query.Encode()
	if debug {
		dbDebug(debug)
		defer dbDebug(false)
	}
	user, err := userFromAPIKey(apiKey)
	return &user, debug, err
}
func newREST(obj dbutil.DBObject, w http.ResponseWriter, r *http.Request) {
	db := datastore
	user, debug, err := apiKey(w, r)
	if err != nil && !insecure {
		log.Println("AUTH ERROR:", err)
		jsonError(w, err, http.StatusUnauthorized)
		return
	}
	//log.Println("auth ok")
	//spew.Println("\nUSER:", user, "\n")

	var id string
	i := strings.LastIndex(r.URL.Path, "/")
	if i < len(r.URL.Path)-1 {
		id = r.URL.Path[i+1:]
	}
	body := bodyCopy(r)
	//log.Println("DEBUG:", debug, "BODY:", body)
	if debug {
		log.Printf("(%s) PATH:%s ID:%s T:%T Q:%s BODY:%s", r.Method, r.URL.Path, id, obj, r.URL.RawQuery, body)
	}
	method := strings.ToUpper(r.Method)
	//log.Println("M:", method, "ID:", id)
	switch method {
	case "PUT", "POST":
		bodyString := bodyCopy(r)
		content := r.Header.Get("Content-Type")
		//fmt.Println("CONTENT:", content)
		if strings.Contains(content, "application/json") {
			if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
				fmt.Println("***** BODY:", bodyString)
				fmt.Println("***** ERR:", err)
				jsonError(w, err, http.StatusInternalServerError)
				return
			}
		} else {
			objFromForm(obj, r.Form)
		}
		if !insecure {
			obj.ModifiedBy(user.USR, time.Now())
		}
	case "OPTIONS", "HEAD":
		cors(w)
		fmt.Fprintln(w, "options:", bodyCopy(r))
	}

	// Make the change
	switch method {
	case "GET":
		if len(id) == 0 {
			q, val, err := fullQuery(r)
			if err != nil {
				log.Println("query error:", err)
				jsonError(w, err, http.StatusInternalServerError)
				return
			}
			if debug && len(val) > 0 {
				log.Println("Q VAL:", val)
			}
			list, err := db.ListQuery(obj, q, val...)
			if err != nil {
				log.Println("list error:", err)
				jsonError(w, err, http.StatusInternalServerError)
				return
			}
			//log.Println("LIST LEN:", len(list.([]dbutil.DBObject)))
			//log.Println("GET LIST:", list)
			sendJSON(w, list)
			return
		}
		// column name meta data
		if strings.HasSuffix(r.URL.Path, "/columns") {
			log.Println("GET PATH------->", r.URL.Path)
			data := struct {
				Columns []string
			}{
				Columns: obj.Names(),
			}
			sendJSON(w, data)
			return
		}

		if err := db.FindByID(obj, id); err != nil {
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		sendJSON(w, obj)
		return
	case "DELETE":
		if len(id) == 0 {
			err := fmt.Errorf("delete requires object id")
			jsonError(w, err, http.StatusBadRequest)
			return
		}
		if err := db.DeleteByID(obj, id); err != nil {
			sqle := err.(sqlite3.Error)
			log.Printf("DELETE ERROR (%T) code:%d ext:%d %s\n", sqle, sqle.Code, sqle.ExtendedCode, sqle)
			//if sqle.Code == sqlite3.ErrConstraintForeignKey {
			if sqle.Code == sqlite3.ErrConstraint {
				jsonError(w, err, http.StatusConflict)
				return
			}
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		msg := struct{ Msg string }{"deleted object id: " + id}
		sendJSON(w, msg)
		return
	case "PATCH":
		if len(id) == 0 {
			jsonError(w, "no ID specified", http.StatusBadRequest)
			return
		}

		// if we're patching the object, we first fetch the original copy
		// then overwrite the new fields supplied by the patch object
		if err := db.FindByID(obj, id); err != nil {
			log.Println("PATCH ERR 1:", err)
			jsonError(w, err, http.StatusInternalServerError)
			return
		}

		if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
			log.Println("PATCH ERR 2:", err)
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		if !insecure {
			obj.ModifiedBy(user.USR, time.Now())
		}
		if err := db.Save(obj); err != nil {
			log.Println("PATCH ERR 3:", err)
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		db.FindSelf(obj) // load what DB has for verification
		sendJSON(w, obj)
	case "PUT":
		log.Println("PUT OBJ:", obj)
		if err := db.Save(obj); err != nil {
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		db.FindSelf(obj) // load what DB has for verification
		sendJSON(w, obj)
	case "POST":
		if err := db.Add(obj); err != nil {
			log.Println("add error:", err)
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		db.FindSelf(obj) // load what DB has for verification
		sendJSON(w, obj)
	}
	// TODO: special debug flag for spew?
	//spew.Dump(obj)
}
