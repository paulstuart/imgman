'use strict';

var toISOString = function(ts) {
    function pad(number) {
      return (number < 10) ? '0' + number : number;
    }
    return ts.getUTCFullYear() +
        '-' + pad(ts.getUTCMonth() + 1) +
        '-' + pad(ts.getUTCDate()) +
        'T' + pad(ts.getUTCHours()) +
        ':' + pad(ts.getUTCMinutes()) +
        ':' + pad(ts.getUTCSeconds()) +
        '.' + (ts.getUTCMilliseconds() / 1000).toFixed(3).slice(2, 5) +
        'Z';
}


if (!Date.prototype.toISOString) {
  (function() {

    function pad(number) {
      if (number < 10) {
        return '0' + number;
      }
      return number;
    }

    Date.prototype.toISOString = function() {
      return this.getUTCFullYear() +
        '-' + pad(this.getUTCMonth() + 1) +
        '-' + pad(this.getUTCDate()) +
        'T' + pad(this.getUTCHours()) +
        ':' + pad(this.getUTCMinutes()) +
        ':' + pad(this.getUTCSeconds()) +
        '.' + (this.getUTCMilliseconds() / 1000).toFixed(3).slice(2, 5) +
        'Z';
    };

  }());
}

var pingURL = "http://10.100.182.16:8080/dcman/api/pings?debug=true";
var dcman = 'http://localhost:8080/dcman'

var sitesURL        = "/api/site/" ; 
var userURL         = "/api/user/" ; 

var urlDebug = false;

var userInfo = {};

var mySTI = 1;

var MACs = {}

function isNumeric(n) {
  return !isNaN(parseFloat(n)) && isFinite(n);
}


var fromCookie = function() {
    // reload existing auth from cookie
    for (var cookie of document.cookie.split("; ")) {
        const tuple = cookie.split("=")
        if (tuple[0] == "userinfo") {
            if (tuple[1].length > 0) {
                return JSON.parse(atob(tuple[1]));
            }
        }
    }
    return null
}

var killCookie = function() {
    var xhttp = new XMLHttpRequest();
    xhttp.open("GET", "api/logout", true);
    xhttp.send();
}

const store = new Vuex.Store({
    state: {
        apiKey: "",
        level: 0,
        active: false,
        login: "",
        USR: null,
        hosts: {"": ""},
	updates: [],
    },
    getters: {
        canEdit: state => {
            return (state.level > 0)
        },
        isAdmin: state => {
            return (state.level > 1)
        },
        userName: state => {
            return state.login
        },
        apiKey: state => {
            return state.apiKey
        },
        loggedIn: state => {
            return state.active
        },
        updates: state => {
            return state.updates
        },
        hostlist: state => {
            return state.hosts 
        },
    },
    mutations: {
        setUser(state, user) {
            state.level = user.Level
            state.apiKey = user.APIKey
            state.login = user.Login
            state.USR = user.USR
            state.active = true
            if (user["COOKIE"] === true) {
                get("api/check")
                    .catch(x => {
                        console.log("user is bad:", x)
                    })
            }
        },
        logOut(state) {
            console.log("logging out:", state.login);
            state.level = 0
            state.apiKey = ""
            state.login = ""
            state.active = false
            state.USR = null
        },
        addUpdate(state, update) {
            state.updates.unshift(update)
            state.hosts[update.Host] = update.Host
        },
    },
})

const authVue = {
    computed: {
        canEdit: function() {
            return this.$store.getters.canEdit
        },
    },
    created: function() {
        // needed if refreshing a page and session expired
        if (! this.$store.getters.loggedIn) {
            console.log("NOT LOGGED IN!");
            router.push("/auth/login")
        }
    },
}

var apikey = function() {
    if (userInfo && userInfo.APIKey && userInfo.APIKey.length > 0) {
        return userInfo.APIKey
    }
    return ""
}


var receiver = function(obj) {
    console.log("RECV:", obj)
    store.commit("addUpdate", obj)
}

