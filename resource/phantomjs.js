
function waitFor(testLoaded, onReady, timeOutMillis) {
    var start = new Date().getTime(),
        loaded = false,
        interval = setInterval(function() {
            if ( (new Date().getTime() - start < timeOutMillis) && !loaded ) {
                //如果尚未超时，判断等待条件
                // If not time-out yet and loaded not yet fulfilled
                loaded = testLoaded();
            } else {
                if(!loaded) {
                    //抵达这里说明尚未满足等待条件但是已经超时，依旧结束
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

//去除引号
var input = system.args[1].replace(/"/g, "");
//文字记录中，@前的为应该请求的url
var url = input.split('@')[0];
//@后的为应该生成的html的路径
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
            //等待页面中body元素出现pagerenderready的属性
            // Check in the page if a specific element is now visible
            return page.evaluate(function() {
                return $('body').attr('pagerenderready');
            });
        },
        function() {
           console.log("Now success");
           console.log(page.content);
           fs.write(dest, page.content, 'w');
           //最终调用phantom.exist保证phantom退出
           phantom.exit();
        }, 8000);  //最长等待八秒
        
        
    }
});