// TODO: generate from cookie data

// register the menu component
Vue.component('my-nav', {
  template: '#main-menu',
  props: ['app', 'hey', 'msg'],
})



menuVue = new Vue({
  el: '#main-nav',
  data: {
      hello: "my name is waldo",
      hey: "what's up, duuuuude?",
      msg: "secret message",
      myapp: {
        auth: {
            loggedIn: true,
            user: {
                name: "Waldo"
            }
        },
      },
    },
});


var ipMIX = {
    props: [ 'Kind', 'What'],
    methods: {
      columnar: function(obj, key) {
          var val = obj[key];
          switch (key) {
              case "Hostname":
                  return '<a href="static/html/server_edit.html?SID=' + obj.ID + '">' + val + '</a>'; 
              case "Bad":
                  return obj.Bad ? "Bad" : "";
          }
          return val;
      },
      myFilter: function(a, b, c) {
          if (! this.What) {
              return a
          }
          if (this.What == a.What) {
              return a
          }
      },
    },
}

newTable('demo-grid', '#custom-grid-template', [ipMIX])

NetworkView = new Vue({
  el: '#ip-list',
  data: {
      DCD: 1,
      RID: 0,
      dcs: [],
      iplist: [],
      What: '',
      whatlist: [
        '',
        'ipmi',
        'internal',
        'public',
        'vip',
      ],
      searchQuery: '',
      gridColumns: [
        "DC",
        "Kind",
        "What",
        "Hostname",
        "IP",
        "Note"
      ],
    }
  ,
  created: function () {
      this.loadDC()
  },

  methods: {
    loadIPs: function () {
         var self = this,
              url = networkURL + "?DCD=" + self.DCD;

         fetchData(url, function(data) {
             if (data) {
                 self.iplist = data
             }
         })
    },
    loadDC: function () {
         self = this

         fetchData(dcURL, function(data) {
             if (data) {
                 data.unshift({DCD:0, Name:' All '})
                 self.dcs = data
             }
             self.loadIPs()
         })

    },
  },

  watch: {
    'What': function(val, oldVal) {
          console.log("new what:" +  val);
     },
      /*
          //console.log("what was " + oldVal + " and is now " + val);
    */
    'DCD': function(val, oldVal){
            this.loadDC()
      },
    },

  mixins: [menuMIX],

});

// Define some components
var Foo = Vue.extend({
    template: '<p>This is foo!</p>'
})

var Bar = Vue.extend({
    template: '<p>This is bar!</p>'
})

// The router needs a root component to render.
// For demo purposes, we will just use an empty one
// because we are using the HTML as the app template.
// !! Note that the App is not a Vue instance.
var App = Vue.extend({})

// Create a router instance.
// You can pass in additional options here, but let's
// keep it simple for now.
var router = new VueRouter({root: '/dcman/static/html/index.html'})

// Define some routes.
// Each route should map to a component. The "component" can
// either be an actual component constructor created via
// Vue.extend(), or just a component options object.
// We'll talk about nested routes later.
router.map({
    '/foo': {
        component: Foo
    },
    '/bar': {
        component: Bar
    },
    '/network': {
        component: NetworkView
    }
})

// Now we can start the app!
// The router will create an instance of App and mount to
// the element matching the selector #app.
router.start(App, '#app')