var sockinit = function(url, receiver) {
    try {
        var sock = new WebSocket(url);
        //sock.binaryType = 'blob'; // can set it to 'blob' or 'arraybuffer 
        console.log("Websocket - status: " + sock.readyState);
        sock.onopen = function(m) { 
            console.log("CONNECTION opened..." + this.readyState);
        }   
        sock.onmessage = function(m) { 
            receiver(JSON.parse(m.data))
        }
        sock.onerror = function(m) {
            console.log("Error occured sending..." + m.data);
        }
        sock.onclose = function(m) { 
            console.log("Disconnected - status " + this.readyState);
        }
        // bind sock to inner send func
        return function(obj) {
            sock.send(JSON.stringify(obj));
        }
    } catch(exception) {
        console.log(exception);
    }
};

const WS = "ws://" + window.location.hostname + ":" + window.location.port + "/sock";
console.log("WS:", WS)

var sender = sockinit(WS, receiver);

var get = function(url) {
  return new Promise(function(resolve, reject) {
    if (urlDebug) {
        url += ((url.indexOf("?") > 0) ? "&" : "?") + "debug=true"
    }
    // Do the usual XHR stuff
    var req = new XMLHttpRequest();
    req.open("GET", url);
    var key = store.getters.apiKey;
    if (key && key.length > 0) {
        req.setRequestHeader("X-API-KEY", key)
    } else {
        router.push("/auth/login")
        //reject(Error("no api key set"));
        return
        //console.log("no api key set")
    }

    req.onload = function() {
      // This is called even on 404 etc, so check the status
      if (req.status == 200) {
        // Resolve the promise with the response text
        var obj = (req.responseText.length > 0) ? JSON.parse(req.responseText) : null;
        resolve(obj)
      }
      else {
          if (req.status == 401) {
              store.commit("logOut")
              //  resolve(null)
              reject(Error(req.statusText));
                killCookie();
               router.go("/auth/login")
              return
          }
          // Otherwise reject with the status text
          // which will hopefully be a meaningful error
          console.log("rejecting!!! ack:",req.status, "txt:", req.statusText)
          reject(Error(req.statusText));
      }
    };

    // Handle network errors
    req.onerror = function() {
      console.log("get network error");
      reject(Error("Network Error"));
    };

    // Make the request
    req.send();
  });
}

var posty = function(url, data, method) {
    return new Promise(function(resolve, reject) {
    // Do the usual XHR stuff
    if (typeof method == "undefined") method = 'POST';
    var req = new XMLHttpRequest();
    if (urlDebug) {
        url += (url.indexOf('?') > 0) ? '&debug=true' : '?debug=true'
    }
    req.open(method, url);
    const key = store.getters.apiKey;
    if (key.length > 0) {
        req.setRequestHeader("X-API-KEY", key)
    }
    req.setRequestHeader("Content-Type", "application/json")

    req.onload = function() {
        // This is called even on 404 etc
        // so check the status
        //console.log('get status:', req.status, 'txt:', req.statusText)
        if (req.status >= 200 && req.status < 300) {
            if (req.responseText.length > 0) {
                resolve(JSON.parse(req.responseText))
            } else {
                resolve(null)
            }
        }
        else {
            // Otherwise reject with the status text
            // which will hopefully be a meaningful error
            console.log('rejecting!!! ack:',req.status, 'txt:', req.statusText)
            if (req.getResponseHeader("Content-Type") === "application/json") {
                reject(JSON.parse(req.responseText));
            } else {
                reject(Error(req.statusText));
            }
        }
    };

    // Handle network errors
    req.onerror = function() {
        console.log('posty network error');
        reject(Error("Network Error"));
    };

    // Make the request
    req.send(JSON.stringify(data));
  });
}

var getIt = function(url, what) {
    return function(id, query) {
        if (query) {
            if (id > 0) {
                url += query + id
            } else {
                url += query
            }
        } else if (id > 0) {
            url += id
        }
        return get(url).then(function(result) {
            console.log('fetched:', what)
            return result;
        })
        .catch(function(x) {
            console.log('fetch failed for:', what, 'because:', x);
        });
    }
}

