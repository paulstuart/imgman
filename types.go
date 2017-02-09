package main

import (
	"database/sql/driver"
	"fmt"
	"time"
)

//go:generate dbgen

type jsonTime time.Time

const sqlTime = `2006-01-02 15:04:05`

func (d jsonTime) MarshalJSON() ([]byte, error) {
	t := time.Time(d)
	if t.IsZero() {
		return []byte(`""`), nil
	}
	stamp := fmt.Sprintf(`"%s"`, t.Format(sqlTime))
	return []byte(stamp), nil
}

func (d *jsonTime) UnmarshalJSON(in []byte) error {
	s := string(in)
	if len(in) < 3 {
		return nil
	}
	if d == nil {
		d = new(jsonTime)
	}
	//const longform = `"2006-01-02T15:04:05.000Z"`
	const longform = `"2006-01-02T15:04:05.000Z"`
	if len(s) == len(longform) {
		t, err := time.Parse(longform, s)
		*d = jsonTime(t)
		return err
	}
	/*
		t, err := time.Parse(`"2006-1-2"`, s)
		if err != nil {
			t, err = time.Parse(`"2006/1/2"`, s)
		}
	*/
	t, err := time.Parse(sqlTime, s)
	if err == nil {
		*d = jsonTime(t)
	}
	return err
}

// Scan implements the Scanner interface.
func (d *jsonTime) Scan(value interface{}) error {
	*d = jsonTime(value.(time.Time))
	return nil
}

// Value implements the driver Valuer interface.
func (d *jsonTime) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	return time.Time(*d), nil
}

type user struct {
	USR    int64   `sql:"usr" key:"true" table:"users"`
	RealID int64   // when emulating another user, retain real identity
	Login  *string `sql:"login"`
	First  *string `sql:"firstname"`
	Last   *string `sql:"lastname"`
	Email  *string `sql:"email"`
	APIKey *string `sql:"apikey"`
	Level  int     `sql:"admin"`
}

func (u *user) apiKey() string {
	if u.APIKey == nil {
		return ""
	}
	return *u.APIKey
}

// FullUser has *all* user fields exposed
type fullUser struct {
	USR      int64   `sql:"usr" key:"true" table:"users"`
	RealID   int64   // when emulating another user, retain real identity
	Login    *string `sql:"login"`
	First    *string `sql:"firstname"`
	Last     *string `sql:"lastname"`
	Email    *string `sql:"email"`
	APIKey   *string `sql:"apikey"`
	Password *string `sql:"pw_hash"`
	Salt     *string `sql:"pw_salt"`
	Level    int     `sql:"admin"`
}

type site struct {
	STI      int64     `sql:"sti" key:"true" table:"sites"`
	Name     *string   `sql:"name"`
	Address  *string   `sql:"address"`
	City     *string   `sql:"city"`
	State    *string   `sql:"state"`
	Postal   *string   `sql:"postal"`
	Country  *string   `sql:"country"`
	Phone    *string   `sql:"phone"`
	Web      *string   `sql:"web"`
	USR      *int64    `sql:"usr"  audit:"user"`
	Modified time.Time `sql:"ts" audit:"time"`
}

type pxeDevice struct {
	DID        int64   `sql:"did" key:"true" table:"pxedevice"`
	STI        int64   `sql:"sti"`
	RID        int64   `sql:"rid"`
	Site       *string `sql:"site"`
	Rack       int     `sql:"rack"`
	RU         int     `sql:"ru"`
	Hostname   *string `sql:"hostname"`
	Profile    *string `sql:"profile"`
	MAC        *string `sql:"mac"`
	IP         *string `sql:"ip"`
	IPMI       *string `sql:"ipmi"`
	Note       *string `sql:"note"`
	Restricted bool    `sql:"restricted"`
}

type pxeRequest struct {
	Site   string
	Image  string
	Device pxeDevice
}

type audit struct {
	AID      int64     `sql:"aid" key:"true" table:"audit_view"`
	USR      int64     `sql:"usr"`
	STI      int64     `sql:"sti"`
	Site     *string   `sql:"site"`
	Hostname *string   `sql:"hostname"`
	Log      *string   `sql:"log"`
	User     *string   `sql:"user"`
	TS       time.Time `sql:"ts" audit:"time"`
}

type pxeHost struct {
	ID       int64   `sql:"id" key:"true" table:"pxehosts"`
	Sitename *string `sql:"sitename"`
	Hostname *string `sql:"hostname"`
}

/*
type event struct {
	//TS   jsonTime  `json:"ts"   sql:"TS" table:"events"`
	TS   time.Time  `json:"ts"   sql:"TS" table:"events"`
	Host string    `json:"host" sql:"Host"`
	Kind string    `json:"kind" sql:"Kind"` // dhcp, tftp, http
	Msg  string    `json:"msg"  sql:"Msg"`
}
*/

type event struct {
	TS   time.Time `sql:"TS" table:"events"`
	Host string    `sql:"Host"`
	Kind string    `sql:"Kind"` // dhcp, tftp, http
	Msg  string    `sql:"Msg"`
}

type credentials struct {
	IP       string `sql:"ip" table:"credentials"`
	Username string `sql:"username"`
	Password string `sql:"password"`
}

type tsTest struct {
	ID       int64   `sql:"id" key:"true" table:"tstest"`
	Host string    `sql:"Hostname"`
	Msg  string    `sql:"msg"`
	TS   int64      `sql:"ts"`
}

/*
CREATE TABLE IF NOT EXISTS "tstest" (
    id integer primary key,
    hostname text,
    msg text,
    ts integer DEFAULT CURRENT_TIMESTAMP
*/
