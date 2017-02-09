package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	allSites    = []site{}
	currentSite site
)

func homePage(w http.ResponseWriter, r *http.Request) {
	name := filepath.Join(assetDir, "static/html/spa.html")
	file, err := os.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	fi, _ := file.Stat()
	w.Header().Set("Cache-control", "public, max-age=259200")
	cors(w)
	http.ServeContent(w, r, name, fi.ModTime(), file)
}

func authorizeUser(w http.ResponseWriter, yes bool) {
	c := &http.Cookie{
		Name: authCookie,
		Path: "/",
	}
	if yes {
		c.Expires = time.Now().Add(sessionMinutes)
		c.Value = oktaHash
	}
	http.SetCookie(w, c)
}

func remember(w http.ResponseWriter, u *user) {
	c := &http.Cookie{
		Name: cookieID,
		Path: "/",
	}
	if u != nil {
		c.Expires = time.Now().Add(sessionMinutes)
		c.Value = u.Cookie()
	}
	http.SetCookie(w, c)
	c = &http.Cookie{
		Name: "userinfo",
		Path: "/",
	}
	if u != nil {
		c.Expires = time.Now().Add(sessionMinutes)
		//c.Value = b64(fmt.Sprintf(`{"username": "%s", "admin": %d}`, u.login(), u.Level))
		b, err := json.Marshal(&u)
		if err != nil {
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		c.Value = b64(string(b))
	}
	http.SetCookie(w, c)
}

func ipmiMAC(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ipmi := r.URL.Path
	if len(ipmi) == 0 {
		notFound(w, r)
		return
	}
	mac, _ := findMAC(ipmi)
	d := struct {
		MacEth0 string
	}{
		MacEth0: mac,
	}
	j, _ := json.MarshalIndent(d, " ", " ")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, string(j))
}

func bulkPings(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	timeout := pingTimeout
	if text := r.Form.Get("timeout"); len(text) > 0 {
		if t, err := strconv.Atoi(text); err == nil {
			timeout = t
		}
	}
	if text := r.Form.Get("debug"); len(text) > 0 {
		if debug, err := strconv.ParseBool(text); err == nil && debug {
			for k, v := range r.Form {
				log.Println("K:", k, "(", len(k), ") V:", v)
			}
		}
	}
	if iplist, ok := r.Form["ips"]; ok && len(iplist) > 0 {
		ips := strings.Split(iplist[0], ",")
		pings := bulkPing(timeout, ips...)
		j, _ := json.MarshalIndent(pings, " ", " ")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, string(j))
	}
}

func ipmiCredentialsGet(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		ipmi := r.Form.Get("ipmi")
		username, password, err := getCredentials(ipmi)
		if err != nil {
			log.Println("error getting creds for ipmp:", ipmi, "error:", err)
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, username, password)
	}
}

func ipmiCredentialsSet(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		ipmi := r.Form.Get("ipmi")
		username := r.Form.Get("username")
		password := r.Form.Get("password")
		err := setCredentials(ipmi, username, password)
		if err != nil {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintln(w, err)
		}
	}
}

func apiPXE(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		pxe := new(pxeRequest)
		loadObj(r, pxe)
		fmt.Println("PXE BOOT REQUEST:", pxe)
		user, err := apiUser(r)
		if err != nil {
			log.Println("AUTH ERROR:", err)
			jsonError(w, err, http.StatusUnauthorized)
			return
		}
		log.Println("H:", *pxe.Device.Hostname, "M:", pxe.Image)
		msg := "pxeboot initiated"
		a := &audit{
			STI:      pxe.Device.STI,
			USR:      user.USR,
			Hostname: pxe.Device.Hostname,
			Log:      &msg,
			TS:       time.Now(),
		}
		fmt.Println("SET MAC HOST")
		setMacHost(*pxe.Device.MAC, *pxe.Device.Hostname)
		fmt.Println("MAC HOST SET")

		if err := dbAdd(a); err != nil {
			log.Println("DB ERROR:", err)
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		fmt.Println("PRE EXEC")
		if pxe.Device.Note != nil && len(*pxe.Device.Note) > 0 {
			note := *pxe.Device.Note + "\n" + msg
			pxe.Device.Note = &note
			//pxe.Note = *pxe.Note + "\n" + msg
		} else {
			pxe.Device.Note = &msg
		}
		//fmt.Println("GO EXEC")
        ipmiHostSave(pxe.Device)
		pxeExec(pxe.Site, *pxe.Device.Hostname, *pxe.Device.IPMI, pxe.Image)
		sendJSON(w, pxe.Device)
	}
}