var getAuditLog = getIt('/api/audit', 'audit');

var pagedCommon = {
    data: function() {
        return {
            rows: [],
            columns: [],
            searchQuery: '',
            startRow: 0,
            pagerows: 25,
            sizes: [10, 25, 50, 100, 'all'],
        }
    },
    computed: {
        rowsPerPage: function() {
            if (this.pagerows == 'all') return null;
            return parseInt(this.pagerows);
        },
        filteredRows: function() {
            return this.searchData(this.rows)
        },
    },
    methods: {
        resetStartRow: function() {
            this.startRow = 0;
        },
        searchData: function(data) {
            if (this.searchQuery.length == 0) {
                return data
            }
            return data.filter(obj => {
                for (var k of this.columns) {
                    const value = obj[k];
                    if (_.isString(value) && value.indexOf(this.searchQuery) >= 0) {
                        return true
                    }
                }
                return false
            })
        },
    },
}


var editVue = {
    methods: {
        saveSelf: function() {
            var data = this.myself()
            var id = this.myID()
            if (id > 0) {
                postIt(this.dataURL + id + "?debug=true", data, this.showList, 'PATCH')
            } else {
                postIt(this.dataURL + id + "?debug=true", data, this.showList)
            }
        },
        deleteSelf: function(event) {
            console.log('delete event: ' + event)
            postIt(this.dataURL + this.myID(), null, this.showList, 'DELETE')
        },
        showList: function(ev) {
            router.go(this.listURL)
        },
    }
}


// TODO: these should be generated from a factory function
function getSiteLIST(all) {
    return get(sitesURL).then(function(result) {
        console.log('sitelist fetched:', result.length);
        if (all) {
            result.unshift({STI:0, Name:'All Sites'})
        }
        return result;
    })
    .catch(function(x) {
      console.log('Could not load sitelist: ', x);
    });
}


function remember() {
    var cookies = document.cookie.split("; ");
    for (var i=0; i < cookies.length; i++) {
        var tuple = cookies[i].split('=')
        if (tuple[0] === 'X-API-KEY') {
            // all changeable actions require this key
            window.user_apikey = tuple[1]; 
            break
        } 
    }
}

remember();

Vue.component('main-menu', {
    template: '#tmpl-main-menu',
    props: ['app', 'msg'],
    data: function() {
       return {
           searchText: '',
       }
    },
    created: function() {
        this.userinfo()
    },
    methods: {
        'doSearch': function(ev) {
            var text = cleanText(this.searchText);
            if (text.length > 0) {
                console.log('initiate search for:',text)
                if (this.$route.name == 'search') {
                    // already on search page
                    return
                }
                router.go({name: 'search', params: { searchText: text }})
            }
        },
        'userinfo': function() {
            var cookies = document.cookie.split("; ");
            for (var i=0; i < cookies.length; i++) {
                var tuple = cookies[i].split('=')
                if (tuple[0] != 'userinfo') continue;
                if (tuple[1].length == 0) break; // no cookie value so don't bother
                var user = JSON.parse(atob(tuple[1]));
                break
            }
        },
    }
})

//
// Audit Log
//

var auditLog = Vue.component('audit-log', {
    template: '#tmpl-audit-log',
    mixins: [pagedCommon],
    data: function() {
        return {
            columns: ['TS', 'Site', 'Hostname', 'Log', 'User'],
            rows: [],
	    url: "api/audit/",
        }
    },
    created: function() {
	this.loadData()
    },
    methods: {
        loadData: function() {
            get(this.url).then(data => {
                if (data) {
                    this.rows = data
                    console.log("loaded", data.length, "ip records")
                }
            })
        },
        linkable: function(key) {
            //return (key == 'Login')
        },
        linkpath: function(entry, key) {
            //return '/user/edit/' + entry['USR']
        }
    }
})

//
// User List
//

