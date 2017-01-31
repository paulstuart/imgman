package main

import (
	"database/sql/driver"
	"fmt"
	"time"
)

//go:generate dbgen

type jsonDate time.Time

func (d jsonDate) MarshalJSON() ([]byte, error) {
	t := time.Time(d)
	if t.IsZero() {
		return []byte(`""`), nil
	}
	stamp := fmt.Sprintf(`"%s"`, t.Format("2006-01-02"))
	return []byte(stamp), nil
}

func (d *jsonDate) UnmarshalJSON(in []byte) error {
	s := string(in)
	fmt.Printf("\nPARSE THIS: (%d) %s\n\n", len(s), s)
	if len(in) < 3 {
		return nil
	}
	if d == nil {
		d = new(jsonDate)
	}
	const longform = `"2006-01-02T15:04:05.000Z"`
	if len(s) == len(longform) {
		t, err := time.Parse(longform, s)
		*d = jsonDate(t)
		return err
	}
	t, err := time.Parse(`"2006-1-2"`, s)
	if err != nil {
		t, err = time.Parse(`"2006/1/2"`, s)
	}
	if err == nil {
		*d = jsonDate(t)
	}
	return err
}

// Scan implements the Scanner interface.
func (d *jsonDate) Scan(value interface{}) error {
	*d = jsonDate(value.(time.Time))
	return nil
}

// Value implements the driver Valuer interface.
func (d *jsonDate) Value() (driver.Value, error) {
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
	DID      int64   `sql:"did" key:"true" table:"pxedevice"`
	STI      int64   `sql:"sti"`
	RID      int64   `sql:"rid"`
	Site     *string `sql:"site"`
	Rack     int     `sql:"rack"`
	RU       int     `sql:"ru"`
	Hostname *string `sql:"hostname"`
	Profile  *string `sql:"profile"`
	MAC      *string `sql:"mac"`
	IP       *string `sql:"ip"`
	IPMI     *string `sql:"ipmi"`
	Note     *string `sql:"note"`
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
	ID      int64   `sql:"id" key:"true" table:"pxehosts"`
	Sitename *string `sql:"sitename"`
	Hostname *string `sql:"hostname"`
}

type event struct {
	TS   time.Time `sql:"ts" table:"pxehosts"`
	Host string    `sql:"host"`
	Kind string    `sql:"kind"` // dhcp, tftp, http
	Msg  string    `sql:"msg"`
}
