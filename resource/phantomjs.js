
function waitFor(testLoaded, onReady, timeOutMillis) {
    var start = new Date().getTime(),
        loaded = false,
        interval = setInterval(function() {
            if ( (new Date().getTime() - start < timeOutMillis) && !loaded ) {
                // If not time-out yet and loaded not yet fulfilled
                loaded = testLoaded();
            } else {
                if(!loaded) {
                    console.log("'waitFor()' timeout");
                }
                
                // loaded fulfilled (timeout and/or loaded is 'true')
                console.log("'waitFor()' finished in " + (new Date().getTime() - start) + "ms.");
                onReady(); //< Do what it's supposed to do once the loaded is fulfilled
                clearInterval(interval); //< Stop this interval
                phantom.exit();
                return;
            }
        }, 250); //< repeat check every 250ms
}


var page = require('webpage').create();
var system = require('system');
var fs = require('fs');

var input = system.args[1].replace(/"/g, "");
var url = input.split('@')[0];
var dest = input.split('@')[1];

console.log("Phantom receving parameters: " + system.args[1]);

page.open(url, function (status) {
    var htmlcontent;
    // Check for page load success
    if (status !== "success") {
        console.log("Unable to access network");
        phantom.exit();
    } else {
        waitFor(function() {
            // Check in the page if a specific element is now visible
            return page.evaluate(function() {
                return $('body').attr('pagerenderready');
            });
        },
        function() {
           console.log("Now success");
           console.log(page.content);
           fs.write(dest, page.content, 'w');
           phantom.exit();
        }, 8000);  //Default Max Timout is 8s
        
        
    }
});