var userList = Vue.component('user-list', {
    template: '#tmpl-user-list',
    mixins: [pagedCommon],
    data: function() {
        return {
            columns: ['Email', 'First', 'Last', 'Level'],
            rows: [],
            url: userURL,
        }
    },
    created: function() {
        this.loadData()
    },
    methods: {
        loadData: function() {
            get(this.url).then(data => this.rows = data)
        },
        addUser: function() {
            router.push('/user/edit/0')
        },
        linkable: function(key) {
            return (key == 'Email')
        },
        linkpath: function(entry, key) {
            return '/user/edit/' + entry['USR']
        }
    }
})

//
// USER EDIT
//

var userEdit = Vue.component('user-edit', {
    template: '#tmpl-user-edit',
    data: function() {
        return {
            User: {},
            listURL: '/user/list',
		url: "api/user/",
            levels: [
                {Level:0, Label: 'User'},
                {Level:1, Label: 'Editor'},
                {Level:2, Label: 'Admin'},
            ],
        }
    },
    created: function() {
        this.loadSelf()
    },
    methods: {
        loadSelf: function () {
            var id = this.$route.params.USR;
            if (id > 0) {
                var url = this.dataURL + id;

                get(this.url + id).then(u => this.User = u)
            } else {
                this.User = {
                    USR: null,
                    Email: "",
                    First: "",
                    Last: "",
                    Level: null,
                }
            }
        },
        showList: function() {
            router.push("/user/list")
        },
        saveSelf: function() {
            if (this.User.USR > 0) {
                    posty(this.url + this.User.USR, this.User, "PATCH").then(this.showList)
            } else {
                    posty(this.url + this.User.USR, this.User).then(this.showList)
            }
        },
        deleteSelf: function() {
            posty(this.url + this.User.USR, null, "DELETE").then(this.showList)
        },
    },
})


// Base APP component, this is the root of the app
var App = Vue.extend({
    data: function(){
        return {
            myapp: {
                auth: {
                    loggedIn: false,
                    user: {
                        name: null, 
                        admin: 0,
                    }
                },
            },
        }
    },
    methods: {
        fresher: function(ev) {
            console.log("the fresh maker!")
            this.$broadcast('ip-reload', 'please')
        },
    },
    events: {
        'user-info': function (user) {
            this.myapp.auth.user.name = user.username;
            this.myapp.auth.loggedIn = true;
        },
        'user-auth': function (user) {
            console.log('*** user auth event:', user)
            this.myapp.auth.user.name = user.Login;
            this.myapp.auth.user.admin = user.Level;
            window.user_apikey = user.APIKey
            userInfo = user;
            this.myapp.auth.loggedIn = true;
        },
        'logged-out': function () {
            console.log('*** logged out event')
            this.myapp.auth.user.name = null
            this.myapp.auth.user.admin = 0
            this.myapp.auth.loggedIn = false
            window.user_apikey = ''
            get('api/logout')
        },
    },
})



var userLogin = Vue.component('user-login', {
    template: '#tmpl-user-login',
    data: function() {
        return {
            username: '',
            password: '',
            placeholder: 'first.last@pubmatic.com',
            errorMsg: ''
        }
    },
    methods: {
        cancel: function() {
            router.go('/')
        },
        login: function(ev) {
            var data = {Username: this.username, Password: this.password};
            var url = '/api/login';
            posty(url, data).then(user => {
                this.$store.commit("setUser", user)
                router.push("/")
            }).catch(msg => this.errorMsg = msg.Error)
	},
    },
})


var userLogout = Vue.component('user-logout', {
    template: '#tmpl-user-logout',
    methods: {
        cancel: function() {
            router.go('/')
        },
        logout: function(ev) {
            console.log("logout button selected")
            this.$store.commit("logOut")
            router.push("/auth/login")
        },
    }
})