func apiLogin(w http.ResponseWriter, r *http.Request) {
	method := strings.ToUpper(r.Method)
	switch method {
	case "POST":
		obj := &credentials{}
		content := r.Header.Get("Content-Type")
		if strings.Contains(content, "application/json") {
			if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
				fmt.Println("***** ERR:", err)
				jsonError(w, err, http.StatusInternalServerError)
				return
			}
		} else {
			objFromForm(obj, r.Form)
		}
		remoteAddr := remoteHost(r)
		log.Println("user:", obj.Username)
		user, err := userAuth(obj.Username, obj.Password)
		if err != nil {
			auditLog(0, remoteAddr, "Login", err.Error())
			jsonError(w, err, http.StatusUnauthorized)
			return
		}
		auditLog(user.USR, remoteAddr, "Login", "Login succeeded for "+obj.Username)
		cors(w)
		c := &http.Cookie{
			Name:    "X-API-KEY",
			Path:    "/",
			Expires: time.Now().Add(4 * time.Hour),
			Value:   user.apiKey(),
		}
		http.SetCookie(w, c)
		remember(w, user)
		sendJSON(w, user)
	default:
		jsonError(w, "invalid http method:"+r.Method, http.StatusUnauthorized)
	}
}

func apiLogout(w http.ResponseWriter, r *http.Request) {
	cors(w)
	c := &http.Cookie{
		Name:    "SAML",
		Path:    "/",
		Expires: time.Unix(0, 0),
	}
	http.SetCookie(w, c)
	c.Name = "dcuser"
	http.SetCookie(w, c)
	c.Name = "userinfo"
	http.SetCookie(w, c)
	c.Name = "redirect"
	http.SetCookie(w, c)
	c.Name = "X-API-KEY"
	http.SetCookie(w, c)
}

func siteList(w http.ResponseWriter, r *http.Request) {
	if sites, err := getSites(); err != nil {
		jsonError(w, err, http.StatusInternalServerError)
	} else {
		sendJSON(w, sites)
	}
}

func pxeList(w http.ResponseWriter, r *http.Request) {
	if hosts, err := dbList(&pxeHost{}); err != nil {
		jsonError(w, err, http.StatusInternalServerError)
	} else {
		sendJSON(w, hosts)
	}
}

func menuHandler(w http.ResponseWriter, r *http.Request) {
	site := r.URL.Path
	if menus, err := menuList(site); err != nil {
		fmt.Println("menu fetch error:", err)
		jsonError(w, err, http.StatusInternalServerError)
	} else {
		sendJSON(w, menus)
	}
}

func hostInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var host = struct {
			Hostname string
			STI      int64
		}{}
		if err := loadObj(r, &host); err != nil {
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		device, err := getHost(host.Hostname, host.STI)
		if err != nil {
			jsonError(w, err, http.StatusInternalServerError)
			return
		}
		sendJSON(w, device)
	}
}

// get user info for self
func apiCheck(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	apiKey := r.Header.Get("X-API-KEY")
	if len(apiKey) == 0 {
		apiKey = query.Get("X-API-KEY")
	}
	if len(apiKey) == 0 {
		jsonError(w, "missing API key", http.StatusBadRequest)
		return
	}
	u, err := userFromAPIKey(apiKey)
	if err != nil {
		jsonError(w, err, http.StatusUnauthorized)
		return
	}
	sendJSON(w, u)
}

func apiEvents(w http.ResponseWriter, r *http.Request) {
	/*
		if err := eventsJSON(w); err != nil {
			log.Println("events error:", err)
		}
	*/
	o := event{}
	q := fmt.Sprintf("select %s from %s", o.SelectFields(), o.TableName())
	fmt.Println("EVENTS QUERY:", q)
	dbStreamJSON(w, q)
}

func activeJSON(w http.ResponseWriter, r *http.Request) {
		sendJSON(w, activeList())
}

var webHandlers = []hFunc{
	{"/static/", StaticPage},
	{"/api/audit/", MakeREST(audit{})},
	{"/api/check", apiCheck},
	{"/api/events", apiEvents},
	{"/api/login", apiLogin},
	{"/api/logout", apiLogout},
	{"/api/mac/", ipmiMAC},
	{"/api/host/", hostInfo},
	{"/api/menus/", menuHandler},
	{"/api/site/", siteList},
	{"/api/pxehost/", MakeREST(pxeHost{})},
	{"/api/pings", bulkPings},
	{"/api/pxeboot", apiPXE},
	{"/api/user/", MakeREST(user{})},
	{"/active", activeJSON},
	{"/", homePage},
}
