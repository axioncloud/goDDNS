const net = require('node:net');

var win;
var tray;

function showWindow() {
    if (win == null) {
        var nw = require('nw.gui');
        win = nw.Window.get();
    }
    win.show();
    win.on('reload', function () { tray.remove(); tray = null; })
}
function showTray() {
    if (tray != null) {
        return;
    }
    var nw = require('nw.gui');
    tray = new nw.Tray({ title: 'goDDNS', icon: './goddns.png', tooltip: "goDDNS" });

    tray.on("click", showWindow)
    // Give it a menu
    var menu = new nw.Menu();
    menu.append(new nw.MenuItem({ type: 'normal', label: 'Open goDDNS', click: function () {
        showWindow()
    } }));
    menu.append(new nw.MenuItem({ type: 'normal', label: 'Exit', click: function () {
        exit()
    } }));
    tray.menu = menu;
    
    var startingNotification = new window.Notification("goDDNS", {body: "goDDNS started...", icon: "./goddns.ico", badge: "./goddns.ico"});
    startingNotification.addEventListener("click", (event) => {startingNotification.close(); showWindow();});
    window.onbeforeunload = (function (){
        if (win != null) {
            win.hide()
        }
        removeTray();
    });
}
function removeTray() {
    tray.remove();
    tray = null;
}

function minimize() {
    console.log("minimizing window");
    var gui = require('nw.gui');
    var win = gui.Window.get();
    win.minimize();
}
function exit() {
    console.log("Closing window");
    var gui = require('nw.gui');
    var win = gui.Window.get();
    removeTray();
    win.close();
}
function btn_danger() {
    $("#btn-exit").addClass("btn-danger").removeClass("btn-secondary")
}
function btn_normal() {
    $("#btn-exit").addClass("btn-secondary").removeClass("btn-danger")
}
function submitProvider() {
    console.debug("Submitting form")
    goDDNS.POST("/providers", $('#providerForm').serialize())
    return true;
}


var PIPE_PATH = "\\\\.\\pipe\\goddns";

var log = console.log;

const goDDNS = {
    response: {
        DATA: null,
        JSON: null,
        STATUS: null,
        HEADER: null,
    },
    responseJSON: null,
    responseStatus: null,
    responseHeader: null,
    
    clear() {
        Object.keys(this.response).forEach(key => {this.response[key] = null;})
    },
    parseResponse() {
        var resp = this.response.DATA
        var arr = resp.split("\r\n\r\n", 2);
        this.response.HEADER = arr[0];
        if (arr[arr.length - 1].length == 0) {
            this.response.JSON = "";
        } else {
            this.response.JSON = JSON.parse(arr[arr.length - 1]);
        }
        
        this.response.STATUS = this.response.HEADER.substring(0, this.response.HEADER.indexOf("\r\n")).match(/([0-9])\w+/g)[0];
    },
    /**
     * @param {string} location - location of GET request
     * @param {function} _callback - callback function for after data has been received and processed. Function is passed in the JSON received
     */
    GET(location, _callback) {
        setTimeout(() => {
            var client = net.connect(PIPE_PATH, function() {
                log('client connected');
            }).on('data', function(data) {
                goDDNS.response.DATA = data.toString();
                goDDNS.parseResponse();
                client.end();
                if (_callback !== undefined) {
                    _callback(goDDNS.response.JSON)
                }
            }).on('end', function() {
                log('client disconnected');
            })
    
            client.write(`GET ${location} HTTP/1.0\r\n\r\n`);
        }, 0);

        return this.response
    },
    POST(location, form_data) {
        setTimeout(() => {
            var client = net.connect(PIPE_PATH, function() {
                log('client connected');
            }).on('data', function(data) {
                goDDNS.response.DATA = data.toString();
                goDDNS.parseResponse();
                client.end();
            }).on('end', function() {
                log('client disconnected');
            })
    
            client.write(`POST ${location} HTTP/1.0\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: ${form_data.length}\r\n\r\n${form_data}`);
        }, 0);

        return this.response
    },
    GET_PROVIDERS(_callback) {
        this.GET("/providers", _callback)
    },
    clearProvidersList() {
        $('#selectProvider').empty();
    },
    loadProvidersList() {
        this.GET_PROVIDERS((json) => {
            var providerSelect = $('#selectProvider');
            for (let key in json) {
                var provider = json[key]
                providerSelect.append(`<option id="${key}" ${provider.SELECTED === 1 ? 'selected' : ''}>${provider.NAME}</option>`)
            }
        })
    },
    loadHostname() {
        
    },
    loadUsername() {
        
    },
    loadPassword() {
        
    },
    refreshSettings() {
        
    }
};