var pagedGrid = Vue.component("paged-grid", {
    template: "#tmpl-paged-grid",
    props: {
        data: Array,
        columns: Array,
        linkable: Function,
        linkpath: Function,
        startRow: Number,
        rowsPerPage: Number,
        filename: String,
    },
    data: function() {
        var sortOrders = {}
        if (this.columns) {
            this.columns.forEach(function (key) {
                sortOrders[key] = 1
            })
        }
        return {
              sortKey: "",
              sortOrders: sortOrders,
              currentRow: this.startRow
        }
    },
    computed: {
        rowStatus: function() {
            if (! this.rowsPerPage) {
                return this.data.length + ((this.data.length === 1) ? " row" : " rows")
            }
            var status =
                " Page " +
                (this.currentRow / this.rowsPerPage + 1) +
                " / " +
                (Math.ceil(this.data.length / this.rowsPerPage));

            if (this.data.length >  this.rowsPerPage) {
                status += " (" + this.data.length + " rows) ";
            }
            return status
        },
        canDownload: function() {
            return (this.data && (this.data.length > 0) && this.filename && (this.filename.length > 0))
        },
        limitBy: function() {
           var data = (this.rowsPerPage > 0) ? this.data.slice(this.currentRow, this.currentRow + this.rowsPerPage) : this.data;
           var orderBy = (this.sortOrders[this.sortKey] > 0) ? "asc" : "desc";
           return (this.sortKey.length > 0) ? _.orderBy(data, this.sortKey, orderBy) : data
        }
    },
    methods: {
        sortBy: function (column) {
            console.log("sort by:",column)
            this.sortKey = column
            this.sortOrders[column] = this.sortOrders[column] * -1
        },
        movePages: function(amount) {
            var row = this.currentRow + (amount * this.rowsPerPage);
            if (row >= 0 && row < this.data.length) {
                this.currentRow = row;
            }
        },
        download() {
            // TODO: perhaps get fancier and use this?
            // https://github.com/eligrey/FileSaver.js#saving-text
            var filename = this.filename;
            if (filename.indexOf(".") < 0 ) {
                filename += ".xls";
            }

            // gather up our data to save (tab delimited)
            var text = this.columns.join("\t") + "\n";
            for (var i=0; i < this.data.length; i++) {
                var line = [];
                for (var j=0; j < this.columns.length; j++) {
                    var col = this.columns[j];
                    line.push(this.data[i][col])
                }
                text += line.join("\t") + "\n";
            }

            var element = document.createElement("a");
            var ctype = "application/vnd.ms-excel";
            element.setAttribute("href", "data:" + ctype + ";charset=utf-8," + encodeURIComponent(text));
            element.setAttribute("download", filename);

            element.style.display = "none";
            document.body.appendChild(element);

            element.click();

            document.body.removeChild(element);
        },
    }
});


