function WebSocketInit() {
	if ("WebSocket" in window) {
		var ws = new WebSocket("ws://" + document.location.host + "/ws");

		ws.onmessage = function(evt) { 
			connectionOK();
	    	newLiveData(evt.data);
	    };
	    
	    ws.onclose = function(){
	    	connectionLost()
	        setTimeout(function(){WebSocketInit()}, 5000);
	    };
	    	
		ws.onopen = function() {
			connectionOK();
		};
	}
	else {
		alert("WebSocket NOT supported by your Browser!");
	}
}