var pagedSlices = Vue.component("paged-slices", {
    template: "#tmpl-paged-slices",
    props: {
        data: Array,
        columns: Array,
        linkable: Function,
        linkpath: Function,
        startRow: Number,
        rowsPerPage: Number,
        filename: String,
    },
    data: function() {
        var sortOrders = {}
        if (this.columns) {
            this.columns.forEach(function (key) {
                sortOrders[key] = 1
            })
        }
        return {
              sortKey: "",
              sortOrders: sortOrders,
              currentRow: this.startRow
        }
    },
    computed: {
        rowStatus: function() {
            if (! this.rowsPerPage) {
                return this.data.length + ((this.data.length === 1) ? " row" : " rows")
            }
            var status =
                " Page " +
                (this.currentRow / this.rowsPerPage + 1) +
                " / " +
                (Math.ceil(this.data.length / this.rowsPerPage));

            if (this.data.length >  this.rowsPerPage) {
                status += " (" + this.data.length + " rows) ";
            }
            return status
        },
        canDownload: function() {
            return (this.data && (this.data.length > 0) && this.filename && (this.filename.length > 0))
        },
        limitBy: function() {
           var data = (this.rowsPerPage > 0) ? this.data.slice(this.currentRow, this.currentRow + this.rowsPerPage) : this.data;
           var orderBy = (this.sortOrders[this.sortKey] > 0) ? "asc" : "desc";
           return (this.sortKey.length > 0) ? _.orderBy(data, this.sortKey, orderBy) : data
        }
    },
    methods: {
        sortBy: function (column) {
            console.log("sort by:",column)
            this.sortKey = column
            this.sortOrders[column] = this.sortOrders[column] * -1
        },
        movePages: function(amount) {
            var row = this.currentRow + (amount * this.rowsPerPage);
            if (row >= 0 && row < this.data.length) {
                this.currentRow = row;
            }
        },
        download() {
            // TODO: perhaps get fancier and use this?
            // https://github.com/eligrey/FileSaver.js#saving-text
            var filename = this.filename;
            if (filename.indexOf(".") < 0 ) {
                filename += ".xls";
            }

            // gather up our data to save (tab delimited)
            var text = this.columns.join("\t") + "\n";
            for (var i=0; i < this.data.length; i++) {
                var line = [];
                for (var j=0; j < this.columns.length; j++) {
                    var col = this.columns[j];
                    line.push(this.data[i][col])
                }
                text += line.join("\t") + "\n";
            }

            var element = document.createElement("a");
            var ctype = "application/vnd.ms-excel";
            element.setAttribute("href", "data:" + ctype + ";charset=utf-8," + encodeURIComponent(text));
            element.setAttribute("download", filename);

            element.style.display = "none";
            document.body.appendChild(element);

            element.click();

            document.body.removeChild(element);
        },
    }
});


var imagePage = Vue.component('image-page', {
    template: '#tmpl-image-page',
    data: function() {
        return {
            sites: [],
            fields: 'STI DID Hostname Rack RU MAC IP IPMI Note'.split(' '),
            menu: '',
            menus: [],
            Device: {
                STI: 0,
                DID: 0,
                Hostname: '',
                Rack: 0,
                RU: 0,
                MAC: '',
                IP: '',
                IPMI: '',
                Note: '',
                Restricted: false,
            },
            STI: 0,
            ErrorMsg: ''
        }
    },
    created: function() {
        getSiteLIST().then(s => this.sites = s);
    },
    computed: {
        notReady: function() {
            return (this.Device.MAC.length == 0 || this.Device.IP.length == 0 || this.Device.IPMI.length == 0 || this.menu.length == 0)
        }
    },
    methods: {
        reset: function() {
            this.Device.DID     = 0
            this.Device.Rack    = ''
            this.Device.RU      = ''
            this.Device.MAC     = ''
            this.Device.IP      = ''
            this.Device.IPMI    = ''
            this.Device.Profile = ''
            this.Device.Note    = ''
            this.Device.Restricted  = false
        },
        reimage: function() {
            var url = '/api/pxeboot';
            const site = this.sites.find(s => s.STI == this.STI);
            const data = {
                Site: site['Name'],
                Image: this.menu,
                Device: this.Device,
            }
            posty(url, data).then(resp => {
                const obj = {
                    TS: toISOString(new Date()),   
                    Host: this.Device.Hostname,
                    Kind: "PXE boot initiated",
                    Msg: "image: " + this.menu,
                }
                store.commit("addUpdate", obj)
                    console.log("RESP:",resp)
                    this.Device.Note = resp.Note
                    router.push("/")
                }).catch(function(oops) {
                    console.log('oops:',oops);
                    this.ErrorMsg = oops['Error']
            })
        },
        loadSelf: function() {
            var url = '/api/host/';
            var data = {
                Hostname: this.Device.Hostname,
                STI: this.STI,
            }
            posty(url, data).then(device => {
                if (device.length == 1) {
                    this.Device = device[0]
                } else {
                    //throw(Error("wrong number of devices"));
                    console.log("wrong number of devices");
                }
            }).catch(oops => {
                this.reset()
            })
        },
        home: function() {
                router.push("/")
        },
    },
    watch: {
        "STI": function() {
            this.reset()
            this.Device.Hostname = ''
            this.menu           = ''
            this.menus          = []
            for (var i=0; i < this.sites.length; i++) {
                var site = this.sites[i];
                if (site.STI == this.STI) {
                    console.log("site:",site.Name)
                    get('/api/menus/' + site.Name).then(list => {
                        list.unshift('')
                        this.menus = list;
                    })
                    break;
                }
            }
        },
    }
})

//
// Site List
//

var pxeHosts = Vue.component('pxe-hosts', {
    template: '#tmpl-pxe-hosts',
    mixins: [pagedCommon],
    data: function() {
        return {
            columns: ["Sitename", "Hostname"],
            url: "api/pxehost/",
        }
    },
    created: function() {
        this.loadData()
    },
    methods: {
        loadData: function() {
            get(this.url).then(data => {
                if (data) {
                    this.rows = data
                    console.log("loaded", data.length, "sites")
                }
            })
        },
	addSite: function() {
		router.push("/site/edit/0")
	},
        linkable: function(key) {
            return (key == 'Sitename')
        },
        linkpath: function(entry, key) {
            return '/site/edit/' + entry['ID']
        }
    }
})

var siteEdit = Vue.component('pxe-edit', {
    template: '#tmpl-pxe-edit',
    data: function() {
        return {
            Site: {},
            url: 'api/pxehost/',
        }
    },
    created: function () {
	this.loadSelf()
    },
    methods: {
        loadSelf: function () {
            var id = this.$route.params.ID;
            if (id > 0) {
                get(this.url + id).then(s => this.Site = s)
            }
        },
	showList: function() {
		router.push("/site/list")
	},
	saveSelf: function() {
                if (this.Site.ID > 0) {
                        posty(this.url + this.Site.ID, this.Site, "PATCH").then(this.showList)
                } else {
                        posty(this.url + this.Site.ID, this.Site).then(this.showList)
                }
        },
    },

})

var homePage = Vue.component('home-page', {
    template: '#tmpl-home-page',
    mixins: [pagedCommon],
    data: function() {
        return {
            columns: ['TS', 'Host', 'Kind', 'Msg'],
            url: "api/site/",
            hostfilter: "",
        }
    },
    created: function() {
        this.loadData()
    },
    computed: {
	    eventrows: function() {
            if (this.hostfilter.length > 0) {
                return this.searchData(this.$store.getters.updates).filter(u => u.Host == this.hostfilter);
            }
            return this.searchData(this.$store.getters.updates);
	    },
        hostlist: function() {
            return this.$store.getters.hostlist
        },
    },
    methods: {
        loadData: function() {
	    get("api/events").then(events => {
            for (let e of events) {
                this.$store.commit("addUpdate", e)
            }
	    })
        },
        linkable: function(key) {
            return (key == 'Login')
        },
        linkpath: function(entry, key) {
            return '/user/edit/' + entry['USR']
        }
    }
})
// Assign the new router
//var router = new VueRouter({history: true})

const routes = [
{ path: "/auth/login",  	component: userLogin },
{ path: '/audit/log', 		component: auditLog },
{ path: '/auth/login',  	component: userLogin },
{ path: '/auth/logout', 	component: userLogout },
{ path: '/site/edit/:ID', 	component: siteEdit },
{ path: '/site/list', 		component: pxeHosts },
{ path: '/user/edit/:USR',	component: userEdit },
{ path: '/user/list', 		component: userList },
{ path: '/image', 		component: imagePage },
{ path: '/', 			component: homePage },
]

const router = new VueRouter({
   routes // short for routes: routes
})

// load user info from cookie if it exists
const checkUser = fromCookie();
if (checkUser) {
    checkUser['COOKIE'] = true;
    store.commit("setUser", checkUser)
    //console.log("user is set");
}

//store.commit("addUpdate", {TS: "now", Host: "locohost", Kind: "what ev er", Msg: "this is fake"})

var app = new Vue({
    router,
    store
}).$mount("#myapp")

router.beforeEach((to, from, next) => {
    if (store.getters.loggedIn || to.path == "/auth/login") {
        next()
    } else {
        next("/auth/login")
    }
})


